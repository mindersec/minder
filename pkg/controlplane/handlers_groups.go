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

	"github.com/go-playground/validator/v10"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type createGroupValidation struct {
	Name           string `db:"name" validate:"required"`
	OrganizationId int32  `db:"organization_id" validate:"required"`
}

// CreateGroup creates a group
func (s *Server) CreateGroup(ctx context.Context, req *pb.CreateGroupRequest) (*pb.CreateGroupResponse, error) {
	// validate that the org and name are not empty
	validator := validator.New()
	err := validator.Struct(createGroupValidation{OrganizationId: req.OrganizationId, Name: req.Name})
	if err != nil {
		return nil, err
	}

	_, err = s.store.GetGroupByName(ctx, req.Name)

	if err != nil && err != sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "failed to get group by name: %s", err)
	} else if err == nil {
		return nil, status.Errorf(codes.AlreadyExists, "group already exists")
	}

	if req.IsProtected == nil {
		isProtected := false
		req.IsProtected = &isProtected
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, req.OrganizationId) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	grp, err := s.store.CreateGroup(ctx, db.CreateGroupParams{
		OrganizationID: req.OrganizationId,
		Name:           req.Name,
		Description:    sql.NullString{String: req.Description, Valid: true},
		IsProtected:    *req.IsProtected,
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create group: %s", err)
	}

	resp := &pb.CreateGroupResponse{
		GroupId:        grp.ID,
		OrganizationId: grp.OrganizationID,
		Name:           grp.Name,
		Description:    grp.Description.String,
		IsProtected:    grp.IsProtected,
		CreatedAt:      timestamppb.New(grp.CreatedAt),
		UpdatedAt:      timestamppb.New(grp.UpdatedAt)}

	return resp, nil
}

// GetGroupById returns a group by id
func (s *Server) GetGroupById(ctx context.Context, req *pb.GetGroupByIdRequest) (*pb.GetGroupByIdResponse, error) {
	if req.GroupId == 0 {
		return nil, status.Error(codes.InvalidArgument, "group id cannot be empty")
	}

	grp, err := s.store.GetGroupByID(ctx, req.GroupId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to get group by id: %s", err)
	}

	// check if user is authorized
	if !IsRequestAuthorized(ctx, grp.OrganizationID) {
		return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
	}

	var resp pb.GetGroupByIdResponse
	resp.Group = &pb.GroupRecord{
		GroupId:        grp.ID,
		OrganizationId: grp.OrganizationID,
		Name:           grp.Name,
		Description:    grp.Description.String,
		IsProtected:    grp.IsProtected,
		CreatedAt:      timestamppb.New(grp.CreatedAt),
		UpdatedAt:      timestamppb.New(grp.UpdatedAt),
	}

	// check if user is authorized
	if IsRequestAuthorized(ctx, grp.OrganizationID) {
		return &resp, nil
	}
	return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
}

// GetGroupByName returns a group by name
func (s *Server) GetGroupByName(ctx context.Context, req *pb.GetGroupByNameRequest) (*pb.GetGroupByNameResponse, error) {
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "group name cannot be empty")
	}

	grp, err := s.store.GetGroupByName(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to get group by name: %s", err)
	}

	var resp pb.GetGroupByNameResponse
	resp.Group = &pb.GroupRecord{
		GroupId:        grp.ID,
		OrganizationId: grp.OrganizationID,
		Name:           grp.Name,
		Description:    grp.Description.String,
		IsProtected:    grp.IsProtected,
		CreatedAt:      timestamppb.New(grp.CreatedAt),
		UpdatedAt:      timestamppb.New(grp.UpdatedAt),
	}

	// check if user is authorized
	if IsRequestAuthorized(ctx, grp.OrganizationID) {
		return &resp, nil
	}
	return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
}

// GetGroups returns a list of groups
func (s *Server) GetGroups(ctx context.Context, req *pb.GetGroupsRequest) (*pb.GetGroupsResponse, error) {
	if req.OrganizationId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "organization id cannot be empty")
	}

	// define default values for limit and offset
	if req.Limit == -1 {
		req.Limit = PaginationLimit
	}

	grps, err := s.store.ListGroups(ctx, db.ListGroupsParams{
		OrganizationID: req.OrganizationId,
		Limit:          req.Limit,
		Offset:         req.Offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to get groups: %s", err)
	}

	var resp pb.GetGroupsResponse
	for _, group := range grps {
		resp.Groups = append(resp.Groups, &pb.GroupRecord{
			GroupId:        group.ID,
			OrganizationId: group.OrganizationID,
			Name:           group.Name,
			Description:    group.Description.String,
			IsProtected:    group.IsProtected,
			CreatedAt:      timestamppb.New(group.CreatedAt),
			UpdatedAt:      timestamppb.New(group.UpdatedAt),
		})
	}

	// check if user is authorized
	if IsRequestAuthorized(ctx, req.OrganizationId) {
		return &resp, nil
	}
	return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
}

type deleteGroupValidation struct {
	Id int32 `db:"id" validate:"required"`
}

// DeleteGroup is a handler that deletes a group
func (s *Server) DeleteGroup(ctx context.Context,
	in *pb.DeleteGroupRequest) (*pb.DeleteGroupResponse, error) {
	validator := validator.New()
	err := validator.Struct(deleteGroupValidation{Id: in.Id})
	if err != nil {
		return nil, err
	}

	// first check if the group exists and is not protected
	group, err := s.store.GetGroupByID(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	if in.Force == nil {
		isProtected := false
		in.Force = &isProtected
	}

	if !*in.Force && group.IsProtected {
		errcode := status.Errorf(codes.PermissionDenied, "cannot delete a protected group")
		return nil, errcode
	}

	// if we do not force the deletion, we need to check if there are roles
	if !*in.Force {
		// list roles belonging to that role
		roles, err := s.store.ListRolesByGroupID(ctx, db.ListRolesByGroupIDParams{GroupID: sql.NullInt32{Int32: group.ID, Valid: true}})
		if err != nil {
			return nil, err
		}

		if len(roles) > 0 {
			errcode := status.Errorf(codes.FailedPrecondition, "cannot delete the group, there are roles associated with it")
			return nil, errcode
		}
	}

	// check if group exists
	grp, err := s.store.GetGroupByID(ctx, in.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "group not found")
	}

	// check if user is authorized
	if IsRequestAuthorized(ctx, grp.OrganizationID) {
		err = s.store.DeleteGroup(ctx, in.Id)
		if err != nil {
			return nil, err
		}

		return &pb.DeleteGroupResponse{}, nil
	}
	return nil, status.Errorf(codes.PermissionDenied, "user is not authorized to access this resource")
}
