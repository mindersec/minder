// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package querier provides tools to interact with the Minder database
package querier

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mindersec/minder/internal/projects"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// ProjectHandlers interface provides functions to interact with projects
type ProjectHandlers interface {
	GetProjectByID(ctx context.Context, id uuid.UUID) (*pb.Project, error)
	ListAllParentProjects(ctx context.Context) ([]*pb.Project, error)
}

// GetProjectByID returns a project by ID
func (t *Type) GetProjectByID(ctx context.Context, id uuid.UUID) (*pb.Project, error) {
	ret, err := t.db.querier.GetProjectByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Try to parse the project metadata to complete the response fields
	pDisplay := ret.Name
	pDescription := ""
	meta, err := projects.ParseMetadata(&ret)
	if err == nil {
		pDisplay = meta.Public.DisplayName
		pDescription = meta.Public.Description
	}
	parentID := ""
	if ret.ParentID.Valid {
		parentID = ret.ParentID.UUID.String()
	}
	return &pb.Project{
		ProjectId:      ret.ID.String(),
		Name:           ret.Name,
		CreatedAt:      timestamppb.New(ret.CreatedAt),
		UpdatedAt:      timestamppb.New(ret.UpdatedAt),
		DisplayName:    pDisplay,
		Description:    pDescription,
		ParentId:       parentID,
		IsOrganization: ret.IsOrganization,
	}, nil
}

// ListAllParentProjects returns all parent projects
func (t *Type) ListAllParentProjects(ctx context.Context) ([]*pb.Project, error) {
	ret, err := t.db.querier.ListAllParentProjects(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*pb.Project, len(ret))
	for i, p := range ret {
		// Try to parse the project metadata to complete the response fields
		pDisplay := p.Name
		pDescription := ""
		meta, err := projects.ParseMetadata(&p)
		if err == nil {
			pDisplay = meta.Public.DisplayName
			pDescription = meta.Public.Description
		}
		parentID := ""
		if p.ParentID.Valid {
			parentID = p.ParentID.UUID.String()
		}
		result[i] = &pb.Project{
			ProjectId:      p.ID.String(),
			Name:           p.Name,
			CreatedAt:      timestamppb.New(p.CreatedAt),
			UpdatedAt:      timestamppb.New(p.UpdatedAt),
			DisplayName:    pDisplay,
			Description:    pDescription,
			ParentId:       parentID,
			IsOrganization: p.IsOrganization,
		}
	}
	return result, nil
}
