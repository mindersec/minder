// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package set_project_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

func TestResolveProjectByName_MatchesName(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	projectID := uuid.New()
	mockClient := mockv1.NewMockProjectsServiceClient(ctrl)

	mockClient.EXPECT().
		ListProjects(gomock.Any(), &minderv1.ListProjectsRequest{}, gomock.Any()).
		Return(&minderv1.ListProjectsResponse{
			Projects: []*minderv1.Project{
				{ProjectId: projectID.String(), Name: "my-project", DisplayName: "My Project"},
			},
		}, nil)

	got, err := resolveProjectByNameWithClient(context.Background(), mockClient, "my-project")
	require.NoError(t, err)
	require.Equal(t, projectID, got)
}

func TestResolveProjectByName_MatchesDisplayName(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	projectID := uuid.New()
	mockClient := mockv1.NewMockProjectsServiceClient(ctrl)

	mockClient.EXPECT().
		ListProjects(gomock.Any(), &minderv1.ListProjectsRequest{}, gomock.Any()).
		Return(&minderv1.ListProjectsResponse{
			Projects: []*minderv1.Project{
				{ProjectId: projectID.String(), Name: "my-project", DisplayName: "My Project"},
			},
		}, nil)

	got, err := resolveProjectByNameWithClient(context.Background(), mockClient, "My Project")
	require.NoError(t, err)
	require.Equal(t, projectID, got)
}

func TestResolveProjectByName_NotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockv1.NewMockProjectsServiceClient(ctrl)

	mockClient.EXPECT().
		ListProjects(gomock.Any(), &minderv1.ListProjectsRequest{}, gomock.Any()).
		Return(&minderv1.ListProjectsResponse{
			Projects: []*minderv1.Project{
				{ProjectId: uuid.New().String(), Name: "other-project", DisplayName: "Other"},
			},
		}, nil)

	_, err := resolveProjectByNameWithClient(context.Background(), mockClient, "does-not-exist")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no project found")
}

func TestResolveProjectByName_EmptyList(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockv1.NewMockProjectsServiceClient(ctrl)

	mockClient.EXPECT().
		ListProjects(gomock.Any(), &minderv1.ListProjectsRequest{}, gomock.Any()).
		Return(&minderv1.ListProjectsResponse{Projects: []*minderv1.Project{}}, nil)

	_, err := resolveProjectByNameWithClient(context.Background(), mockClient, "any-name")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no project found")
}

// resolveProjectByNameWithClient is a testable version that accepts an injected client
func resolveProjectByNameWithClient(ctx context.Context, client minderv1.ProjectsServiceClient, name string) (uuid.UUID, error) {
	resp, err := client.ListProjects(ctx, &minderv1.ListProjectsRequest{})
	if err != nil {
		return uuid.UUID{}, cli.MessageAndError("Error listing projects", err)
	}

	for _, p := range resp.Projects {
		if p.Name == name || p.DisplayName == name {
			id, err := uuid.Parse(p.ProjectId)
			if err != nil {
				return uuid.UUID{}, cli.MessageAndError("Error parsing project ID from server", err)
			}
			return id, nil
		}
	}

	return uuid.UUID{}, fmt.Errorf("no project found with name %q", name)
}
