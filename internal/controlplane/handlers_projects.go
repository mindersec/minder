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

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/projects"
	"github.com/stacklok/minder/internal/projects/features"
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
			// project was deleted while we were iterating
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, status.Errorf(codes.Internal, "error getting project: %v", err)
		}

		var description, displayName string
		meta, err := projects.ParseMetadata(&project)
		// ignore error if we can't parse the metadata. This information is not critical... yet.
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to parse metadata")
			description = ""
			displayName = project.Name
		} else {
			description = meta.Public.Description
			displayName = meta.Public.DisplayName
		}

		resp.Projects = append(resp.Projects, &minderv1.Project{
			ProjectId:   project.ID.String(),
			Name:        project.Name,
			Description: description,
			DisplayName: displayName,
			CreatedAt:   timestamppb.New(project.CreatedAt),
			UpdatedAt:   timestamppb.New(project.UpdatedAt),
		})
	}
	return &resp, nil
}

// ListChildProjects returns the list of subprojects for the current project
func (s *Server) ListChildProjects(
	ctx context.Context,
	req *minderv1.ListChildProjectsRequest,
) (*minderv1.ListChildProjectsResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	var projs []*minderv1.Project
	var err error

	if req.Recursive {
		projs, err = s.getChildProjects(ctx, projectID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error getting subprojects: %v", err)
		}
	} else {
		projs, err = s.getImmediateChildrenProjects(ctx, projectID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "error getting subprojects: %v", err)
		}
	}

	resp := minderv1.ListChildProjectsResponse{
		Projects: projs,
	}
	return &resp, nil
}

func (s *Server) getChildProjects(ctx context.Context, projectID uuid.UUID) ([]*minderv1.Project, error) {
	projs, err := s.store.GetChildrenProjects(ctx, projectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting subprojects: %v", err)
	}

	out := make([]*minderv1.Project, 0, len(projs))
	for _, project := range projs {
		out = append(out, &minderv1.Project{
			ProjectId:   project.ID.String(),
			Name:        project.Name,
			Description: "",
			// TODO: We need to agree on how to handle metadata for subprojects
			DisplayName: project.Name,
			CreatedAt:   timestamppb.New(project.CreatedAt),
			UpdatedAt:   timestamppb.New(project.UpdatedAt),
		})
	}

	return out, nil
}

func (s *Server) getImmediateChildrenProjects(ctx context.Context, projectID uuid.UUID) ([]*minderv1.Project, error) {
	projs, err := s.store.GetImmediateChildrenProjects(ctx, projectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting subprojects: %v", err)
	}

	out := make([]*minderv1.Project, 0, len(projs))
	for _, project := range projs {
		out = append(out, &minderv1.Project{
			ProjectId:   project.ID.String(),
			Name:        project.Name,
			Description: "",
			// TODO: We need to agree on how to handle metadata for subprojects
			DisplayName: project.Name,
			CreatedAt:   timestamppb.New(project.CreatedAt),
			UpdatedAt:   timestamppb.New(project.UpdatedAt),
		})
	}

	return out, nil
}

// CreateProject creates a new subproject
func (s *Server) CreateProject(
	ctx context.Context,
	req *minderv1.CreateProjectRequest,
) (*minderv1.CreateProjectResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	if !features.ProjectAllowsProjectHierarchyOperations(ctx, s.store, projectID) {
		return nil, util.UserVisibleError(codes.PermissionDenied,
			"project does not allow project hierarchy operations")
	}

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

	if err := projects.ValidateName(req.Name); err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid project name: %v", err)
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
		if db.ErrIsUniqueViolation(err) {
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

	// The parent is supposed to have the feature flag, not the subproject
	if !features.ProjectAllowsProjectHierarchyOperations(ctx, s.store, subProject.ParentID.UUID) {
		return nil, util.UserVisibleError(codes.PermissionDenied,
			"project does not allow project hierarchy operations")
	}

	if err := s.projectDeleter.DeleteProject(ctx, projectID, qtx); err != nil {
		return nil, status.Errorf(codes.Internal, "error deleting project: %v", err)
	}

	if err := s.store.Commit(tx); err != nil {
		return nil, status.Errorf(codes.Internal, "error committing transaction: %v", err)
	}

	return &minderv1.DeleteProjectResponse{
		ProjectId: projectID.String(),
	}, nil
}

// UpdateProject updates a project. Note that this does not reparent nor
// touches the project's metadata directly. There is only a subset of
// fields that can be updated.
func (s *Server) UpdateProject(
	ctx context.Context,
	req *minderv1.UpdateProjectRequest,
) (*minderv1.UpdateProjectResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	tx, err := s.store.BeginTransaction()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error starting transaction: %v", err)
	}
	defer s.store.Rollback(tx)

	qtx := s.store.GetQuerierWithTransaction(tx)

	project, err := qtx.GetProjectByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "project not found")
		}
		return nil, status.Errorf(codes.Internal, "error getting project: %v", err)
	}

	meta, err := projects.ParseMetadata(&project)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error parsing metadata: %v", err)
	}

	if req.GetDisplayName() != "" {
		meta.Public.DisplayName = req.GetDisplayName()
	} else {
		// Display name cannot be empty, it will
		// default to the project name.
		meta.Public.DisplayName = project.Name
	}

	meta.Public.Description = req.GetDescription()

	serialized, err := projects.SerializeMetadata(meta)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error serializing metadata: %v", err)
	}

	outproj, err := qtx.UpdateProjectMeta(ctx, db.UpdateProjectMetaParams{
		ID:       project.ID,
		Metadata: serialized,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error updating project: %v", err)
	}

	if err := s.store.Commit(tx); err != nil {
		return nil, status.Errorf(codes.Internal, "error committing transaction: %v", err)
	}

	return &minderv1.UpdateProjectResponse{
		Project: &minderv1.Project{
			ProjectId:   outproj.ID.String(),
			Name:        outproj.Name,
			Description: meta.Public.Description,
			DisplayName: meta.Public.DisplayName,
			CreatedAt:   timestamppb.New(outproj.CreatedAt),
			UpdatedAt:   timestamppb.New(outproj.UpdatedAt),
		},
	}, nil
}

// PatchProject patches a project. Note that this does not reparent nor
// touches the project's metadata directly. There is only a subset of
// fields that can be updated.
func (s *Server) PatchProject(
	ctx context.Context,
	req *minderv1.PatchProjectRequest,
) (*minderv1.PatchProjectResponse, error) {
	entityCtx := engine.EntityFromContext(ctx)
	projectID := entityCtx.Project.ID

	tx, err := s.store.BeginTransaction()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error starting transaction: %v", err)
	}
	defer s.store.Rollback(tx)

	qtx := s.store.GetQuerierWithTransaction(tx)

	project, err := qtx.GetProjectByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "project not found")
		}
		return nil, status.Errorf(codes.Internal, "error getting project: %v", err)
	}

	meta, err := projects.ParseMetadata(&project)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error parsing metadata: %v", err)
	}

	req.GetUpdateMask().Normalize()
	for _, path := range req.GetUpdateMask().GetPaths() {
		switch path {
		case "display_name":
			meta.Public.DisplayName = req.GetPatch().GetDisplayName()
		case "description":
			meta.Public.Description = req.GetPatch().GetDescription()
		}
	}

	serialized, err := projects.SerializeMetadata(meta)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error serializing metadata: %v", err)
	}

	outproj, err := qtx.UpdateProjectMeta(ctx, db.UpdateProjectMetaParams{
		ID:       project.ID,
		Metadata: serialized,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error updating project: %v", err)
	}

	if err := s.store.Commit(tx); err != nil {

		return nil, status.Errorf(codes.Internal, "error committing transaction: %v", err)
	}

	return &minderv1.PatchProjectResponse{
		Project: &minderv1.Project{
			ProjectId:   outproj.ID.String(),
			Name:        outproj.Name,
			Description: meta.Public.Description,
			DisplayName: meta.Public.DisplayName,
			CreatedAt:   timestamppb.New(outproj.CreatedAt),
			UpdatedAt:   timestamppb.New(outproj.UpdatedAt),
		},
	}, nil
}
