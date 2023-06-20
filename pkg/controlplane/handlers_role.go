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
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateRoleValidation is a struct for validating the CreateRole request
type CreateRoleValidation struct {
	GroupId int32  `db:"group_id" validate:"required"`
	Name    string `db:"name" validate:"required"`
}

// CreateRole is a service for creating an organization
func (s *Server) CreateRole(ctx context.Context,
	in *pb.CreateRoleRequest) (*pb.CreateRoleResponse, error) {
	// validate that the company and name are not empty
	validator := validator.New()
	err := validator.Struct(CreateRoleValidation{GroupId: in.GroupId, Name: in.Name})
	if err != nil {
		return nil, err
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.GroupId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	if in.IsAdmin == nil {
		isAdmin := false
		in.IsAdmin = &isAdmin
	}

	if in.IsProtected == nil {
		isProtected := false
		in.IsProtected = &isProtected
	}

	role, err := s.store.CreateRole(ctx, db.CreateRoleParams{GroupID: in.GroupId,
		Name: in.Name, IsAdmin: *in.IsAdmin, IsProtected: *in.IsProtected})
	if err != nil {
		return nil, err
	}

	return &pb.CreateRoleResponse{Id: role.ID, Name: role.Name,
		IsAdmin: role.IsAdmin, IsProtected: role.IsProtected,
		GroupId:   role.GroupID,
		CreatedAt: timestamppb.New(role.CreatedAt),
		UpdatedAt: timestamppb.New(role.UpdatedAt)}, nil
}

type deleteRoleValidation struct {
	Id int32 `db:"id" validate:"required"`
}

// DeleteRole is a service for deleting a role
func (s *Server) DeleteRole(ctx context.Context,
	in *pb.DeleteRoleRequest) (*pb.DeleteRoleResponse, error) {
	validator := validator.New()
	err := validator.Struct(deleteRoleValidation{Id: in.Id})
	if err != nil {
		return nil, err
	}

	// first check if the role exists and is not protected
	role, err := s.store.GetRoleByID(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, role.GroupID) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	if in.Force == nil {
		isProtected := false
		in.Force = &isProtected
	}

	if !*in.Force && role.IsProtected {
		errcode := fmt.Errorf("cannot delete a protected role")
		return nil, errcode
	}

	// if we do not force the deletion, we need to check if there are users
	if !*in.Force {
		// list users belonging to that role
		users, err := s.store.ListUsersByRoleID(ctx, in.Id)
		if err != nil {
			return nil, err
		}

		if len(users) > 0 {
			errcode := fmt.Errorf("cannot delete the role, there are users associated with it")
			return nil, errcode
		}
	}

	// otherwise we delete, and delete users in cascade
	err = s.store.DeleteRole(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteRoleResponse{}, nil
}

// GetRoles is a service for getting roles
func (s *Server) GetRoles(ctx context.Context,
	in *pb.GetRolesRequest) (*pb.GetRolesResponse, error) {
	if in.GroupId == 0 {
		return nil, status.Error(codes.InvalidArgument, "group id is required")
	}

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

	roles, err := s.store.ListRoles(ctx, db.ListRolesParams{
		GroupID: in.GroupId,
		Limit:   *in.Limit,
		Offset:  *in.Offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get roles: %s", err)
	}

	var resp pb.GetRolesResponse
	resp.Roles = make([]*pb.RoleRecord, 0, len(roles))
	for _, role := range roles {
		resp.Roles = append(resp.Roles, &pb.RoleRecord{
			Id:          role.ID,
			GroupId:     role.GroupID,
			Name:        role.Name,
			IsAdmin:     role.IsAdmin,
			IsProtected: role.IsProtected,
			CreatedAt:   timestamppb.New(role.CreatedAt),
			UpdatedAt:   timestamppb.New(role.UpdatedAt),
		})
	}

	return &resp, nil
}

// GetRoleById is a service for getting a role by id
func (s *Server) GetRoleById(ctx context.Context,
	in *pb.GetRoleByIdRequest) (*pb.GetRoleByIdResponse, error) {
	if in.Id == 0 {
		return nil, status.Error(codes.InvalidArgument, "role id is required")
	}

	role, err := s.store.GetRoleByID(ctx, in.Id)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get role: %s", err)
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, role.GroupID) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	var resp pb.GetRoleByIdResponse
	resp.Role = &pb.RoleRecord{
		Id:          role.ID,
		GroupId:     role.GroupID,
		Name:        role.Name,
		IsAdmin:     role.IsAdmin,
		IsProtected: role.IsProtected,
		CreatedAt:   timestamppb.New(role.CreatedAt),
		UpdatedAt:   timestamppb.New(role.UpdatedAt),
	}

	return &resp, nil
}

// GetRoleByName is a service for getting a role by name
func (s *Server) GetRoleByName(ctx context.Context,
	in *pb.GetRoleByNameRequest) (*pb.GetRoleByNameResponse, error) {
	if in.GroupId == 0 {
		return nil, status.Error(codes.InvalidArgument, "group id is required")
	}
	if in.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "role name is required")
	}

	role, err := s.store.GetRoleByName(ctx, db.GetRoleByNameParams{GroupID: in.GroupId, Name: in.Name})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get role: %s", err)
	}

	var resp pb.GetRoleByNameResponse
	resp.Role = &pb.RoleRecord{
		Id:          role.ID,
		GroupId:     role.GroupID,
		Name:        role.Name,
		IsAdmin:     role.IsAdmin,
		IsProtected: role.IsProtected,
		CreatedAt:   timestamppb.New(role.CreatedAt),
		UpdatedAt:   timestamppb.New(role.UpdatedAt),
	}

	return &resp, nil
}
