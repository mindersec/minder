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
	"encoding/json"
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	gauth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/mediator/internal/db"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

func stringToNullString(s string) *sql.NullString {
	if s == "" {
		return &sql.NullString{Valid: false}
	}
	return &sql.NullString{String: s, Valid: true}
}

// CreateUser is a service for user self registration
//
//gocyclo:ignore
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

	var userOrg uuid.UUID
	var userProject uuid.UUID

	orgmeta := &OrgMeta{
		Company: subject + " - Self enrolled",
	}

	marshaled, err := json.Marshal(orgmeta)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal org metadata: %s", err)
	}

	baseName := subject
	if token.PreferredUsername() != "" {
		baseName = token.PreferredUsername()
	}

	// otherwise self-enroll user, by creating a new org and project and making the user an admin of those
	organization, err := qtx.CreateOrganization(ctx, db.CreateOrganizationParams{
		Name:     baseName + "-org",
		Metadata: marshaled,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create organization: %s", err)
	}
	orgProject, userRoles, err := CreateDefaultRecordsForOrg(ctx, qtx, organization, baseName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create default organization records: %s", err)
	}

	userOrg = organization.ID
	userProject = uuid.MustParse(orgProject.ProjectId)
	user, err := qtx.CreateUser(ctx, db.CreateUserParams{OrganizationID: userOrg,
		Email:           *stringToNullString(token.Email()),
		FirstName:       *stringToNullString(token.GivenName()),
		LastName:        *stringToNullString(token.FamilyName()),
		IdentitySubject: subject})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user: %s", err)
	}

	_, err = qtx.AddUserProject(ctx, db.AddUserProjectParams{UserID: user.ID, ProjectID: userProject})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add user to project: %s", err)
	}

	for _, id := range userRoles {
		_, err := qtx.AddUserRole(ctx, db.AddUserRoleParams{UserID: user.ID, RoleID: id})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to add user to role: %s", err)
		}
	}
	err = s.store.Commit(tx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %s", err)
	}

	return &pb.CreateUserResponse{
		Id:              user.ID,
		OrganizationId:  user.OrganizationID.String(),
		OrganizatioName: organization.Name,
		ProjectId:       userProject.String(),
		ProjectName:     orgProject.Name,
		Email:           &user.Email.String,
		IdentitySubject: user.IdentitySubject,
		FirstName:       &user.FirstName.String,
		LastName:        &user.LastName.String,
		CreatedAt:       timestamppb.New(user.CreatedAt),
	}, nil
}

type deleteUserValidation struct {
	Id int32 `db:"id" validate:"required"`
}

// DeleteUser is a service for deleting an user
func (s *Server) DeleteUser(ctx context.Context,
	in *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	validator := validator.New()
	err := validator.Struct(deleteUserValidation{Id: in.Id})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid argument: %s", err.Error())
	}

	// first check if the user exists and is not protected
	user, err := s.store.GetUserByID(ctx, in.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %s", err)
	}

	// check if user is authorized
	if err := AuthorizedOnOrg(ctx, user.OrganizationID); err != nil {
		return nil, err
	}

	if in.Force == nil {
		isProtected := false
		in.Force = &isProtected
	}

	err = s.store.DeleteUser(ctx, in.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete user: %s", err)
	}

	return &pb.DeleteUserResponse{}, nil
}

func getUserDependencies(ctx context.Context, store db.Store, user db.User) ([]*pb.Project, error) {
	// get all the projects associated with that user
	projects, err := store.GetUserProjects(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	var projectsPB []*pb.Project
	for _, proj := range projects {
		projectsPB = append(projectsPB, &pb.Project{
			ProjectId: proj.ID.String(),
			Name:      proj.Name,
			CreatedAt: timestamppb.New(proj.CreatedAt),
			UpdatedAt: timestamppb.New(proj.UpdatedAt),
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
		OrganizationId:  user.OrganizationID.String(),
		Email:           &user.Email.String,
		IdentitySubject: user.IdentitySubject,
		FirstName:       &user.FirstName.String,
		LastName:        &user.LastName.String,
		CreatedAt:       timestamppb.New(user.CreatedAt),
		UpdatedAt:       timestamppb.New(user.UpdatedAt),
	}

	projects, err := getUserDependencies(ctx, s.store, user)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get user dependencies: %s", err)
	}
	resp.Projects = projects

	return &resp, nil
}
