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
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stacklok/mediator/pkg/auth"
	mcrypto "github.com/stacklok/mediator/pkg/crypto"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type createUserValidation struct {
	OrganizationId int32  `db:"organization_id" validate:"required"`
	Email          string `db:"email" validate:"omitempty,email"`
	FirstName      string `db:"first_name" validate:"omitempty,alphaunicode"`
	LastName       string `db:"last_name" validate:"omitempty,alphaunicode"`
	Username       string `db:"username" validate:"required"`
	Password       string `validate:"omitempty,min=8,containsany=_.;?&@"`
}

func stringToNullString(s *string) *sql.NullString {
	if s == nil {
		return &sql.NullString{Valid: false}
	}
	return &sql.NullString{String: *s, Valid: true}
}

// CreateUser is a service for creating an organization
//
//gocyclo:ignore
func (s *Server) CreateUser(ctx context.Context,
	in *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {

	// validate that the company and name are not empty, and email is valid if exists
	validator := validator.New()
	format := createUserValidation{OrganizationId: in.OrganizationId, Username: in.Username}

	if in.Email != nil {
		format.Email = *in.Email
	}

	if in.Password != nil {
		format.Password = *in.Password
	}

	err := validator.Struct(format)
	if err != nil {
		return nil, err
	}

	if in.IsProtected == nil {
		isProtected := false
		in.IsProtected = &isProtected
	}

	// if email is blank, set to null
	if in.Email != nil && *in.Email == "" {
		in.Email = nil
	}

	// if password is not set, we generate a random one
	seed := time.Now().UnixNano()

	if in.Password == nil || *in.Password == "" {
		pass := util.RandomPassword(8, seed)
		in.Password = &pass
	}

	// hash the password for storing in the database
	pHash, err := mcrypto.GeneratePasswordHash(*in.Password)
	if err != nil {
		return nil, err
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.OrganizationId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	// if we have group ids we check if they exist
	if in.GroupIds != nil {
		for _, id := range in.GroupIds {
			group, err := s.store.GetGroupByID(ctx, id)
			if err != nil {
				return nil, status.Errorf(codes.NotFound, "group not found")
			}

			// group must belong to org
			if group.OrganizationID != in.OrganizationId {
				return nil, status.Errorf(codes.InvalidArgument, "group does not belong to organization")
			}
		}
	}

	// if we have role ids we check if they exist
	if in.RoleIds != nil {
		for _, id := range in.RoleIds {
			role, err := s.store.GetRoleByID(ctx, id)
			if err != nil {
				return nil, status.Errorf(codes.NotFound, "role not found")
			}

			// role must belong to org
			if role.OrganizationID != in.OrganizationId {
				return nil, status.Errorf(codes.InvalidArgument, "role does not belong to organization")
			}
		}
	}

	needsPassPtr := false
	if in.NeedsPasswordChange != nil {
		needsPassPtr = *in.NeedsPasswordChange
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

	user, err := qtx.CreateUser(ctx, db.CreateUserParams{OrganizationID: in.OrganizationId,
		Email: *stringToNullString(in.Email), Username: in.Username, Password: pHash,
		FirstName: *stringToNullString(in.FirstName), LastName: *stringToNullString(in.LastName),
		IsProtected: *in.IsProtected, NeedsPasswordChange: needsPassPtr})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create user")
	}

	// now attach the groups and roles
	for _, id := range in.GroupIds {
		_, err := qtx.AddUserGroup(ctx, db.AddUserGroupParams{UserID: user.ID, GroupID: id})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to add user to group")
		}
	}
	for _, id := range in.RoleIds {
		_, err := qtx.AddUserRole(ctx, db.AddUserRoleParams{UserID: user.ID, RoleID: id})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to add user to role")
		}
	}
	err = s.store.Commit(tx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction")
	}

	return &pb.CreateUserResponse{Id: user.ID, OrganizationId: user.OrganizationID, Email: &user.Email.String,
		Username: user.Username, Password: *in.Password, FirstName: &user.FirstName.String,
		LastName: &user.LastName.String, IsProtected: &user.IsProtected,
		NeedsPasswordChange: &user.NeedsPasswordChange, CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt)}, nil
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
		return nil, err
	}

	// first check if the user exists and is not protected
	user, err := s.store.GetUserByID(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, user.OrganizationID) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	if in.Force == nil {
		isProtected := false
		in.Force = &isProtected
	}

	if !*in.Force && user.IsProtected {
		errcode := fmt.Errorf("cannot delete a protected user")
		return nil, errcode
	}

	err = s.store.DeleteUser(ctx, in.Id)
	if err != nil {
		return nil, err
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
		return nil, status.Errorf(codes.Unknown, "failed to get users: %s", err)
	}

	var resp pb.GetUsersResponse
	resp.Users = make([]*pb.UserRecord, 0, len(users))
	for _, user := range users {
		resp.Users = append(resp.Users, &pb.UserRecord{
			Id:                  user.ID,
			OrganizationId:      user.OrganizationID,
			Email:               &user.Email.String,
			Username:            user.Username,
			FirstName:           &user.FirstName.String,
			LastName:            &user.LastName.String,
			IsProtected:         &user.IsProtected,
			NeedsPasswordChange: &user.NeedsPasswordChange,
			CreatedAt:           timestamppb.New(user.CreatedAt),
			UpdatedAt:           timestamppb.New(user.UpdatedAt),
		})
	}

	return &resp, nil
}

// GetUsersByOrganization is a service for getting a list of users of an organization
func (s *Server) GetUsersByOrganization(ctx context.Context,
	in *pb.GetUsersByOrganizationRequest) (*pb.GetUsersByOrganizationResponse, error) {
	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.OrganizationId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
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
		OrganizationID: in.OrganizationId,
		Limit:          *in.Limit,
		Offset:         *in.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	var resp pb.GetUsersByOrganizationResponse
	resp.Users = make([]*pb.UserRecord, 0, len(users))
	for _, user := range users {
		resp.Users = append(resp.Users, &pb.UserRecord{
			Id:                  user.ID,
			OrganizationId:      user.OrganizationID,
			Email:               &user.Email.String,
			Username:            user.Username,
			FirstName:           &user.FirstName.String,
			LastName:            &user.LastName.String,
			IsProtected:         &user.IsProtected,
			NeedsPasswordChange: &user.NeedsPasswordChange,
			CreatedAt:           timestamppb.New(user.CreatedAt),
			UpdatedAt:           timestamppb.New(user.UpdatedAt),
		})
	}

	return &resp, nil
}

// GetUsersByGroup is a service for getting a list of users of a group
func (s *Server) GetUsersByGroup(ctx context.Context,
	in *pb.GetUsersByGroupRequest) (*pb.GetUsersByGroupResponse, error) {
	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.GroupId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
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

	users, err := s.store.ListUsersByGroup(ctx, db.ListUsersByGroupParams{
		GroupID: in.GroupId,
		Limit:   *in.Limit,
		Offset:  *in.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	var resp pb.GetUsersByGroupResponse
	resp.Users = make([]*pb.UserRecord, 0, len(users))
	for _, user := range users {
		resp.Users = append(resp.Users, &pb.UserRecord{
			Id:                  user.ID,
			OrganizationId:      user.OrganizationID,
			Email:               &user.Email.String,
			Username:            user.Username,
			FirstName:           &user.FirstName.String,
			LastName:            &user.LastName.String,
			IsProtected:         &user.IsProtected,
			NeedsPasswordChange: &user.NeedsPasswordChange,
			CreatedAt:           timestamppb.New(user.CreatedAt),
			UpdatedAt:           timestamppb.New(user.UpdatedAt),
		})
	}

	return &resp, nil
}

func getUserDependencies(ctx context.Context, store db.Store, user db.User) ([]*pb.GroupRecord, []*pb.RoleRecord, error) {
	// get all the roles associated with that user
	roles, err := store.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	// get all the groups associated with that user
	groups, err := store.GetUserGroups(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	// convert to right data type
	var rolesPB []*pb.RoleRecord
	for _, role := range roles {
		rolesPB = append(rolesPB, &pb.RoleRecord{
			Id:             role.ID,
			OrganizationId: role.OrganizationID,
			GroupId:        &role.GroupID.Int32,
			Name:           role.Name,
			IsAdmin:        role.IsAdmin,
			IsProtected:    role.IsProtected,
			CreatedAt:      timestamppb.New(role.CreatedAt),
			UpdatedAt:      timestamppb.New(role.UpdatedAt),
		})
	}

	var groupsPB []*pb.GroupRecord
	for _, group := range groups {
		groupsPB = append(groupsPB, &pb.GroupRecord{
			GroupId:        group.ID,
			OrganizationId: group.OrganizationID,
			Name:           group.Name,
			Description:    group.Description.String,
			IsProtected:    group.IsProtected,
			CreatedAt:      timestamppb.New(group.CreatedAt),
			UpdatedAt:      timestamppb.New(group.UpdatedAt),
		})
	}

	return groupsPB, rolesPB, nil
}

// GetUserById is a service for getting a user by id
func (s *Server) GetUserById(ctx context.Context,
	in *pb.GetUserByIdRequest) (*pb.GetUserByIdResponse, error) {
	if in.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.Id) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	user, err := s.store.GetUserByID(ctx, in.Id)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get user: %s", err)
	}

	groups, roles, err := getUserDependencies(ctx, s.store, user)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get user dependencies: %s", err)
	}

	var resp pb.GetUserByIdResponse
	resp.User = &pb.UserRecord{
		Id:                  user.ID,
		OrganizationId:      user.OrganizationID,
		Email:               &user.Email.String,
		Username:            user.Username,
		FirstName:           &user.FirstName.String,
		LastName:            &user.LastName.String,
		IsProtected:         &user.IsProtected,
		NeedsPasswordChange: &user.NeedsPasswordChange,
		CreatedAt:           timestamppb.New(user.CreatedAt),
		UpdatedAt:           timestamppb.New(user.UpdatedAt),
	}

	resp.Groups = groups
	resp.Roles = roles
	return &resp, nil
}

// GetUserByUserName is a service for getting an user by username
func (s *Server) GetUserByUserName(ctx context.Context,
	in *pb.GetUserByUserNameRequest) (*pb.GetUserByUserNameResponse, error) {
	if in.GetUsername() == "" {
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}

	user, err := s.store.GetUserByUserName(ctx, in.Username)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get user: %s", err)
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, user.OrganizationID) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	groups, roles, err := getUserDependencies(ctx, s.store, user)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get user dependencies: %s", err)
	}

	var resp pb.GetUserByUserNameResponse
	resp.User = &pb.UserRecord{
		Id:                  user.ID,
		OrganizationId:      user.OrganizationID,
		Email:               &user.Email.String,
		Username:            user.Username,
		Password:            user.Password,
		FirstName:           &user.FirstName.String,
		LastName:            &user.LastName.String,
		NeedsPasswordChange: &user.NeedsPasswordChange,
		CreatedAt:           timestamppb.New(user.CreatedAt),
		UpdatedAt:           timestamppb.New(user.UpdatedAt),
	}

	resp.Groups = groups
	resp.Roles = roles
	return &resp, nil
}

// GetUserByEmail is a service for getting an user by email
func (s *Server) GetUserByEmail(ctx context.Context,
	in *pb.GetUserByEmailRequest) (*pb.GetUserByEmailResponse, error) {
	if in.GetEmail() == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	user, err := s.store.GetUserByEmail(ctx, sql.NullString{String: in.Email, Valid: true})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get user: %s", err)
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, user.OrganizationID) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	var resp pb.GetUserByEmailResponse
	resp.User = &pb.UserRecord{
		Id:                  user.ID,
		OrganizationId:      user.OrganizationID,
		Email:               &user.Email.String,
		Username:            user.Username,
		FirstName:           &user.FirstName.String,
		LastName:            &user.LastName.String,
		NeedsPasswordChange: &user.NeedsPasswordChange,
		CreatedAt:           timestamppb.New(user.CreatedAt),
		UpdatedAt:           timestamppb.New(user.UpdatedAt),
	}

	groups, roles, err := getUserDependencies(ctx, s.store, user)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get user dependencies: %s", err)
	}
	resp.Groups = groups
	resp.Roles = roles

	return &resp, nil
}

// GetUser is a service for getting personal user details
func (s *Server) GetUser(ctx context.Context,
	in *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	claims, _ := ctx.Value(TokenInfoKey).(auth.UserClaims)
	// check if user is authorized
	if !IsRequestAuthorized(ctx, claims.UserId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	// check if user exists
	user, err := s.store.GetUserByID(ctx, claims.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %s", err)
	}

	var resp pb.GetUserResponse
	resp.User = &pb.UserRecord{
		Id:                  user.ID,
		OrganizationId:      user.OrganizationID,
		Email:               &user.Email.String,
		Username:            user.Username,
		FirstName:           &user.FirstName.String,
		LastName:            &user.LastName.String,
		NeedsPasswordChange: &user.NeedsPasswordChange,
		CreatedAt:           timestamppb.New(user.CreatedAt),
		UpdatedAt:           timestamppb.New(user.UpdatedAt),
	}

	groups, roles, err := getUserDependencies(ctx, s.store, user)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get user dependencies: %s", err)
	}
	resp.Groups = groups
	resp.Roles = roles

	return &resp, nil
}

type updatePasswordValidation struct {
	Password             string `validate:"min=8,containsany=_.;?&@"`
	PasswordConfirmation string `validate:"min=8,containsany=_.;?&@"`
}

// UpdatePassword is a service for updating a user's password
func (s *Server) UpdatePassword(ctx context.Context, in *pb.UpdatePasswordRequest) (*pb.UpdatePasswordResponse, error) {
	claims, _ := ctx.Value(TokenInfoKey).(auth.UserClaims)
	// check if user is authorized
	if !IsRequestAuthorized(ctx, claims.UserId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	// validate password
	validator := validator.New()
	err := validator.Struct(updatePasswordValidation{Password: in.Password, PasswordConfirmation: in.PasswordConfirmation})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument,
			"password must be at least 8 characters long and contain one of the following characters: !@#?*")
	}

	if in.Password != in.PasswordConfirmation {
		return nil, status.Error(codes.InvalidArgument, "passwords do not match")
	}

	// hash the password for storing in the database
	pHash, err := mcrypto.GeneratePasswordHash(in.Password)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate password hash: %s", err)
	}

	// check if the previous password was the same
	user, err := s.store.GetUserByID(ctx, claims.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %s", err)
	}

	match, _ := mcrypto.VerifyPasswordHash(pHash, user.Password)
	if match {
		return nil, status.Errorf(codes.NotFound, "User and password not found: %s", err)
	}

	_, err = s.store.UpdatePassword(ctx, db.UpdatePasswordParams{ID: claims.UserId, Password: pHash})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update password: %s", err)
	}

	// revoke token for the user
	_, err = s.store.RevokeUserToken(ctx, db.RevokeUserTokenParams{ID: claims.UserId,
		MinTokenIssuedTime: sql.NullTime{Time: time.Unix(time.Now().Unix(), 0), Valid: true}})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to revoke user token: %s", err)
	}

	return &pb.UpdatePasswordResponse{}, nil
}

type updateProfileValidation struct {
	Email     string `db:"email" validate:"omitempty,email"`
	FirstName string `db:"first_name" validate:"omitempty,alphaunicode"`
	LastName  string `db:"last_name" validate:"omitempty,alphaunicode"`
}

// UpdateProfile is a service for updating a user's profile
//
//gocyclo:ignore
func (s *Server) UpdateProfile(ctx context.Context, in *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	claims, _ := ctx.Value(TokenInfoKey).(auth.UserClaims)
	// check if user is authorized
	if !IsRequestAuthorized(ctx, claims.UserId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	// validate that at least one field is being updated
	if (in.Email == nil || *in.Email == "") && (in.FirstName == nil || *in.FirstName == "") &&
		(in.LastName == nil || *in.LastName == "") {
		return nil, status.Errorf(codes.InvalidArgument, "at least one field must be updated")
	}

	updateProfileValidation := updateProfileValidation{}
	if in.Email != nil {
		updateProfileValidation.Email = *in.Email
	}
	if in.FirstName != nil {
		updateProfileValidation.FirstName = *in.FirstName
	}
	if in.LastName != nil {
		updateProfileValidation.LastName = *in.LastName
	}
	validator := validator.New()
	err := validator.Struct(updateProfileValidation)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid fields for updating profile")
	}

	// get details of user
	user, err := s.store.GetUserByID(ctx, claims.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get user: %s", err)
	}

	// now update the modified fields
	if in.Email != nil && *in.Email != "" {
		user.Email = sql.NullString{String: *in.Email, Valid: true}
	}
	if in.FirstName != nil && *in.FirstName != "" {
		user.FirstName = sql.NullString{String: *in.FirstName, Valid: true}
	}
	if in.LastName != nil && *in.LastName != "" {
		user.LastName = sql.NullString{String: *in.LastName, Valid: true}
	}

	_, err = s.store.UpdateUser(ctx, db.UpdateUserParams{ID: claims.UserId, Email: user.Email,
		FirstName: user.FirstName, LastName: user.LastName})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update user: %s", err)
	}

	// return updated details
	return &pb.UpdateProfileResponse{}, nil
}
