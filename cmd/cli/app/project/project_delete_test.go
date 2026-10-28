// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package project

import (
	"context"
	"fmt"
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
	const testProjectID = "00000000-0000-0000-0000-000000000000"

	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		projectsClient := mockv1.NewMockProjectsServiceClient(ctrl)

		projectsClient.EXPECT().
			DeleteProject(gomock.Any(), gomock.Cond(func(r *minderv1.DeleteProjectRequest) bool {
				return r.GetContext().GetProject() == testProjectID
			})).
			Return(&minderv1.DeleteProjectResponse{ProjectId: testProjectID}, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), projectsClient)
	}

	tests := []cli.CmdTestCase{
		{
			Name:              "delete sub-project - success",
			Args:              []string{"project", "delete", "-j", testProjectID},
			MockSetup:         setupSuccess,
			ExpectEmptyStdout: true,
			ExpectedStderr:    fmt.Sprintf("Successfully deleted project with id: %s\n", testProjectID),
		},
		{
			Name:          "fails when project flag is missing",
			Args:          []string{"project", "delete"},
			ExpectedError: "required flag(s) \"project\" not set",
		},
		{
			Name: "server error handling",
			Args: []string{"project", "delete", "-j", testProjectID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				projectsClient := mockv1.NewMockProjectsServiceClient(ctrl)

				projectsClient.EXPECT().
					DeleteProject(gomock.Any(), gomock.Cond(func(r *minderv1.DeleteProjectRequest) bool {
						return r.GetContext().GetProject() == testProjectID
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
