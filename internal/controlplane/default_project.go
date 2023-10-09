//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/mediator/internal/db"
	github "github.com/stacklok/mediator/internal/providers/github"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// OrgMeta is the metadata associated with an organization
type OrgMeta struct {
	Company string `json:"company"`
}

// ProjectMeta is the metadata associated with a project
type ProjectMeta struct {
	Description string `json:"description"`
	IsProtected bool   `json:"is_protected"`
}

// CreateDefaultRecordsForOrg creates the default records, such as projects, roles and provider for the organization
func CreateDefaultRecordsForOrg(ctx context.Context, qtx db.Querier,
	org db.Project, projectName string) (*pb.Project, []int32, error) {
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
		Name:     projectName,
		Metadata: jsonmeta,
	})
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to create default project: %v", err)
	}

	prj := pb.Project{
		ProjectId:   project.ID.String(),
		Name:        project.Name,
		Description: projectmeta.Description,
		IsProtected: projectmeta.IsProtected,
		CreatedAt:   timestamppb.New(project.CreatedAt),
		UpdatedAt:   timestamppb.New(project.UpdatedAt),
	}

	// we can create the default role for org and for project
	// This creates the role for the organization admin
	role1, err := qtx.CreateRole(ctx, db.CreateRoleParams{
		OrganizationID: org.ID,
		Name:           fmt.Sprintf("%s-org-admin", org.Name),
		IsAdmin:        true,
		IsProtected:    true,
	})
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to create default org role: %v", err)
	}

	// this creates te role for the project admin
	role2, err := qtx.CreateRole(ctx, db.CreateRoleParams{
		OrganizationID: org.ID,
		ProjectID:      uuid.NullUUID{UUID: project.ID, Valid: true},
		Name:           fmt.Sprintf("%s-project-admin", org.Name),
		IsAdmin:        true,
		IsProtected:    true,
	})

	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to create default project role: %v", err)
	}

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
	return &prj, []int32{role1.ID, role2.ID}, nil
}
