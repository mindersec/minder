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
	"net/url"

	"github.com/google/uuid"
	gauth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/projects"
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

	var userProject uuid.UUID

	baseName := subject
	if token.PreferredUsername() != "" {
		baseName = token.PreferredUsername()
	}

	orgProject, err := projects.ProvisionSelfEnrolledProject(ctx, s.authzClient, qtx, baseName, subject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create default organization records: %s", err)
	}

	userProject = uuid.MustParse(orgProject.ProjectId)
	user, err := qtx.CreateUser(ctx, subject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %s", err)
	}

	err = s.store.Commit(tx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %s", err)
	}

	// Telemetry logging
	logger.BusinessRecord(ctx).Project = userProject

	return &pb.CreateUserResponse{
		Id:              user.ID,
		ProjectId:       userProject.String(),
		ProjectName:     orgProject.Name,
		IdentitySubject: user.IdentitySubject,
		CreatedAt:       timestamppb.New(user.CreatedAt),
	}, nil
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

	parsedURL, err := url.Parse(s.cfg.Identity.Server.IssuerUrl)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to parse issuer URL: %v", err)
	}

	err = DeleteUser(ctx, s.store, s.authzClient, subject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete user from database: %v", err)
	}

	tokenUrl := parsedURL.JoinPath("realms/stacklok/protocol/openid-connect/token")

	clientSecret, err := s.cfg.Identity.Server.GetClientSecret()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get client secret: %v", err)
	}

	clientCredentials := clientcredentials.Config{
		ClientID:     s.cfg.Identity.Server.ClientId,
		ClientSecret: clientSecret,
		TokenURL:     tokenUrl.String(),
	}

	ccToken, err := clientCredentials.Token(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get client access token: %v", err)
	}

	deleteUrl := parsedURL.JoinPath("admin/realms/stacklok/users", subject)
	request, err := http.NewRequest("DELETE", deleteUrl.String(), nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to construct account deletion request: %v", err)
	}

	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(ccToken))
	resp, err := client.Do(request)
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
