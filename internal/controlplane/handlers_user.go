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
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/util"
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
	var userRoles []int32

	if containsSuperadminRole(token) {
		// if the token has superadmin access to the realm, then make give them a superadmin role in the DB
		userOrg = rootOrganization
		userProject = rootProject
		userRoles = append(userRoles, superadminRole)
	} else {
		orgmeta := &OrgMeta{
			Company: subject + " - Self enrolled",
		}

		marshaled, err := json.Marshal(orgmeta)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal org metadata: %s", err)
		}

		// otherwise self-enroll user, by creating a new org and project and making the user an admin of those
		organization, err := qtx.CreateOrganization(ctx, db.CreateOrganizationParams{
			Name:     subject + "-org",
			Metadata: marshaled,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create organization: %s", err)
		}
		orgProject, orgRoles, err := CreateDefaultRecordsForOrg(ctx, qtx, organization)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create default organization records: %s", err)
		}

		userOrg = organization.ID
		userProject = uuid.MustParse(orgProject.ProjectId)
		for _, role := range orgRoles {
			userRoles = append(userRoles, role.Id)
		}
	}

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

	return &pb.CreateUserResponse{Id: user.ID, OrganizationId: user.OrganizationID.String(), Email: &user.Email.String,
		IdentitySubject: user.IdentitySubject, FirstName: &user.FirstName.String, LastName: &user.LastName.String,
		CreatedAt: timestamppb.New(user.CreatedAt), UpdatedAt: timestamppb.New(user.UpdatedAt)}, nil
}

func containsSuperadminRole(openIdToken openid.Token) bool {
	if realmAccess, ok := openIdToken.Get("realm_access"); ok {
		if realms, ok := realmAccess.(map[string]interface{}); ok {
			if roles, ok := realms["roles"]; ok {
				if userRoles, ok := roles.([]interface{}); ok {
					if slices.Contains(userRoles, "superadmin") {
						return true
					}
				}
			}
		}
	}
	return false
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

// GetUsers is a service for getting a list of users
func (s *Server) GetUsers(ctx context.Context,
	in *pb.GetUsersRequest) (*pb.GetUsersResponse, error) {

	// define default values for limit and offset
	if in.Limit == nil || *in.Limit == -1 {
		in.Limit = new(int32)
		*in.Limit = PaginationLimit
	}
	if in.Offset == nil {
		in.Offset = new(int32)
		*in.Offset = 0
	}

	users, err := s.store.ListUsers(ctx, db.ListUsersParams{
		Limit:  *in.Limit,
		Offset: *in.Offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get users: %s", err)
	}

	var resp pb.GetUsersResponse
	resp.Users = make([]*pb.UserRecord, 0, len(users))
	for idx := range users {
		user := &users[idx]
		resp.Users = append(resp.Users, &pb.UserRecord{
			Id:              user.ID,
			OrganizationId:  user.OrganizationID.String(),
			Email:           &user.Email.String,
			IdentitySubject: user.IdentitySubject,
			FirstName:       &user.FirstName.String,
			LastName:        &user.LastName.String,
			CreatedAt:       timestamppb.New(user.CreatedAt),
			UpdatedAt:       timestamppb.New(user.UpdatedAt),
		})
	}

	return &resp, nil
}

// GetUsersByOrganization is a service for getting a list of users of an organization
func (s *Server) GetUsersByOrganization(ctx context.Context,
	in *pb.GetUsersByOrganizationRequest) (*pb.GetUsersByOrganizationResponse, error) {
	orgID, err := uuid.Parse(in.OrganizationId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid organization id")
	}

	// check if user is authorized
	if err := AuthorizedOnOrg(ctx, orgID); err != nil {
		return nil, err
	}

	// define default values for limit and offset
	if in.Limit == nil || *in.Limit == -1 {
		in.Limit = new(int32)
		*in.Limit = PaginationLimit
	}
	if in.Offset == nil {
		in.Offset = new(int32)
		*in.Offset = 0
	}

	users, err := s.store.ListUsersByOrganization(ctx, db.ListUsersByOrganizationParams{
		OrganizationID: orgID,
		Limit:          *in.Limit,
		Offset:         *in.Offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get users: %s", err)
	}

	var resp pb.GetUsersByOrganizationResponse
	resp.Users = make([]*pb.UserRecord, 0, len(users))
	for idx := range users {
		user := &users[idx]
		resp.Users = append(resp.Users, &pb.UserRecord{
			Id:              user.ID,
			OrganizationId:  user.OrganizationID.String(),
			Email:           &user.Email.String,
			IdentitySubject: user.IdentitySubject,
			FirstName:       &user.FirstName.String,
			LastName:        &user.LastName.String,
			CreatedAt:       timestamppb.New(user.CreatedAt),
			UpdatedAt:       timestamppb.New(user.UpdatedAt),
		})
	}

	return &resp, nil
}

// GetUsersByProject is a service for getting a list of users of a project
func (s *Server) GetUsersByProject(ctx context.Context,
	in *pb.GetUsersByProjectRequest) (*pb.GetUsersByProjectResponse, error) {
	projID, err := uuid.Parse(in.ProjectId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid project id")
	}

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, projID); err != nil {
		return nil, err
	}

	// define default values for limit and offset
	if in.Limit == nil || *in.Limit == -1 {
		in.Limit = new(int32)
		*in.Limit = PaginationLimit
	}
	if in.Offset == nil {
		in.Offset = new(int32)
		*in.Offset = 0
	}

	users, err := s.store.ListUsersByProject(ctx, db.ListUsersByProjectParams{
		ProjectID: projID,
		Limit:     *in.Limit,
		Offset:    *in.Offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get users: %s", err)
	}

	var resp pb.GetUsersByProjectResponse
	resp.Users = make([]*pb.UserRecord, 0, len(users))
	for idx := range users {
		user := &users[idx]
		resp.Users = append(resp.Users, &pb.UserRecord{
			Id:              user.ID,
			OrganizationId:  user.OrganizationID.String(),
			Email:           &user.Email.String,
			IdentitySubject: user.IdentitySubject,
			FirstName:       &user.FirstName.String,
			LastName:        &user.LastName.String,
			CreatedAt:       timestamppb.New(user.CreatedAt),
			UpdatedAt:       timestamppb.New(user.UpdatedAt),
		})
	}

	return &resp, nil
}

func getUserDependencies(ctx context.Context, store db.Store, user db.User) ([]*pb.ProjectRecord, []*pb.RoleRecord, error) {
	// get all the roles associated with that user
	roles, err := store.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	// get all the projects associated with that user
	projects, err := store.GetUserProjects(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	// convert to right data type
	var rolesPB []*pb.RoleRecord
	for idx := range roles {
		role := &roles[idx]
		pid := role.ProjectID.UUID.String()
		rolesPB = append(rolesPB, &pb.RoleRecord{
			Id:             role.ID,
			OrganizationId: role.OrganizationID.String(),
			ProjectId:      &pid,
			Name:           role.Name,
			IsAdmin:        role.IsAdmin,
			IsProtected:    role.IsProtected,
			CreatedAt:      timestamppb.New(role.CreatedAt),
			UpdatedAt:      timestamppb.New(role.UpdatedAt),
		})
	}

	var projectsPB []*pb.ProjectRecord
	for _, proj := range projects {
		projectsPB = append(projectsPB, &pb.ProjectRecord{
			ProjectId:      proj.ID.String(),
			OrganizationId: proj.ParentID.UUID.String(),
			Name:           proj.Name,
			CreatedAt:      timestamppb.New(proj.CreatedAt),
			UpdatedAt:      timestamppb.New(proj.UpdatedAt),
		})
	}

	return projectsPB, rolesPB, nil
}

// GetUserById is a service for getting a user by id
func (s *Server) GetUserById(ctx context.Context,
	in *pb.GetUserByIdRequest) (*pb.GetUserByIdResponse, error) {
	if in.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	// check if user is authorized
	if err := AuthorizedOnUser(ctx, in.UserId); err != nil {
		return nil, err
	}

	user, err := s.store.GetUserByID(ctx, in.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get user: %s", err)
	}

	projects, roles, err := getUserDependencies(ctx, s.store, user)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get user dependencies: %s", err)
	}

	var resp pb.GetUserByIdResponse
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

	resp.Projects = projects
	resp.Roles = roles
	return &resp, nil
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

	projects, roles, err := getUserDependencies(ctx, s.store, user)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get user dependencies: %s", err)
	}
	resp.Projects = projects
	resp.Roles = roles

	return &resp, nil
}
