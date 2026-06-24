// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package repo

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
func TestReconcileCommand(t *testing.T) {
	const (
		repoID   = "12345678-1234-1234-1234-123456789012"
		repoName = "123456789"
	)

	tests := []cli.CmdTestCase{
		{
			Name: "reconcile by name",
			Args: []string{"repo", "reconcile", "-n", repoName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProjectsServiceClient(ctrl)

				mockResp := &minderv1.CreateEntityReconciliationTaskResponse{}
				cli.LoadFixture(t, "mock_repo_reconcile.json", mockResp)

				client.EXPECT().
					CreateEntityReconciliationTask(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), client)
			},
			GoldenFileName: "reconcile_success.txt",
		},
		{
			Name: "reconcile by id",
			Args: []string{"repo", "reconcile", "-i", repoID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProjectsServiceClient(ctrl)

				mockResp := &minderv1.CreateEntityReconciliationTaskResponse{}
				cli.LoadFixture(t, "mock_repo_reconcile.json", mockResp)

				client.EXPECT().
					CreateEntityReconciliationTask(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), client)
			},
			GoldenFileName: "reconcile_id_success.txt",
		},
		{
			Name:          "fails when missing both name and id",
			Args:          []string{"repo", "reconcile"},
			ExpectedError: "at least one of the flags in the group [name id] is required",
		},
		{
			Name:          "fails when passing both name and id",
			Args:          []string{"repo", "reconcile", "-n", repoName, "-i", repoID},
			ExpectedError: "if any flags in the group [name id] are set none of the others can be",
		},
		{
			Name: "server error handling",
			Args: []string{"repo", "reconcile", "-n", "missing-repo"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProjectsServiceClient(ctrl)

				client.EXPECT().
					CreateEntityReconciliationTask(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "entity not found")).
					Times(1)

				return cli.WithRPCClient[minderv1.ProjectsServiceClient](context.Background(), client)
			},
			ExpectedError: "entity not found",
		},
	}

	cli.RunCmdTests(t, tests, RepoCmd)
}
