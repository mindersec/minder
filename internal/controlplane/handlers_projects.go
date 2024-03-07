// Copyright 2024 Stacklok, Inc
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
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/projects"
	"github.com/stacklok/minder/internal/util"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ListProjects returns the list of projects for the current user
func (s *Server) ListProjects(
	ctx context.Context,
	_ *minderv1.ListProjectsRequest,
) (*minderv1.ListProjectsResponse, error) {
	userInfo, err := s.store.GetUserBySubject(ctx, auth.GetUserSubjectFromContext(ctx))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting user: %v", err)
	}

	projs, err := s.authzClient.ProjectsForUser(ctx, userInfo.IdentitySubject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting projects for user: %v", err)
	}

	resp := minderv1.ListProjectsResponse{}

	for _, projectID := range projs {
		project, err := s.store.GetProjectByID(ctx, projectID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error getting project: %v", err)
		}
		resp.Projects = append(resp.Projects, &minderv1.Project{
			ProjectId:   project.ID.String(),
			Name:        project.Name,
			Description: "",
			CreatedAt:   timestamppb.New(project.CreatedAt),
			UpdatedAt:   timestamppb.New(project.UpdatedAt),
		})
	}
	return &resp, nil
}

// CreateProject creates a new subproject
func (s *Server) CreateProject(
	ctx context.Context,
	req *minderv1.CreateProjectRequest,
) (*minderv1.CreateProjectResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	tx, err := s.store.BeginTransaction()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error starting transaction: %v", err)
	}
	defer s.store.Rollback(tx)

	qtx := s.store.GetQuerierWithTransaction(tx)

	parent, err := qtx.GetProjectByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "project not found")
		}
		return nil, status.Errorf(codes.Internal, "error getting project: %v", err)
	}

	// TODO: Remove this once we handle a full hierarchy
	if parent.ParentID.Valid {
		return nil, util.UserVisibleError(codes.InvalidArgument, "cannot create subproject of a subproject")
	}

	subProject, err := qtx.CreateProject(ctx, db.CreateProjectParams{
		Name: req.Name,
		ParentID: uuid.NullUUID{
			UUID:  parent.ID,
			Valid: true,
		},
		Metadata: json.RawMessage(`{}`),
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return nil, util.UserVisibleError(codes.AlreadyExists, "project named %s already exists", req.Name)
		}
		return nil, status.Errorf(codes.Internal, "error creating subproject: %v", err)
	}

	if err := s.authzClient.Adopt(ctx, parent.ID, subProject.ID); err != nil {
		return nil, status.Errorf(codes.Internal, "error creating subproject: %v", err)
	}

	if err := s.store.Commit(tx); err != nil {
		return nil, status.Errorf(codes.Internal, "error committing transaction: %v", err)
	}

	return &minderv1.CreateProjectResponse{
		Project: &minderv1.Project{
			ProjectId:   subProject.ID.String(),
			Name:        subProject.Name,
			Description: "",
			CreatedAt:   timestamppb.New(subProject.CreatedAt),
			UpdatedAt:   timestamppb.New(subProject.UpdatedAt),
		},
	}, nil
}

// DeleteProject deletes a subproject
func (s *Server) DeleteProject(
	ctx context.Context,
	_ *minderv1.DeleteProjectRequest,
) (*minderv1.DeleteProjectResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	tx, err := s.store.BeginTransaction()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error starting transaction: %v", err)
	}
	defer s.store.Rollback(tx)

	qtx := s.store.GetQuerierWithTransaction(tx)

	subProject, err := qtx.GetProjectByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "project not found")
		}
		return nil, status.Errorf(codes.Internal, "error getting project: %v", err)
	}

	if !subProject.ParentID.Valid {
		return nil, util.UserVisibleError(codes.InvalidArgument, "cannot delete a top-level project")
	}

	if err := projects.DeleteProject(ctx, projectID, qtx, s.authzClient); err != nil {
		return nil, status.Errorf(codes.Internal, "error deleting project: %v", err)
	}

	if err := s.store.Commit(tx); err != nil {
		return nil, status.Errorf(codes.Internal, "error committing transaction: %v", err)
	}

	return &minderv1.DeleteProjectResponse{
		ProjectId: projectID.String(),
	}, nil
}
