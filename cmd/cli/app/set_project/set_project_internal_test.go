// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package set_project

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
		ListProjects(gomock.Any(), gomock.Any()).
		Return(&minderv1.ListProjectsResponse{
			Projects: []*minderv1.Project{
				{ProjectId: projectID.String(), Name: "my-project", DisplayName: "My Project"},
			},
		}, nil)

	got, err := resolveProjectByNameFromClient(context.Background(), mockClient, "my-project")
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
		ListProjects(gomock.Any(), gomock.Any()).
		Return(&minderv1.ListProjectsResponse{
			Projects: []*minderv1.Project{
				{ProjectId: projectID.String(), Name: "my-project", DisplayName: "My Project"},
			},
		}, nil)

	got, err := resolveProjectByNameFromClient(context.Background(), mockClient, "My Project")
	require.NoError(t, err)
	require.Equal(t, projectID, got)
}

func TestResolveProjectByName_NotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockv1.NewMockProjectsServiceClient(ctrl)
	mockClient.EXPECT().
		ListProjects(gomock.Any(), gomock.Any()).
		Return(&minderv1.ListProjectsResponse{
			Projects: []*minderv1.Project{
				{ProjectId: uuid.New().String(), Name: "other-project", DisplayName: "Other"},
			},
		}, nil)

	_, err := resolveProjectByNameFromClient(context.Background(), mockClient, "does-not-exist")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no project found")
}

func TestResolveProjectByName_EmptyList(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockv1.NewMockProjectsServiceClient(ctrl)
	mockClient.EXPECT().
		ListProjects(gomock.Any(), gomock.Any()).
		Return(&minderv1.ListProjectsResponse{Projects: []*minderv1.Project{}}, nil)

	_, err := resolveProjectByNameFromClient(context.Background(), mockClient, "any-name")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no project found")
}

func TestResolveProjectByName_ServerError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mockv1.NewMockProjectsServiceClient(ctrl)
	mockClient.EXPECT().
		ListProjects(gomock.Any(), gomock.Any()).
		Return(nil, status.Error(codes.Internal, "server error"))

	_, err := resolveProjectByNameFromClient(context.Background(), mockClient, "my-project")
	require.Error(t, err)
	require.Contains(t, err.Error(), "server error")
}
