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
	"encoding/json"
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

// ProjectMeta is the metadata associated with a project
type ProjectMeta struct {
	Description string `json:"description"`
	IsProtected bool   `json:"is_protected"`
}

type createProjectValidation struct {
	Name           string `db:"name" validate:"required"`
	OrganizationId string `db:"organization_id" validate:"required"`
}

func getProjectDependencies(ctx context.Context, store db.Store, project db.Project) ([]*pb.RoleRecord, []*pb.UserRecord, error) {
	const MAX_ITEMS = 999

	// get all the roles associated with that project
	roles, err := store.ListRolesByProjectID(ctx, db.ListRolesByProjectIDParams{
		ProjectID: uuid.NullUUID{UUID: project.ID, Valid: true},
		Limit:     MAX_ITEMS, Offset: 0})
	if err != nil {
		return nil, nil, err
	}

	// get all the users associated with that projects
	users, err := store.ListUsersByProject(ctx, db.ListUsersByProjectParams{
		ProjectID: project.ID, Limit: MAX_ITEMS, Offset: 0})
	if err != nil {
		return nil, nil, err
	}

	// convert to right data type
	var rolesPB []*pb.RoleRecord
	for idx := range roles {
		role := &roles[idx]
		pID := role.ProjectID.UUID.String()
		rolesPB = append(rolesPB, &pb.RoleRecord{
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

	var usersPB []*pb.UserRecord
	for idx := range users {
		user := &users[idx]
		usersPB = append(usersPB, &pb.UserRecord{
			Id:              user.ID,
			OrganizationId:  user.OrganizationID.String(),
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

// CreateProject creates a project
func (s *Server) CreateProject(ctx context.Context, req *pb.CreateProjectRequest) (*pb.CreateProjectResponse, error) {
	// validate that the org and name are not empty
	validator := validator.New()
	err := validator.Struct(createProjectValidation{OrganizationId: req.OrganizationId, Name: req.Name})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %s", err)
	}

	if req.IsProtected == nil {
		isProtected := false
		req.IsProtected = &isProtected
	}

	orgID, err := uuid.Parse(req.OrganizationId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid organization ID")
	}

	// check if user is authorized
	if err := AuthorizedOnOrg(ctx, orgID); err != nil {
		return nil, err
	}

	projmeta := &ProjectMeta{
		Description: req.Description,
		IsProtected: *req.IsProtected,
	}

	projMetaJSON, err := json.Marshal(projmeta)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal project metadata: %s", err)
	}

	prj, err := s.store.CreateProject(ctx, db.CreateProjectParams{
		ParentID: uuid.NullUUID{
			UUID:  orgID,
			Valid: true,
		},
		Name:     req.Name,
		Metadata: projMetaJSON,
	})

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create project: %s", err)
	}

	resp := &pb.CreateProjectResponse{
		ProjectId:      prj.ID.String(),
		OrganizationId: prj.ParentID.UUID.String(),
		Name:           prj.Name,
		Description:    req.Description,
		IsProtected:    *req.IsProtected,
		CreatedAt:      timestamppb.New(prj.CreatedAt),
		UpdatedAt:      timestamppb.New(prj.UpdatedAt)}

	return resp, nil
}

// GetProjectById returns a project by id
func (s *Server) GetProjectById(ctx context.Context, req *pb.GetProjectByIdRequest) (*pb.GetProjectByIdResponse, error) {
	projID, err := uuid.Parse(req.ProjectId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "malformed project ID")
	}

	prj, err := s.store.GetProjectByID(ctx, projID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "project not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get project by id: %s", err)
	}

	// check if user is authorized
	if err := AuthorizedOnOrg(ctx, prj.ParentID.UUID); err != nil {
		return nil, err
	}

	roles, users, err := getProjectDependencies(ctx, s.store, prj)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get project dependencies: %s", err)
	}

	var resp pb.GetProjectByIdResponse
	resp.Project = &pb.ProjectRecord{
		ProjectId:      prj.ID.String(),
		OrganizationId: prj.ParentID.UUID.String(),
		Name:           prj.Name,
		CreatedAt:      timestamppb.New(prj.CreatedAt),
		UpdatedAt:      timestamppb.New(prj.UpdatedAt),
	}
	resp.Roles = roles
	resp.Users = users

	return &resp, nil
}

// GetProjectByName returns a projects by name
func (s *Server) GetProjectByName(ctx context.Context, req *pb.GetProjectByNameRequest) (*pb.GetProjectByNameResponse, error) {
	if req.Name == "" {
		return nil, status.Errorf(codes.InvalidArgument, "project name cannot be empty")
	}

	prj, err := s.store.GetProjectByName(ctx, req.Name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to get project by name: %s", err)
	}

	// check if user is authorized
	if err := AuthorizedOnOrg(ctx, prj.ParentID.UUID); err != nil {
		return nil, err
	}

	roles, users, err := getProjectDependencies(ctx, s.store, prj)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get organization dependencies: %s", err)
	}

	var resp pb.GetProjectByNameResponse
	resp.Project = &pb.ProjectRecord{
		ProjectId:      prj.ID.String(),
		OrganizationId: prj.ParentID.UUID.String(),
		Name:           prj.Name,
		CreatedAt:      timestamppb.New(prj.CreatedAt),
		UpdatedAt:      timestamppb.New(prj.UpdatedAt),
	}

	resp.Roles = roles
	resp.Users = users

	return &resp, nil

}

// GetProjects returns a list of projects
func (s *Server) GetProjects(ctx context.Context, req *pb.GetProjectsRequest) (*pb.GetProjectsResponse, error) {
	// define default values for limit and offset
	if req.Limit == -1 {
		req.Limit = PaginationLimit
	}

	orgID, err := uuid.Parse(req.OrganizationId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid organization ID")
	}

	// check if user is authorized
	if err := AuthorizedOnOrg(ctx, orgID); err != nil {
		return nil, err
	}

	prj, err := s.store.GetChildrenProjects(ctx, orgID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to get projects: %s", err)
	}

	// NOTE(jaosorior): We need to do this because the database query returns
	// the calling project as well. We need to remove it from the list.
	prj = prj[db.CalculateProjectHierarchyOffset(0):]

	var resp pb.GetProjectsResponse
	for _, project := range prj {
		resp.Projects = append(resp.Projects, &pb.ProjectRecord{
			ProjectId:      project.ID.String(),
			OrganizationId: project.ParentID.UUID.String(),
			Name:           project.Name,
			CreatedAt:      timestamppb.New(project.CreatedAt),
			UpdatedAt:      timestamppb.New(project.UpdatedAt),
		})
	}

	return &resp, nil
}

type deleteProjectValidation struct {
	Id string `db:"id" validate:"required"`
}

// DeleteProject is a handler that deletes a project
func (s *Server) DeleteProject(ctx context.Context,
	in *pb.DeleteProjectRequest) (*pb.DeleteProjectResponse, error) {
	validator := validator.New()
	err := validator.Struct(deleteProjectValidation{Id: in.Id})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %s", err)
	}

	projID, err := uuid.Parse(in.Id)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "malformed project ID")
	}

	// first check if the project exists and is not protected
	project, err := s.store.GetProjectByID(ctx, projID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "project not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get project by id: %s", err)
	}

	if in.Force == nil {
		isProtected := false
		in.Force = &isProtected
	}

	// TODO FIX THIS
	// if !*in.Force && projects.IsProtected {
	// 	return nil, status.Errorf(codes.PermissionDenied, "cannot delete a protected projects")
	// }

	// if we do not force the deletion, we need to check if there are roles
	if !*in.Force {
		// list roles belonging to that role
		roles, err := s.store.ListRolesByProjectID(ctx, db.ListRolesByProjectIDParams{
			ProjectID: uuid.NullUUID{UUID: project.ID, Valid: true}})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list roles by projects id: %s", err)
		}

		if len(roles) > 0 {
			return nil, status.Errorf(codes.FailedPrecondition, "cannot delete the projects, there are roles associated with it")
		}
	}

	// check if user is authorized on the org
	if err := AuthorizedOnOrg(ctx, project.ParentID.UUID); err != nil {
		return nil, err
	}
	_, err = s.store.DeleteProject(ctx, projID)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteProjectResponse{}, nil
}
