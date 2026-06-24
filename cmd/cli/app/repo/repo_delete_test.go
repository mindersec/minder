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
func TestDeleteCommand(t *testing.T) {
	const (
		repoID   = "12345678-1234-1234-1234-123456789012"
		repoName = "mock-owner/mock-repo"
	)

	tests := []cli.CmdTestCase{
		{
			Name: "delete repository by name",
			Args: []string{"repo", "delete", "-n", repoName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)

				mockResp := &minderv1.DeleteRepositoryByNameResponse{}
				cli.LoadFixture(t, "mock_repo_delete_name.json", mockResp)

				client.EXPECT().
					DeleteRepositoryByName(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			GoldenFileName: "delete_name.txt",
		},
		{
			Name: "delete repository by id",
			Args: []string{"repo", "delete", "-i", repoID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)

				mockResp := &minderv1.DeleteRepositoryByIdResponse{}
				cli.LoadFixture(t, "mock_repo_delete_id.json", mockResp)

				client.EXPECT().
					DeleteRepositoryById(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			GoldenFileName: "delete_id.txt",
		},
		{
			Name:          "fails when missing both name and id",
			Args:          []string{"repo", "delete"},
			ExpectedError: "at least one of the flags in the group [name id] is required",
		},
		{
			Name: "server error handling",
			Args: []string{"repo", "delete", "-n", "missing-repo"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)

				client.EXPECT().
					DeleteRepositoryByName(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "repository not found")).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			ExpectedError: "repository not found",
		},
	}

	cli.RunCmdTests(t, tests, RepoCmd)
}
