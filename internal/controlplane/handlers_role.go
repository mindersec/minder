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
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
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

	orgID, err := uuid.Parse(in.OrganizationId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid organization id")
	}

	// check if user is authorized
	if err := AuthorizedOnOrg(ctx, orgID); err != nil {
		return nil, err
	}

	// check that organization exists
	_, err = s.store.GetOrganization(ctx, orgID)
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

	roleParams := db.CreateRoleParams{OrganizationID: orgID,
		Name: in.Name, IsAdmin: *in.IsAdmin, IsProtected: *in.IsProtected}

	role, err := s.store.CreateRole(ctx, roleParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create role: %v", err)
	}

	return &pb.CreateRoleByOrganizationResponse{Id: role.ID, Name: role.Name,
		IsAdmin: role.IsAdmin, IsProtected: role.IsProtected,
		OrganizationId: role.OrganizationID.String(),
		CreatedAt:      timestamppb.New(role.CreatedAt),
		UpdatedAt:      timestamppb.New(role.UpdatedAt)}, nil
}

// CreateRoleByProject is a service for creating a role for a project
func (s *Server) CreateRoleByProject(ctx context.Context,
	in *pb.CreateRoleByProjectRequest) (*pb.CreateRoleByProjectResponse, error) {
	// validate that the company and name are not empty
	validator := validator.New()
	err := validator.Struct(CreateRoleValidation{Name: in.Name})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid argument: %v", err)
	}

	projID, err := uuid.Parse(in.ProjectId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid project id")
	}

	// check if user is authorized
	if err := AuthorizedOnProject(ctx, projID); err != nil {
		return nil, err
	}

	orgID, err := uuid.Parse(in.OrganizationId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid organization id")
	}

	// check that organization exists
	_, err = s.store.GetOrganization(ctx, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "organization not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get organization")
	}

	// check that project exists
	_, err = s.store.GetProjectByID(ctx, projID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "project not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get project by id: %s", err)
	}

	if in.IsAdmin == nil {
		isAdmin := false
		in.IsAdmin = &isAdmin
	}

	if in.IsProtected == nil {
		isProtected := false
		in.IsProtected = &isProtected
	}

	roleParams := db.CreateRoleParams{OrganizationID: orgID,
		Name: in.Name, IsAdmin: *in.IsAdmin, IsProtected: *in.IsProtected,
		ProjectID: uuid.NullUUID{UUID: projID, Valid: true}}

	role, err := s.store.CreateRole(ctx, roleParams)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create role: %v", err)
	}

	return &pb.CreateRoleByProjectResponse{Id: role.ID, Name: role.Name,
		IsAdmin: role.IsAdmin, IsProtected: role.IsProtected,
		OrganizationId: role.OrganizationID.String(),
		ProjectId:      role.ProjectID.UUID.String(),
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
	if err := AuthorizedOnOrg(ctx, role.OrganizationID); err != nil {
		return nil, err
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

	roles, err := s.store.ListRoles(ctx, db.ListRolesParams{
		OrganizationID: orgID,
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
		projID := role.ProjectID.UUID.String()
		resp.Roles = append(resp.Roles, &pb.RoleRecord{
			Id:             role.ID,
			OrganizationId: role.OrganizationID.String(),
			ProjectId:      &projID,
			Name:           role.Name,
			IsAdmin:        role.IsAdmin,
			IsProtected:    role.IsProtected,
			CreatedAt:      timestamppb.New(role.CreatedAt),
			UpdatedAt:      timestamppb.New(role.UpdatedAt),
		})
	}

	return &resp, nil
}

// GetRolesByProject is a service for getting roles for a projects
func (s *Server) GetRolesByProject(ctx context.Context,
	in *pb.GetRolesByProjectRequest) (*pb.GetRolesByProjectResponse, error) {
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

	roles, err := s.store.ListRolesByProjectID(ctx, db.ListRolesByProjectIDParams{
		ProjectID: uuid.NullUUID{UUID: projID, Valid: true},
		Limit:     *in.Limit,
		Offset:    *in.Offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get roles by project id: %v", err)
	}

	var resp pb.GetRolesByProjectResponse
	resp.Roles = make([]*pb.RoleRecord, 0, len(roles))
	for idx := range roles {
		role := &roles[idx]
		pID := role.ProjectID.UUID.String()
		resp.Roles = append(resp.Roles, &pb.RoleRecord{
			Id:             role.ID,
			OrganizationId: role.OrganizationID.String(),
			ProjectId:      &pID,
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
	if err := AuthorizedOnOrg(ctx, role.OrganizationID); err != nil {
		return nil, err
	}

	var resp pb.GetRoleByIdResponse
	projID := role.ProjectID.UUID.String()
	resp.Role = &pb.RoleRecord{
		Id:             role.ID,
		OrganizationId: role.OrganizationID.String(),
		ProjectId:      &projID,
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
	orgID, err := uuid.Parse(in.OrganizationId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid organization id")
	}

	// check if user is authorized
	if err := AuthorizedOnOrg(ctx, orgID); err != nil {
		return nil, err
	}

	if in.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "role name is required")
	}

	role, err := s.store.GetRoleByName(ctx, db.GetRoleByNameParams{OrganizationID: orgID, Name: in.Name})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "role not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get role by name: %v", err)
	}

	var resp pb.GetRoleByNameResponse
	projID := role.ProjectID.UUID.String()
	resp.Role = &pb.RoleRecord{
		Id:             role.ID,
		OrganizationId: role.OrganizationID.String(),
		ProjectId:      &projID,
		Name:           role.Name,
		IsAdmin:        role.IsAdmin,
		IsProtected:    role.IsProtected,
		CreatedAt:      timestamppb.New(role.CreatedAt),
		UpdatedAt:      timestamppb.New(role.UpdatedAt),
	}

	return &resp, nil
}
