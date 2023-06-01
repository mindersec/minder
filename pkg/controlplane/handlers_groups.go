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

	"github.com/go-playground/validator/v10"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type createGroupValidation struct {
	Name           string `db:"name" validate:"required"`
	OrganisationId int32  `db:"organisation_id" validate:"required"`
}

// CreateGroup creates a group
func (s *Server) CreateGroup(ctx context.Context, req *pb.CreateGroupRequest) (*pb.CreateGroupResponse, error) {
	// validate that the org and name are not empty
	validator := validator.New()
	err := validator.Struct(createGroupValidation{OrganisationId: req.OrganisationId, Name: req.Name})
	if err != nil {
		return nil, err
	}

	_, err = s.store.GetGroupByName(ctx, req.Name)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get group by name: %w", err)
	} else if err == nil {
		return nil, fmt.Errorf("group already exists")
	}

	if req.IsProtected == nil {
		isProtected := false
		req.IsProtected = &isProtected
	}

	grp, err := s.store.CreateGroup(ctx, db.CreateGroupParams{
		OrganisationID: req.OrganisationId,
		Name:           req.Name,
		Description:    sql.NullString{String: req.Description, Valid: true},
		IsProtected:    *req.IsProtected,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}
	fmt.Println("Group ID: ", grp.ID)

	return &pb.CreateGroupResponse{
		GroupId:        grp.ID,
		OrganisationId: grp.OrganisationID,
		Name:           grp.Name,
		Description:    grp.Description.String,
		IsProtected:    grp.IsProtected,
		CreatedAt:      timestamppb.New(grp.CreatedAt),
		UpdatedAt:      timestamppb.New(grp.UpdatedAt)}, nil
}

// GetGroupById returns a group by id
func (s *Server) GetGroupById(ctx context.Context, req *pb.GetGroupByIdRequest) (*pb.GetGroupByIdResponse, error) {
	if req.GroupId == 0 {
		return nil, fmt.Errorf("group id cannot be empty")
	}

	grp, err := s.store.GetGroupByID(ctx, req.GroupId)
	if err != nil {
		return nil, fmt.Errorf("failed to get group by id: %w", err)
	}

	var resp pb.GetGroupByIdResponse
	resp.Group = &pb.GroupRecord{
		GroupId:        grp.ID,
		OrganisationId: grp.OrganisationID,
		Name:           grp.Name,
		Description:    grp.Description.String,
		IsProtected:    grp.IsProtected,
		CreatedAt:      timestamppb.New(grp.CreatedAt),
		UpdatedAt:      timestamppb.New(grp.UpdatedAt),
	}
	return &resp, nil
}

// GetGroupByName returns a group by name
func (s *Server) GetGroupByName(ctx context.Context, req *pb.GetGroupByNameRequest) (*pb.GetGroupByNameResponse, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("group name cannot be empty")
	}

	grp, err := s.store.GetGroupByName(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get group by name: %w", err)
	}

	var resp pb.GetGroupByNameResponse
	resp.Group = &pb.GroupRecord{
		GroupId:        grp.ID,
		OrganisationId: grp.OrganisationID,
		Name:           grp.Name,
		Description:    grp.Description.String,
		IsProtected:    grp.IsProtected,
		CreatedAt:      timestamppb.New(grp.CreatedAt),
		UpdatedAt:      timestamppb.New(grp.UpdatedAt),
	}
	return &resp, nil
}

// GetGroups returns a list of groups
func (s *Server) GetGroups(ctx context.Context, req *pb.GetGroupsRequest) (*pb.GetGroupsResponse, error) {
	if req.OrganisationId == 0 {
		return nil, fmt.Errorf("organisation id cannot be empty")
	}

	// define default values for limit and offset
	if req.Limit == -1 {
		req.Limit = PaginationLimit
	}

	grps, err := s.store.ListGroups(ctx, db.ListGroupsParams{
		OrganisationID: req.OrganisationId,
		Limit:          req.Limit,
		Offset:         req.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	var resp pb.GetGroupsResponse
	for _, group := range grps {
		resp.Groups = append(resp.Groups, &pb.GroupRecord{
			GroupId:        group.ID,
			OrganisationId: group.OrganisationID,
			Name:           group.Name,
			Description:    group.Description.String,
			IsProtected:    group.IsProtected,
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
		errcode := fmt.Errorf("cannot delete a protected group")
		return nil, errcode
	}

	// if we do not force the deletion, we need to check if there are roles
	if !*in.Force {
		// list roles belonging to that role
		roles, err := s.store.ListRolesByGroupID(ctx, in.Id)
		if err != nil {
			return nil, err
		}

		if len(roles) > 0 {
			errcode := fmt.Errorf("cannot delete the group, there are roles associated with it")
			return nil, errcode
		}
	}

	// otherwise we delete, and delete roles in cascade
	err = s.store.DeleteGroup(ctx, in.Id)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteGroupResponse{}, nil
}
