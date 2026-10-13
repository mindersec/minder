// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package project

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestProjectListCommand(t *testing.T) {
	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		projectsClient := mockv1.NewMockProjectsServiceClient(ctrl)

		mockResp := &minderv1.ListProjectsResponse{}
		cli.LoadFixture(t, "mock_project_list.json", mockResp)

		projectsClient.EXPECT().
			ListProjects(gomock.Any(), gomock.Any()).
			Return(mockResp, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), projectsClient)
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "list projects - table output",
			Args:           []string{"project", "list", "-o", "table"},
			MockSetup:      setupSuccess,
			GoldenFileName: "project_list.table",
		},
		{
			Name:           "list projects - default output (table)",
			Args:           []string{"project", "list"},
			MockSetup:      setupSuccess,
			GoldenFileName: "project_list.table",
		},
		{
			Name:           "list projects - json output",
			Args:           []string{"project", "list", "-o", "json"},
			MockSetup:      setupSuccess,
			GoldenFileName: "project_list.json",
		},
		{
			Name:           "list projects - yaml output",
			Args:           []string{"project", "list", "-o", "yaml"},
			MockSetup:      setupSuccess,
			GoldenFileName: "project_list.yaml",
		},
		{
			Name: "list projects - empty table output",
			Args: []string{"project", "list", "-o", "table"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				projectsClient := mockv1.NewMockProjectsServiceClient(ctrl)

				emptyResp := &minderv1.ListProjectsResponse{}
				cli.LoadFixture(t, "mock_project_list_empty.json", emptyResp)

				projectsClient.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					Return(emptyResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), projectsClient)
			},
			GoldenFileName: "project_list_empty.table",
		},
		{
			Name:          "fails on invalid output format",
			Args:          []string{"project", "list", "-o", "invalid_format"},
			ExpectedError: "invalid argument",
		},
		{
			Name: "server error handling",
			Args: []string{"project", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				projectsClient := mockv1.NewMockProjectsServiceClient(ctrl)

				projectsClient.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), projectsClient)
			},
			ExpectedError: "server error",
		},
	}

	cli.RunCmdTests(t, tests, ProjectCmd)
}
