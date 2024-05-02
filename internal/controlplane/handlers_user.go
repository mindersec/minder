// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"path"
	"strconv"

	gauth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/projects"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// CreateUser is a service for user self registration
func (s *Server) CreateUser(ctx context.Context,
	_ *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {

	tokenString, err := gauth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "no auth token: %v", err)
	}

	token, err := s.vldtr.ParseAndValidate(tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to parse bearer token: %v", err)
	}

	tx, err := s.store.BeginTransaction()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction")
	}
	defer s.store.Rollback(tx)
	qtx := s.store.GetQuerierWithTransaction(tx)
	if qtx == nil {
		return nil, status.Errorf(codes.Internal, "failed to get transaction")
	}

	subject := token.Subject()

	user, err := qtx.CreateUser(ctx, subject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %s", err)
	}
	var userProjects []*db.Project

	userProjects = append(userProjects, s.claimGitHubInstalls(ctx, qtx)...)

	if len(userProjects) == 0 {
		// Set up the default project for the user
		baseName := subject
		if token.PreferredUsername() != "" {
			baseName = token.PreferredUsername()
		}

		project, err := s.projectCreator.ProvisionSelfEnrolledOAuthProject(
			ctx,
			qtx,
			baseName,
			subject,
		)
		if err != nil {
			if errors.Is(err, projects.ErrProjectAlreadyExists) {
				return nil, util.UserVisibleError(codes.AlreadyExists, "project named %s already exists", baseName)
			}
			return nil, status.Errorf(codes.Internal, "failed to create default organization records: %s", err)
		}
		userProjects = append(userProjects, project)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Project = userProjects[0].ID

	err = s.store.Commit(tx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %s", err)
	}

	if len(userProjects) == 0 {
		return nil, status.Errorf(codes.Internal, "failed to create any projects for user")
	}

	return &pb.CreateUserResponse{
		Id:              user.ID,
		ProjectId:       userProjects[0].ID.String(),
		ProjectName:     userProjects[0].Name,
		IdentitySubject: user.IdentitySubject,
		CreatedAt:       timestamppb.New(user.CreatedAt),
	}, nil
}

func (s *Server) claimGitHubInstalls(ctx context.Context, qtx db.Querier) []*db.Project {
	ghId, ok := auth.GetUserClaimFromContext[string](ctx, "gh_id")
	if !ok || ghId == "" {
		return nil
	}
	installs, err := qtx.GetUnclaimedInstallationsByUser(ctx, sql.NullString{String: ghId, Valid: true})
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Str("gh_id", ghId).Msg("failed to get unclaimed installations")
		return nil
	}

	userID, err := strconv.ParseInt(ghId, 10, 64)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Str("gh_id", ghId).Msg("failed to parse gh_id")
		return nil
	}

	var userProjects []*db.Project

	for _, i := range installs {
		// TODO: if we can get an GitHub auth token for the user, we can do the rest with CreateGitHubAppWithoutInvitation
		proj, err := s.ghProviders.CreateGitHubAppWithoutInvitation(ctx, qtx, userID, i.AppInstallationID)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Int64("org_id", i.OrganizationID).Msg("failed to create GitHub app at first login")
			continue
		}
		if proj != nil {
			userProjects = append(userProjects, proj)
		}
	}

	return userProjects
}

// DeleteUser is a service for user self deletion
func (s *Server) DeleteUser(ctx context.Context,
	_ *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {

	tokenString, err := gauth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "no auth token: %v", err)
	}

	token, err := s.vldtr.ParseAndValidate(tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to parse bearer token: %v", err)
	}

	subject := token.Subject()

	err = DeleteUser(ctx, s.store, s.authzClient, s.projectDeleter, subject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete user from database: %v", err)
	}

	resp, err := s.cfg.Identity.Server.Do(ctx, "DELETE", path.Join("admin/realms/stacklok/users", subject), nil, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete account on IdP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return nil, status.Errorf(codes.Internal, "unexpected status code when deleting account: %d", resp.StatusCode)
	}

	return &pb.DeleteUserResponse{}, nil
}

func (s *Server) getUserDependencies(ctx context.Context, user db.User) ([]*pb.Project, error) {
	// get all the projects associated with that user
	projs, err := s.authzClient.ProjectsForUser(ctx, user.IdentitySubject)
	if err != nil {
		return nil, err
	}

	var projectsPB []*pb.Project
	for _, proj := range projs {
		pinfo, err := s.store.GetProjectByID(ctx, proj)
		if err != nil {
			// if the project was deleted while iterating, skip it
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, err
		}

		projectsPB = append(projectsPB, &pb.Project{
			ProjectId: proj.String(),
			Name:      pinfo.Name,
			CreatedAt: timestamppb.New(pinfo.CreatedAt),
			UpdatedAt: timestamppb.New(pinfo.UpdatedAt),
		})
	}

	return projectsPB, nil
}

// GetUser is a service for getting personal user details
func (s *Server) GetUser(ctx context.Context, _ *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	// user is always authorized to get themselves
	tokenString, err := gauth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "no auth token: %v", err)
	}

	openIdToken, err := s.vldtr.ParseAndValidate(tokenString)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "failed to parse bearer token: %v", err)
	}

	// check if user exists
	user, err := s.store.GetUserBySubject(ctx, openIdToken.Subject())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %s", err)
	}

	var resp pb.GetUserResponse
	resp.User = &pb.UserRecord{
		Id:              user.ID,
		IdentitySubject: user.IdentitySubject,
		CreatedAt:       timestamppb.New(user.CreatedAt),
		UpdatedAt:       timestamppb.New(user.UpdatedAt),
	}

	projs, err := s.getUserDependencies(ctx, user)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get user dependencies: %s", err)
	}
	resp.Projects = projs

	return &resp, nil
}
