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

	"github.com/stacklok/mediator/internal/db"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

type createGroupValidation struct {
	Name           string `db:"name" validate:"required"`
	OrganizationId int32  `db:"organization_id" validate:"required"`
}

func getGroupDependencies(ctx context.Context, store db.Store, group db.Group) ([]*pb.RoleRecord, []*pb.UserRecord, error) {
	const MAX_ITEMS = 999

	// get all the roles associated with that group
	roles, err := store.ListRolesByGroupID(ctx, db.ListRolesByGroupIDParams{GroupID: sql.NullInt32{Int32: group.ID, Valid: true},
		Limit: MAX_ITEMS, Offset: 0})
	if err != nil {
		return nil, nil, err
	}

	// get all the users associated with that group
	users, err := store.ListUsersByGroup(ctx, db.ListUsersByGroupParams{GroupID: group.ID, Limit: MAX_ITEMS, Offset: 0})
	if err != nil {
		return nil, nil, err
	}

	// convert to right data type
	var rolesPB []*pb.RoleRecord
	for idx := range roles {
		role := &roles[idx]
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

	var usersPB []*pb.UserRecord
	for idx := range users {
		user := &users[idx]
		usersPB = append(usersPB, &pb.UserRecord{
			Id:              user.ID,
			OrganizationId:  user.OrganizationID,
			Email:           &user.Email.String,
			FirstName:       &user.FirstName.String,
			LastName:        &user.LastName.String,
			IdentitySubject: user.IdentitySubject,
			CreatedAt:       timestamppb.New(user.CreatedAt),
			UpdatedAt:       timestamppb.New(user.UpdatedAt),
		})
	}

	return rolesPB, usersPB, nil
}

// CreateGroup creates a group
func (s *Server) CreateGroup(ctx context.Context, req *pb.CreateGroupRequest) (*pb.CreateGroupResponse, error) {
	// validate that the org and name are not empty
	validator := validator.New()
	err := validator.Struct(createGroupValidation{OrganizationId: req.OrganizationId, Name: req.Name})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %s", err)
	}

	if req.IsProtected == nil {
		isProtected := false
		req.IsProtected = &isProtected
	}

	// check if user is authorized
	if err := AuthorizedOnOrg(ctx, req.OrganizationId); err != nil {
		return nil, err
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "group not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get group by id: %s", err)
	}

	// check if user is authorized
	if err := AuthorizedOnOrg(ctx, grp.OrganizationID); err != nil {
		return nil, err
	}

	roles, users, err := getGroupDependencies(ctx, s.store, grp)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get group dependencies: %s", err)
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
	resp.Roles = roles
	resp.Users = users

	return &resp, nil
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

	// check if user is authorized
	if err := AuthorizedOnOrg(ctx, grp.OrganizationID); err != nil {
		return nil, err
	}

	roles, users, err := getGroupDependencies(ctx, s.store, grp)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get organization dependencies: %s", err)
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

	resp.Roles = roles
	resp.Users = users

	return &resp, nil

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

	// check if user is authorized
	if err := AuthorizedOnOrg(ctx, req.OrganizationId); err != nil {
		return nil, err
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

	return &resp, nil
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
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %s", err)
	}

	// first check if the group exists and is not protected
	group, err := s.store.GetGroupByID(ctx, in.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "group not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get group by id: %s", err)
	}

	if in.Force == nil {
		isProtected := false
		in.Force = &isProtected
	}

	if !*in.Force && group.IsProtected {
		return nil, status.Errorf(codes.PermissionDenied, "cannot delete a protected group")
	}

	// if we do not force the deletion, we need to check if there are roles
	if !*in.Force {
		// list roles belonging to that role
		roles, err := s.store.ListRolesByGroupID(ctx, db.ListRolesByGroupIDParams{GroupID: sql.NullInt32{Int32: group.ID, Valid: true}})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list roles by group id: %s", err)
		}

		if len(roles) > 0 {
			return nil, status.Errorf(codes.FailedPrecondition, "cannot delete the group, there are roles associated with it")
		}
	}

	// check if user is authorized on the org
	if err := AuthorizedOnOrg(ctx, group.OrganizationID); err != nil {
		return nil, err
	}
	err = s.store.DeleteGroup(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteGroupResponse{}, nil
}
