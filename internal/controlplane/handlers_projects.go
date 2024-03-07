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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/auth"
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

	projects, err := s.authzClient.ProjectsForUser(ctx, userInfo.IdentitySubject)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting projects for user: %v", err)
	}

	resp := minderv1.ListProjectsResponse{}

	for _, projectID := range projects {
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
