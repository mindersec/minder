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
func TestProjectCreateCommand(t *testing.T) {
	const testProjectName = "test-subproject"

	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		projectsClient := mockv1.NewMockProjectsServiceClient(ctrl)

		mockResp := &minderv1.CreateProjectResponse{}
		cli.LoadFixture(t, "mock_project_create.json", mockResp)

		projectsClient.EXPECT().
			CreateProject(gomock.Any(), gomock.Cond(func(r *minderv1.CreateProjectRequest) bool {
				return r.GetName() == testProjectName
			})).
			Return(mockResp, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), projectsClient)
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "create sub-project - table output",
			Args:           []string{"project", "create", "-n", testProjectName, "-o", "table"},
			MockSetup:      setupSuccess,
			GoldenFileName: "project_create.table",
		},
		{
			Name:           "create sub-project - default output (table)",
			Args:           []string{"project", "create", "-n", testProjectName},
			MockSetup:      setupSuccess,
			GoldenFileName: "project_create.table",
		},
		{
			Name:           "create sub-project - json output",
			Args:           []string{"project", "create", "-n", testProjectName, "-o", "json"},
			MockSetup:      setupSuccess,
			GoldenFileName: "project_create.json",
		},
		{
			Name:           "create sub-project - yaml output",
			Args:           []string{"project", "create", "-n", testProjectName, "-o", "yaml"},
			MockSetup:      setupSuccess,
			GoldenFileName: "project_create.yaml",
		},
		{
			Name:          "fails on invalid output format",
			Args:          []string{"project", "create", "-n", testProjectName, "-o", "invalid_format"},
			ExpectedError: "invalid argument",
		},
		{
			Name:          "fails when name flag is missing",
			Args:          []string{"project", "create"},
			ExpectedError: "required flag(s) \"name\" not set",
		},
		{
			Name: "server error handling",
			Args: []string{"project", "create", "-n", testProjectName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				projectsClient := mockv1.NewMockProjectsServiceClient(ctrl)

				projectsClient.EXPECT().
					CreateProject(gomock.Any(), gomock.Cond(func(r *minderv1.CreateProjectRequest) bool {
						return r.GetName() == testProjectName
					})).
					Return(nil, status.Error(codes.Internal, "server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), projectsClient)
			},
			ExpectedError: "server error",
		},
	}

	cli.RunCmdTests(t, tests, ProjectCmd)
}
