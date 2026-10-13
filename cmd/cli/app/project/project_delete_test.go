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
func TestProjectDeleteCommand(t *testing.T) {
	const (
		projectID = "12345678-1234-1234-1234-123456789012"
	)

	tests := []cli.CmdTestCase{
		{
			Name: "delete project success",
			Args: []string{"project", "delete", "-j", projectID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProjectsServiceClient(ctrl)

				mockResp := &minderv1.DeleteProjectResponse{}
				cli.LoadFixture(t, "mock_project_delete.json", mockResp)

				client.EXPECT().
					DeleteProject(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), client)
			},
			GoldenFileName: "delete_success.txt",
		},
		{
			Name:          "fails when missing project flag",
			Args:          []string{"project", "delete"},
			ExpectedError: "required flag(s) \"project\" not set",
		},
		{
			Name: "server error handling",
			Args: []string{"project", "delete", "-j", projectID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProjectsServiceClient(ctrl)

				client.EXPECT().
					DeleteProject(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "internal server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), client)
			},
			ExpectedError: "internal server error",
		},
	}

	cli.RunCmdTests(t, tests, ProjectCmd)
}
