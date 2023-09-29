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
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/providers/github"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// OrgMeta is the metadata associated with an organization
type OrgMeta struct {
	Company string `json:"company"`
}

type createOrganizationValidation struct {
	Name    string `db:"name" validate:"required"`
	Company string `db:"company" validate:"required"`
}

// CreateOrganization is a service for creating an organization
// nolint:gocyclo // we should reactor this later.
func (s *Server) CreateOrganization(ctx context.Context,
	in *pb.CreateOrganizationRequest) (*pb.CreateOrganizationResponse, error) {
	// validate that the company and name are not empty
	validator := validator.New()
	err := validator.Struct(createOrganizationValidation{Name: in.Name, Company: in.Company})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid argument: %v", err)
	}

	tx, err := s.store.BeginTransaction()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction")
	}
	defer s.store.Rollback(tx)
	qtx := s.store.GetQuerierWithTransaction(tx)
	if qtx == nil {
		return nil, status.Errorf(codes.Internal, "failed to get transaction")
	}

	meta := OrgMeta{Company: in.Company}
	jsonmeta, err := json.Marshal(meta)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal meta: %v", err)
	}

	org, err := qtx.CreateOrganization(ctx, db.CreateOrganizationParams{Name: in.Name, Metadata: jsonmeta})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create organization: %v", err)
	}

	response := &pb.CreateOrganizationResponse{Id: org.ID.String(), Name: org.Name,
		Company: meta.Company, CreatedAt: timestamppb.New(org.CreatedAt),
		UpdatedAt: timestamppb.New(org.UpdatedAt)}

	if in.CreateDefaultRecords {
		// we need to create the default records for the organization
		defaultProject, defaultRoles, err := CreateDefaultRecordsForOrg(ctx, qtx, org)
		if err != nil {
			return nil, err
		}
		response.DefaultProject = defaultProject
		response.DefaultRoles = defaultRoles
	}
	// commit and return
	err = s.store.Commit(tx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction")
	}

	return response, nil
}

// CreateDefaultRecordsForOrg creates the default records, such as projects, roles and provider for the organization
func CreateDefaultRecordsForOrg(ctx context.Context, qtx db.Querier,
	org db.Project) (*pb.ProjectRecord, []*pb.RoleRecord, error) {
	projectmeta := &ProjectMeta{
		IsProtected: true,
		Description: fmt.Sprintf("Default admin project for %s", org.Name),
	}

	jsonmeta, err := json.Marshal(projectmeta)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to marshal meta: %v", err)
	}

	// we need to create the default records for the organization
	project, err := qtx.CreateProject(ctx, db.CreateProjectParams{
		ParentID: uuid.NullUUID{
			UUID:  org.ID,
			Valid: true,
		},
		Name:     fmt.Sprintf("%s-admin", org.Name),
		Metadata: jsonmeta,
	})
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to create default project: %v", err)
	}

	prj := pb.ProjectRecord{
		ProjectId:      project.ID.String(),
		OrganizationId: project.ParentID.UUID.String(),
		Name:           project.Name,
		Description:    projectmeta.Description,
		IsProtected:    projectmeta.IsProtected,
		CreatedAt:      timestamppb.New(project.CreatedAt),
		UpdatedAt:      timestamppb.New(project.UpdatedAt),
	}

	// we can create the default role for org and for project
	role, err := qtx.CreateRole(ctx, db.CreateRoleParams{
		OrganizationID: org.ID,
		Name:           fmt.Sprintf("%s-org-admin", org.Name),
		IsAdmin:        true,
		IsProtected:    true,
	})
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to create default org role: %v", err)
	}

	roleProject, err := qtx.CreateRole(ctx, db.CreateRoleParams{
		OrganizationID: org.ID,
		ProjectID:      uuid.NullUUID{UUID: project.ID, Valid: true},
		Name:           fmt.Sprintf("%s-project-admin", org.Name),
		IsAdmin:        true,
		IsProtected:    true,
	})

	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to create default project role: %v", err)
	}

	pID := roleProject.ProjectID.UUID.String()
	rl := pb.RoleRecord{Id: role.ID, OrganizationId: role.OrganizationID.String(), Name: role.Name, IsAdmin: role.IsAdmin,
		IsProtected: role.IsProtected, CreatedAt: timestamppb.New(role.CreatedAt), UpdatedAt: timestamppb.New(role.UpdatedAt)}
	rg := pb.RoleRecord{Id: roleProject.ID, Name: roleProject.Name, ProjectId: &pID,
		IsAdmin: roleProject.IsAdmin, IsProtected: roleProject.IsProtected,
		CreatedAt: timestamppb.New(roleProject.CreatedAt), UpdatedAt: timestamppb.New(roleProject.UpdatedAt)}

	// Create GitHub provider
	_, err = qtx.CreateProvider(ctx, db.CreateProviderParams{
		Name:       github.Github,
		ProjectID:  project.ID,
		Implements: github.Implements,
		Definition: json.RawMessage(`{"github": {}}`),
	})
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to create provider: %v", err)
	}
	return &prj, []*pb.RoleRecord{&rl, &rg}, nil
}

// GetOrganizations is a service for getting a list of organizations
func (s *Server) GetOrganizations(ctx context.Context,
	in *pb.GetOrganizationsRequest) (*pb.GetOrganizationsResponse, error) {

	// define default values for limit and offset
	if in.Limit == nil || *in.Limit == -1 {
		in.Limit = new(int32)
		*in.Limit = PaginationLimit
	}
	if in.Offset == nil {
		in.Offset = new(int32)
		*in.Offset = 0
	}

	orgs, err := s.store.ListOrganizations(ctx, db.ListOrganizationsParams{
		Limit:  *in.Limit,
		Offset: *in.Offset,
	})
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get projects: %s", err)
	}

	var resp pb.GetOrganizationsResponse
	resp.Organizations = make([]*pb.OrganizationRecord, 0, len(orgs))
	for _, org := range orgs {
		var orgmeta OrgMeta
		err := json.Unmarshal(org.Metadata, &orgmeta)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal metadata: %s", err)
		}

		resp.Organizations = append(resp.Organizations, &pb.OrganizationRecord{
			Id:        org.ID.String(),
			Name:      org.Name,
			Company:   orgmeta.Company,
			CreatedAt: timestamppb.New(org.CreatedAt),
			UpdatedAt: timestamppb.New(org.UpdatedAt),
		})
	}

	return &resp, nil
}

func getOrganizationDependencies(ctx context.Context, store db.Store,
	org db.Project) ([]*pb.ProjectRecord, []*pb.RoleRecord, []*pb.UserRecord, error) {
	const MAX_ITEMS = 999
	// get the projects for the organization
	projects, err := store.GetChildrenProjects(ctx, org.ID)
	if err != nil {
		return nil, nil, nil, err
	}

	// if there are more than one project, we need to remove the calling project
	// from the list
	if len(projects) > 1 {
		// NOTE(jaosorior): We need to remove the calling project from the list
		// since it's included in the list of projects.
		projects = projects[db.CalculateProjectHierarchyOffset(0):]
	}

	// get the roles for the organization
	roles, err := store.ListRoles(ctx, db.ListRolesParams{OrganizationID: org.ID, Limit: MAX_ITEMS, Offset: 0})
	if err != nil {
		return nil, nil, nil, err
	}

	users, err := store.ListUsersByOrganization(ctx,
		db.ListUsersByOrganizationParams{OrganizationID: org.ID, Limit: MAX_ITEMS, Offset: 0})
	if err != nil {
		return nil, nil, nil, err
	}

	// convert to right data type
	var projectsPB []*pb.ProjectRecord
	for _, project := range projects {
		projectsPB = append(projectsPB, &pb.ProjectRecord{
			ProjectId:      project.ID.String(),
			OrganizationId: project.ParentID.UUID.String(),
			Name:           project.Name,
			CreatedAt:      timestamppb.New(project.CreatedAt),
			UpdatedAt:      timestamppb.New(project.UpdatedAt),
		})
	}

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
			Id:             user.ID,
			OrganizationId: user.OrganizationID.String(),
			Email:          &user.Email.String,
			FirstName:      &user.FirstName.String,
			LastName:       &user.LastName.String,
			CreatedAt:      timestamppb.New(user.CreatedAt),
			UpdatedAt:      timestamppb.New(user.UpdatedAt),
		})
	}

	return projectsPB, rolesPB, usersPB, nil
}

// GetOrganization is a service for getting an organization
func (s *Server) GetOrganization(ctx context.Context,
	in *pb.GetOrganizationRequest) (*pb.GetOrganizationResponse, error) {
	orgID, err := uuid.Parse(in.OrganizationId)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid organization ID")
	}

	// check if user is authorized
	if err := AuthorizedOnOrg(ctx, orgID); err != nil {
		return nil, err
	}

	org, err := s.store.GetOrganization(ctx, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, util.UserVisibleError(codes.NotFound, "organization not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get organization: %s", err)
	}

	projects, roles, users, err := getOrganizationDependencies(ctx, s.store, org)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get organization dependencies: %s", err)
	}

	var resp pb.GetOrganizationResponse
	var orgmeta OrgMeta
	err = json.Unmarshal(org.Metadata, &orgmeta)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal metadata: %s", err)
	}

	resp.Organization = &pb.OrganizationRecord{
		Id:        org.ID.String(),
		Name:      org.Name,
		Company:   orgmeta.Company,
		CreatedAt: timestamppb.New(org.CreatedAt),
		UpdatedAt: timestamppb.New(org.UpdatedAt),
	}
	resp.Projects = projects
	resp.Roles = roles
	resp.Users = users

	return &resp, nil
}

// GetOrganizationByName is a service for getting an organization
func (s *Server) GetOrganizationByName(ctx context.Context,
	in *pb.GetOrganizationByNameRequest) (*pb.GetOrganizationByNameResponse, error) {
	if in.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "organization name is required")
	}

	org, err := s.store.GetOrganizationByName(ctx, in.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "organization not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get organization: %s", err)
	}

	// check if user is authorized
	if err := AuthorizedOnOrg(ctx, org.ID); err != nil {
		return nil, err
	}

	projects, roles, users, err := getOrganizationDependencies(ctx, s.store, org)
	if err != nil {
		return nil, status.Errorf(codes.Unknown, "failed to get organization dependencies: %s", err)
	}

	var resp pb.GetOrganizationByNameResponse
	var orgmeta OrgMeta
	err = json.Unmarshal(org.Metadata, &orgmeta)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal metadata: %s", err)
	}

	resp.Organization = &pb.OrganizationRecord{
		Id:        org.ID.String(),
		Name:      org.Name,
		Company:   orgmeta.Company,
		CreatedAt: timestamppb.New(org.CreatedAt),
		UpdatedAt: timestamppb.New(org.UpdatedAt),
	}
	resp.Projects = projects
	resp.Roles = roles
	resp.Users = users

	return &resp, nil
}

type deleteOrganizationValidation struct {
	Id string `db:"id" validate:"required"`
}

// DeleteOrganization is a handler that deletes a organization
func (s *Server) DeleteOrganization(ctx context.Context,
	in *pb.DeleteOrganizationRequest) (*pb.DeleteOrganizationResponse, error) {
	validator := validator.New()
	err := validator.Struct(deleteOrganizationValidation{Id: in.Id})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %s", err)
	}

	orgID, err := uuid.Parse(in.Id)
	if err != nil {
		return nil, util.UserVisibleError(codes.InvalidArgument, "invalid organization ID")
	}

	_, err = s.store.GetOrganization(ctx, orgID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "organization not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get organization: %s", err)
	}

	if in.Force == nil {
		isProtected := false
		in.Force = &isProtected
	}

	// if we do not force the deletion, we need to check if there are projects
	if !*in.Force {
		// list projects belonging to that organization
		projects, err := s.store.GetChildrenProjects(ctx, orgID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to list projects: %s", err)
		}

		if len(projects) > db.CalculateProjectHierarchyOffset(0) {
			return nil, status.Errorf(codes.InvalidArgument, "cannot delete the organization, there are projects associated with it")
		}
	}

	// otherwise we delete, and delete projects in cascade
	err = s.store.DeleteOrganization(ctx, orgID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete organization: %s", err)
	}

	return &pb.DeleteOrganizationResponse{}, nil
}
