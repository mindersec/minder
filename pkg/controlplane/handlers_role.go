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

	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// CreateRoleValidation is a struct for validating the CreateRole request
type CreateRoleValidation struct {
	Name string `db:"name" validate:"required"`
}

// CreateRoleByOrganization is a service for creating a role for an organization
func (s *Server) CreateRoleByOrganization(ctx context.Context,
	in *pb.CreateRoleByOrganizationRequest) (*pb.CreateRoleByOrganizationResponse, error) {
	// validate that the company and name are not empty
	validator := validator.New()
	err := validator.Struct(CreateRoleValidation{Name: in.Name})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid argument: %v", err)
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.OrganizationId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	// check that organization exists
	_, err = s.store.GetOrganization(ctx, in.OrganizationId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "organization not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get organization")
	}

	if in.IsAdmin == nil {
		isAdmin := false
		in.IsAdmin = &isAdmin
	}

	if in.IsProtected == nil {
		isProtected := false
		in.IsProtected = &isProtected
	}

	roleParams := db.CreateRoleParams{OrganizationID: in.OrganizationId,
		Name: in.Name, IsAdmin: *in.IsAdmin, IsProtected: *in.IsProtected}

	role, err := s.store.CreateRole(ctx, roleParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create role: %v", err)
	}

	return &pb.CreateRoleByOrganizationResponse{Id: role.ID, Name: role.Name,
		IsAdmin: role.IsAdmin, IsProtected: role.IsProtected,
		OrganizationId: role.OrganizationID,
		CreatedAt:      timestamppb.New(role.CreatedAt),
		UpdatedAt:      timestamppb.New(role.UpdatedAt)}, nil
}

// CreateRoleByGroup is a service for creating a role for a group
func (s *Server) CreateRoleByGroup(ctx context.Context,
	in *pb.CreateRoleByGroupRequest) (*pb.CreateRoleByGroupResponse, error) {
	// validate that the company and name are not empty
	validator := validator.New()
	err := validator.Struct(CreateRoleValidation{Name: in.Name})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid argument: %v", err)
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.GroupId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	// check that organization exists
	_, err = s.store.GetOrganization(ctx, in.OrganizationId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "organization not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get organization")
	}

	// check that group exists
	_, err = s.store.GetGroupByID(ctx, in.GroupId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "group not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get group by id: %s", err)
	}

	if in.IsAdmin == nil {
		isAdmin := false
		in.IsAdmin = &isAdmin
	}

	if in.IsProtected == nil {
		isProtected := false
		in.IsProtected = &isProtected
	}

	roleParams := db.CreateRoleParams{OrganizationID: in.OrganizationId,
		Name: in.Name, IsAdmin: *in.IsAdmin, IsProtected: *in.IsProtected,
		GroupID: sql.NullInt32{Int32: in.GroupId, Valid: true}}

	role, err := s.store.CreateRole(ctx, roleParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create role: %v", err)
	}

	return &pb.CreateRoleByGroupResponse{Id: role.ID, Name: role.Name,
		IsAdmin: role.IsAdmin, IsProtected: role.IsProtected,
		OrganizationId: role.OrganizationID,
		GroupId:        role.GroupID.Int32,
		CreatedAt:      timestamppb.New(role.CreatedAt),
		UpdatedAt:      timestamppb.New(role.UpdatedAt)}, nil
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
		return nil, status.Errorf(codes.InvalidArgument, "invalid argument: %v", err)
	}

	// first check if the role exists and is not protected
	role, err := s.store.GetRoleByID(ctx, in.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "role not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get role by id: %v", err)
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, role.OrganizationID) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	if in.Force == nil {
		isProtected := false
		in.Force = &isProtected
	}

	if !*in.Force && role.IsProtected {
		return nil, status.Errorf(codes.InvalidArgument, "cannot delete a protected role")
	}

	// if we do not force the deletion, we need to check if there are users
	if !*in.Force {
		// list users associated with that role
		users, err := s.store.ListUsersByRoleId(ctx, in.Id)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list users by role id: %v", err)
		}

		if len(users) > 0 {
			return nil, status.Errorf(codes.InvalidArgument, "cannot delete the role, there are users associated with it")
		}
	}

	// otherwise we delete, and delete association in cascade
	err = s.store.DeleteRole(ctx, in.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete role: %v", err)
	}

	return &pb.DeleteRoleResponse{}, nil
}

// GetRoles is a service for getting roles
func (s *Server) GetRoles(ctx context.Context,
	in *pb.GetRolesRequest) (*pb.GetRolesResponse, error) {
	if in.OrganizationId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid argument: organization id is required")
	}

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

	roles, err := s.store.ListRoles(ctx, db.ListRolesParams{
		OrganizationID: in.OrganizationId,
		Limit:          *in.Limit,
		Offset:         *in.Offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get roles: %s", err)
	}

	var resp pb.GetRolesResponse
	resp.Roles = make([]*pb.RoleRecord, 0, len(roles))
	for idx := range roles {
		role := &roles[idx]
		resp.Roles = append(resp.Roles, &pb.RoleRecord{
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

	return &resp, nil
}

// GetRolesByGroup is a service for getting roles for a group
func (s *Server) GetRolesByGroup(ctx context.Context,
	in *pb.GetRolesByGroupRequest) (*pb.GetRolesByGroupResponse, error) {
	if in.GroupId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid argument: group id is required")
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

	roles, err := s.store.ListRolesByGroupID(ctx, db.ListRolesByGroupIDParams{
		GroupID: sql.NullInt32{Int32: in.GroupId, Valid: true},
		Limit:   *in.Limit,
		Offset:  *in.Offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get roles by group id: %v", err)
	}

	var resp pb.GetRolesByGroupResponse
	resp.Roles = make([]*pb.RoleRecord, 0, len(roles))
	for idx := range roles {
		role := &roles[idx]
		resp.Roles = append(resp.Roles, &pb.RoleRecord{
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "role not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get role: %v", err)
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, role.OrganizationID) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	var resp pb.GetRoleByIdResponse
	resp.Role = &pb.RoleRecord{
		Id:             role.ID,
		OrganizationId: role.OrganizationID,
		GroupId:        &role.GroupID.Int32,
		Name:           role.Name,
		IsAdmin:        role.IsAdmin,
		IsProtected:    role.IsProtected,
		CreatedAt:      timestamppb.New(role.CreatedAt),
		UpdatedAt:      timestamppb.New(role.UpdatedAt),
	}

	return &resp, nil
}

// GetRoleByName is a service for getting a role by name
func (s *Server) GetRoleByName(ctx context.Context,
	in *pb.GetRoleByNameRequest) (*pb.GetRoleByNameResponse, error) {
	// check if user is authorized
	if !IsRequestAuthorized(ctx, in.OrganizationId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	if in.OrganizationId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "organization id is required")
	}
	if in.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "role name is required")
	}

	role, err := s.store.GetRoleByName(ctx, db.GetRoleByNameParams{OrganizationID: in.OrganizationId, Name: in.Name})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "role not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get role by name: %v", err)
	}

	var resp pb.GetRoleByNameResponse
	resp.Role = &pb.RoleRecord{
		Id:             role.ID,
		OrganizationId: role.OrganizationID,
		GroupId:        &role.GroupID.Int32,
		Name:           role.Name,
		IsAdmin:        role.IsAdmin,
		IsProtected:    role.IsProtected,
		CreatedAt:      timestamppb.New(role.CreatedAt),
		UpdatedAt:      timestamppb.New(role.UpdatedAt),
	}

	return &resp, nil
}
