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
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"path"
	"strconv"

	"github.com/google/uuid"
	gauth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/auth/jwt"
	"github.com/stacklok/minder/internal/authz"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/flags"
	"github.com/stacklok/minder/internal/invite"
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

	token, err := s.jwt.ParseAndValidate(tokenString)
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
			// Check if `project_name_lower_idx` unique constraint was violated. This happens when
			// the project name is already taken. In this case, we will append a random string to the
			// project name. This is a temporary solution until we have a better way to handle this.
			baseName, err = getUniqueProjectBaseName(ctx, s.store, token.PreferredUsername())
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get unique project name: %s", err)
			}
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
	ghId, ok := jwt.GetUserClaimFromContext[string](ctx, "gh_id")
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

	token, err := s.jwt.ParseAndValidate(tokenString)
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

func (s *Server) getUserDependencies(ctx context.Context, user db.User) ([]*pb.ProjectRole, []*pb.Project, error) {
	// get all the projects associated with that user
	projs, err := s.authzClient.ProjectsForUser(ctx, user.IdentitySubject)
	if err != nil {
		return nil, nil, err
	}

	var projectRoles []*pb.ProjectRole
	var deprecatedPrjs []*pb.Project
	for _, proj := range projs {
		pinfo, err := s.store.GetProjectByID(ctx, proj)
		if err != nil {
			// if the project was deleted while iterating, skip it
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, nil, err
		}

		// Try to parse the project metadata to complete the response fields
		pDisplay := pinfo.Name
		pDescr := ""
		meta, err := projects.ParseMetadata(&pinfo)
		if err == nil {
			pDisplay = meta.Public.DisplayName
			pDescr = meta.Public.Description
		}

		// Get all role assignments for this project
		as, err := s.authzClient.AssignmentsToProject(ctx, proj)
		if err != nil {
			return nil, nil, status.Errorf(codes.Internal, "error getting role assignments: %v", err)
		}

		// TODO: Delete once all use ProjectRoles
		deprecatedPrjs = append(deprecatedPrjs, &pb.Project{
			ProjectId:   proj.String(),
			Name:        pinfo.Name,
			CreatedAt:   timestamppb.New(pinfo.CreatedAt),
			UpdatedAt:   timestamppb.New(pinfo.UpdatedAt),
			DisplayName: pDisplay,
			Description: pDescr,
		})

		var projectRole *pb.Role
		if len(as) != 0 {
			// Find the role for the user
			var roleString string
			for _, a := range as {
				if a.Subject == user.IdentitySubject {
					roleString = a.Role
				}
			}
			// Parse role
			authzRole, err := authz.ParseRole(roleString)
			if err != nil {
				return nil, nil, status.Errorf(codes.Internal, "failed to parse role: %v", err)
			}
			projectRole = &pb.Role{
				Name:        authzRole.String(),
				DisplayName: authz.AllRolesDisplayName[authzRole],
				Description: authz.AllRoles[authzRole],
			}
		}

		// Append the project role to the response
		projectRoles = append(projectRoles, &pb.ProjectRole{
			Role: projectRole,
			Project: &pb.Project{
				ProjectId:   proj.String(),
				Name:        pinfo.Name,
				CreatedAt:   timestamppb.New(pinfo.CreatedAt),
				UpdatedAt:   timestamppb.New(pinfo.UpdatedAt),
				DisplayName: pDisplay,
				Description: pDescr,
			},
		})
	}

	return projectRoles, deprecatedPrjs, nil
}

// GetUser is a service for getting personal user details
func (s *Server) GetUser(ctx context.Context, _ *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	// user is always authorized to get themselves
	tokenString, err := gauth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "no auth token: %v", err)
	}

	openIdToken, err := s.jwt.ParseAndValidate(tokenString)
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

	projectRoles, deprecatedPrjs, err := s.getUserDependencies(ctx, user)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get user dependencies: %s", err)
	}
	resp.ProjectRoles = projectRoles
	// nolint: staticcheck
	resp.Projects = deprecatedPrjs
	return &resp, nil
}

// ListInvitations is a service for listing invitations.
func (s *Server) ListInvitations(ctx context.Context, _ *pb.ListInvitationsRequest) (*pb.ListInvitationsResponse, error) {
	// Check if the UserManagement feature is enabled
	if !flags.Bool(ctx, s.featureFlags, flags.UserManagement) {
		return nil, status.Error(codes.Unimplemented, "feature not enabled")
	}
	invitations := make([]*pb.Invitation, 0)

	// Extracts the user email from the token
	tokenEmail, err := jwt.GetUserEmailFromContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user email: %s", err)
	}

	// Get the list of invitations for the user
	invites, err := s.store.GetInvitationsByEmail(ctx, tokenEmail)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get invitations: %s", err)
	}

	// Build the response list of invitations
	for _, i := range invites {
		// Resolve the sponsor's identity and display name
		identity, err := s.idClient.Resolve(ctx, i.IdentitySubject)
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("error resolving identity")
			return nil, util.UserVisibleError(codes.NotFound, "could not find identity %q", i.IdentitySubject)
		}

		// Resolve the project's display name
		targetProject, err := s.store.GetProjectByID(ctx, i.Project)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get project: %s", err)
		}

		// Parse the project metadata, so we can get the display name set by project owner
		meta, err := projects.ParseMetadata(&targetProject)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error parsing project metadata: %v", err)
		}

		invitations = append(invitations, &pb.Invitation{
			Code:           i.Code,
			Role:           i.Role,
			Email:          i.Email,
			Project:        i.Project.String(),
			ProjectDisplay: meta.Public.DisplayName,
			CreatedAt:      timestamppb.New(i.CreatedAt),
			ExpiresAt:      invite.GetExpireIn7Days(i.UpdatedAt),
			Expired:        invite.IsExpired(i.UpdatedAt),
			Sponsor:        identity.String(),
			SponsorDisplay: identity.Human(),
		})
	}

	return &pb.ListInvitationsResponse{
		Invitations: invitations,
	}, nil
}

// ResolveInvitation is a service for resolving an invitation.
func (s *Server) ResolveInvitation(ctx context.Context, req *pb.ResolveInvitationRequest) (*pb.ResolveInvitationResponse, error) {
	// Check if the UserManagement feature is enabled
	if !flags.Bool(ctx, s.featureFlags, flags.UserManagement) {
		return nil, status.Error(codes.Unimplemented, "feature not enabled")
	}

	// Check if the invitation code is valid
	userInvite, err := s.store.GetInvitationByCode(ctx, req.Code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "invitation not found or already used")
		}
		return nil, status.Errorf(codes.Internal, "failed to get invitation: %s", err)
	}

	// Check if the user is trying to resolve their own invitation
	if err = isUserSelfResolving(ctx, s.store, userInvite); err != nil {
		return nil, err
	}

	// Check if the invitation is expired
	if invite.IsExpired(userInvite.UpdatedAt) {
		return nil, util.UserVisibleError(codes.PermissionDenied, "invitation expired")
	}

	// Accept invitation
	if req.Accept {
		if err := s.acceptInvitation(ctx, userInvite); err != nil {
			return nil, err
		}
	}

	// Delete the invitation since its resolved
	deletedInvite, err := s.store.DeleteInvitation(ctx, req.Code)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete invitation: %s", err)
	}

	// Resolve the project's display name
	targetProject, err := s.store.GetProjectByID(ctx, deletedInvite.Project)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project: %s", err)
	}

	// Parse the project metadata, so we can get the display name set by project owner
	meta, err := projects.ParseMetadata(&targetProject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error parsing project metadata: %v", err)
	}

	return &pb.ResolveInvitationResponse{
		Role:           deletedInvite.Role,
		Project:        deletedInvite.Project.String(),
		ProjectDisplay: meta.Public.DisplayName,
		Email:          deletedInvite.Email,
		IsAccepted:     req.Accept,
	}, nil
}

func (s *Server) acceptInvitation(ctx context.Context, userInvite db.GetInvitationByCodeRow) error {
	// Validate in case there's an existing role assignment for the user
	as, err := s.authzClient.AssignmentsToProject(ctx, userInvite.Project)
	if err != nil {
		return status.Errorf(codes.Internal, "error getting role assignments: %v", err)
	}
	// Loop through all role assignments for the project and check if this user already has a role
	for _, existing := range as {
		if existing.Subject == jwt.GetUserSubjectFromContext(ctx) {
			// User already has the same role in the project
			if existing.Role == userInvite.Role {
				return util.UserVisibleError(codes.AlreadyExists, "user already has the same role in the project")
			}
			// Revoke the existing role assignments for the user in the project
			existingRole, err := authz.ParseRole(existing.Role)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to parse invitation role: %s", err)
			}
			// Delete the role assignment
			if err := s.authzClient.Delete(
				ctx,
				jwt.GetUserSubjectFromContext(ctx),
				existingRole,
				uuid.MustParse(*existing.Project),
			); err != nil {
				return status.Errorf(codes.Internal, "error writing role assignment: %v", err)
			}
		}
	}
	// Parse the role
	authzRole, err := authz.ParseRole(userInvite.Role)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to parse invitation role: %s", err)
	}
	// Add the user to the project
	if err := s.authzClient.Write(ctx, jwt.GetUserSubjectFromContext(ctx), authzRole, userInvite.Project); err != nil {
		return status.Errorf(codes.Internal, "error writing role assignment: %v", err)
	}
	return nil
}

// isUserSelfResolving is used to prevent if the user is trying to resolve an invitation they created
func isUserSelfResolving(ctx context.Context, store db.Store, i db.GetInvitationByCodeRow) error {
	// Get current user data
	currentUser, err := store.GetUserBySubject(ctx, jwt.GetUserSubjectFromContext(ctx))
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get user: %s", err)
	}

	// Check if the user is trying to resolve their own invitation
	if currentUser.ID == i.Sponsor {
		return util.UserVisibleError(codes.InvalidArgument, "user cannot resolve their own invitation")
	}

	return nil
}

// generateRandomString is used to generate a random string of a given length
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes)[:length], nil
}

// getUniqueProjectBaseName is used to generate a unique project name
func getUniqueProjectBaseName(ctx context.Context, store db.Store, baseName string) (string, error) {
	uniqueBaseName := baseName
	for {
		_, err := store.GetProjectByName(ctx, uniqueBaseName)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				break
			}
			return "", status.Errorf(codes.Internal, "failed to get project by name: %s", err)
		}
		r, err := generateRandomString(4)
		if err != nil {
			return "", status.Errorf(codes.Internal, "failed to generate random string: %s", err)
		}
		uniqueBaseName = baseName + "-" + r
	}
	return uniqueBaseName, nil
}
