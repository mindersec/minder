// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
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
	tests := []cli.CmdTestCase{
		{
			Name: "list projects table output",
			Args: []string{"project", "list", "-o", "table"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProjectsServiceClient(ctrl)

				mockResp := &minderv1.ListProjectsResponse{}
				cli.LoadFixture(t, "mock_project_list.json", mockResp)

				client.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_table.txt",
		},
		{
			Name: "list projects json output",
			Args: []string{"project", "list", "-o", "json"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProjectsServiceClient(ctrl)

				mockResp := &minderv1.ListProjectsResponse{}
				cli.LoadFixture(t, "mock_project_list.json", mockResp)

				client.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_json.txt",
		},
		{
			Name: "list projects yaml output",
			Args: []string{"project", "list", "-o", "yaml"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProjectsServiceClient(ctrl)

				mockResp := &minderv1.ListProjectsResponse{}
				cli.LoadFixture(t, "mock_project_list.json", mockResp)

				client.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_yaml.txt",
		},
		{
			Name: "server error handling",
			Args: []string{"project", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProjectsServiceClient(ctrl)

				client.EXPECT().
					ListProjects(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "internal server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), client)
			},
			ExpectedError: "internal server error",
		},
	}

	cli.RunCmdTests(t, tests, ProjectCmd)
}
