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
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type createUserValidation struct {
	RoleId   int32  `db:"role_id" validate:"required"`
	Email    string `db:"email" validate:"omitempty,email"`
	Username string `db:"username" validate:"required"`
	Password string `validate:"omitempty,min=8,containsany=!@#?*"`
}

func stringToNullString(s *string) *sql.NullString {
	if s == nil {
		return &sql.NullString{Valid: false}
	}
	return &sql.NullString{String: *s, Valid: true}
}

// CreateUser is a service for creating an organization
func (s *Server) CreateUser(ctx context.Context,
	in *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {

	// validate that the company and name are not empty, and email is valid if exists
	validator := validator.New()
	format := createUserValidation{RoleId: in.RoleId, Username: in.Username}

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

	user, err := s.store.CreateUser(ctx, db.CreateUserParams{RoleID: in.RoleId,
		Email: *stringToNullString(in.Email), Username: in.Username, Password: *in.Password,
		FirstName: *stringToNullString(in.FirstName), LastName: *stringToNullString(in.LastName),
		IsProtected: *in.IsProtected})
	if err != nil {
		return nil, err
	}

	return &pb.CreateUserResponse{Id: user.ID, RoleId: user.RoleID, Email: &user.Email.String,
		Username: user.Username, Password: *in.Password, FirstName: &user.FirstName.String,
		LastName: &user.LastName.String, IsProtected: &user.IsProtected,
		CreatedAt: timestamppb.New(user.CreatedAt), UpdatedAt: timestamppb.New(user.UpdatedAt)}, nil
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
	if in.RoleId == 0 {
		return nil, fmt.Errorf("role id is required")
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

	users, err := s.store.ListUsers(ctx, db.ListUsersParams{
		RoleID: in.RoleId,
		Limit:  *in.Limit,
		Offset: *in.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	var resp pb.GetUsersResponse
	resp.Users = make([]*pb.UserRecord, 0, len(users))
	for _, user := range users {
		resp.Users = append(resp.Users, &pb.UserRecord{
			Id:          user.ID,
			RoleId:      user.RoleID,
			Email:       &user.Email.String,
			Username:    user.Username,
			FirstName:   &user.FirstName.String,
			LastName:    &user.LastName.String,
			IsProtected: &user.IsProtected,
			CreatedAt:   timestamppb.New(user.CreatedAt),
			UpdatedAt:   timestamppb.New(user.UpdatedAt),
		})
	}

	return &resp, nil
}

// GetUserById is a service for getting a user by id
func (s *Server) GetUserById(ctx context.Context,
	in *pb.GetUserByIdRequest) (*pb.GetUserByIdResponse, error) {
	if in.Id == 0 {
		return nil, fmt.Errorf("user id is required")
	}

	user, err := s.store.GetUserByID(ctx, in.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var resp pb.GetUserByIdResponse
	resp.User = &pb.UserRecord{
		Id:          user.ID,
		RoleId:      user.RoleID,
		Email:       &user.Email.String,
		Username:    user.Username,
		FirstName:   &user.FirstName.String,
		LastName:    &user.LastName.String,
		IsProtected: &user.IsProtected,
		CreatedAt:   timestamppb.New(user.CreatedAt),
		UpdatedAt:   timestamppb.New(user.UpdatedAt),
	}

	return &resp, nil
}

// GetUserByUsername is a service for getting an user by username
func (s *Server) GetUserByUsername(ctx context.Context,
	in *pb.GetUserByUserNameRequest) (*pb.GetUserByUserNameResponse, error) {
	if in.GetUsername() == "" {
		return nil, fmt.Errorf("username is required")
	}

	user, err := s.store.GetUserByUserName(ctx, in.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var resp pb.GetUserByUserNameResponse
	resp.User = &pb.UserRecord{
		Id:        user.ID,
		RoleId:    user.RoleID,
		Email:     &user.Email.String,
		Username:  user.Username,
		FirstName: &user.FirstName.String,
		LastName:  &user.LastName.String,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}

	return &resp, nil
}

// GetUserByEmail is a service for getting an user by email
func (s *Server) GetUserByEmail(ctx context.Context,
	in *pb.GetUserByEmailRequest) (*pb.GetUserByEmailResponse, error) {
	if in.GetEmail() == "" {
		return nil, fmt.Errorf("email is required")
	}

	user, err := s.store.GetUserByEmail(ctx, sql.NullString{String: in.Email, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var resp pb.GetUserByEmailResponse
	resp.User = &pb.UserRecord{
		Id:        user.ID,
		RoleId:    user.RoleID,
		Email:     &user.Email.String,
		Username:  user.Username,
		FirstName: &user.FirstName.String,
		LastName:  &user.LastName.String,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.UpdatedAt),
	}

	return &resp, nil
}
