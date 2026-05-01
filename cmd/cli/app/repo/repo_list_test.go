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
func TestListCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name: "list repositories - table output",
			Args: []string{"repo", "list", "-o", "table"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)

				mockResp := &minderv1.ListRepositoriesResponse{}
				cli.LoadFixture(t, "mock_repo_list.json", mockResp)

				client.EXPECT().
					ListRepositories(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_table.txt",
		},
		{
			Name: "list repositories - empty result",
			Args: []string{"repo", "list", "-o", "table"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)

				mockResp := &minderv1.ListRepositoriesResponse{}
				cli.LoadFixture(t, "mock_repo_list_empty.json", mockResp)

				client.EXPECT().
					ListRepositories(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_empty.txt",
		},
		{
			Name:          "fails on invalid output format",
			Args:          []string{"repo", "list", "-o", "invalid_format"},
			ExpectedError: "invalid argument",
		},
		{
			Name: "server error handling",
			Args: []string{"repo", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)
				client.EXPECT().
					ListRepositories(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "internal server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			ExpectedError: "internal server error",
		},
	}

	cli.RunCmdTests(t, tests, RepoCmd)
}
