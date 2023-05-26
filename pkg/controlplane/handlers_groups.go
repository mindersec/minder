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

	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func (s *Server) CreateGroup(ctx context.Context, req *pb.CreateGroupRequest) (*pb.GroupsResponse, error) {
	fmt.Println("Group Name: ", req.Name)
	fmt.Println("Organization ID: ", req.OrganisationId)
	fmt.Println("Description: ", req.Description)

	_, err := s.store.GetGroupByName(ctx, req.Name)
	// check if group already exists
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get group by name: %w", err)
	} else if err == nil {
		return nil, fmt.Errorf("group already exists")
	}

	organizationID := getOrganizationID(req.OrganisationId)

	grp, err := s.store.CreateGroup(ctx, db.CreateGroupParams{
		OrganisationID: organizationID,
		Name:           req.Name,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}
	fmt.Println("Group ID: ", grp.ID)

	return &pb.GroupsResponse{
		GroupId: grp.ID,
		Name:    grp.Name,
	}, nil
}

func (s *Server) GetGroupById(ctx context.Context, req *pb.GetGroupByIdRequest) (*pb.GroupsResponse, error) {
	fmt.Println("Group ID: ", req.GroupId)
	grp, err := s.store.GetGroupByID(ctx, req.GroupId)
	if err != nil {
		return nil, fmt.Errorf("failed to get group by id: %w", err)
	}

	return &pb.GroupsResponse{
		GroupId:     grp.ID,
		Name:        grp.Name,
		Description: grp.Description.String,
	}, nil
}

func (s *Server) GetGroupByName(ctx context.Context, req *pb.GetGroupByNameRequest) (*pb.GroupsResponse, error) {
	grp, err := s.store.GetGroupByName(ctx, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get group by id: %w", err)
	}

	return &pb.GroupsResponse{
		GroupId: grp.ID,
	}, nil
}

func (s *Server) GetGroups(ctx context.Context, req *pb.GetGroupsRequest) (*pb.GetGroupsResponse, error) {

	organizationID := getOrganizationID(req.OrganisationId)

	grps, err := s.store.ListGroups(ctx, db.ListGroupsParams{
		OrganisationID: organizationID,
		Limit:          req.Limit,
		Offset:         req.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	var resp pb.GetGroupsResponse
	for _, group := range grps {
		resp.Groups = append(resp.Groups, &pb.GroupsResponse{
			GroupId: group.ID,
			Name:    group.Name,
		})
	}

	return &resp, nil
}

func getOrganizationID(reqOrganizationID int32) sql.NullInt32 {
	var organizationID sql.NullInt32

	if reqOrganizationID != 0 {
		organizationID.Int32 = int32(reqOrganizationID)
		organizationID.Valid = true
	}

	return organizationID
}
