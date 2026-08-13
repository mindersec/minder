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
func TestProjectCreateCommand(t *testing.T) {
	const (
		projectName = "mock-project"
	)
	tests := []cli.CmdTestCase{
		{
			Name: "create project table output",
			Args: []string{"project", "create", "-n", projectName, "-o", "table"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProjectsServiceClient(ctrl)

				mockResp := &minderv1.CreateProjectResponse{}
				cli.LoadFixture(t, "mock_project_create.json", mockResp)

				client.EXPECT().
					CreateProject(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), client)
			},
			GoldenFileName: "create_table.txt",
		},
		{
			Name: "create project json output",
			Args: []string{"project", "create", "-n", projectName, "-o", "json"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProjectsServiceClient(ctrl)

				mockResp := &minderv1.CreateProjectResponse{}
				cli.LoadFixture(t, "mock_project_create.json", mockResp)

				client.EXPECT().
					CreateProject(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), client)
			},
			GoldenFileName: "create_json.txt",
		},
		{
			Name: "create project yaml output",
			Args: []string{"project", "create", "-n", projectName, "-o", "yaml"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProjectsServiceClient(ctrl)

				mockResp := &minderv1.CreateProjectResponse{}
				cli.LoadFixture(t, "mock_project_create.json", mockResp)

				client.EXPECT().
					CreateProject(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), client)
			},
			GoldenFileName: "create_yaml.txt",
		},
		{
			Name:          "fails when missing name flag",
			Args:          []string{"project", "create"},
			ExpectedError: "required flag(s) \"name\" not set",
		},
		{
			Name: "server error handling",
			Args: []string{"project", "create", "-n", projectName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProjectsServiceClient(ctrl)

				client.EXPECT().
					CreateProject(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "internal server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), client)
			},
			ExpectedError: "internal server error",
		},
	}

	cli.RunCmdTests(t, tests, ProjectCmd)
}
