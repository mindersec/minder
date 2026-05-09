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
func TestGetCommand(t *testing.T) {
	const (
		repoID   = "12345678-1234-1234-1234-123456789012"
		repoName = "mock-owner/mock-repo"
	)

	tests := []cli.CmdTestCase{
		{
			Name: "get repository by name - json output",
			Args: []string{"repo", "get", "-n", repoName, "-o", "json"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)

				mockResp := &minderv1.GetRepositoryByNameResponse{}
				cli.LoadFixture(t, "mock_repo_get.json", mockResp)

				client.EXPECT().
					GetRepositoryByName(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			GoldenFileName: "get_name_json.txt",
		},
		{
			Name: "get repository by id - yaml output",
			Args: []string{"repo", "get", "-i", repoID, "-o", "yaml"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)

				mockResp := &minderv1.GetRepositoryByIdResponse{}
				cli.LoadFixture(t, "mock_repo_get.json", mockResp)

				client.EXPECT().
					GetRepositoryById(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			GoldenFileName: "get_id_yaml.txt",
		},
		{
			Name:          "fails on table output format",
			Args:          []string{"repo", "get", "-n", "mock-owner/mock-repo", "-o", "table"},
			ExpectedError: "invalid argument",
		},
		{
			Name:          "fails when missing both name and id",
			Args:          []string{"repo", "get"},
			ExpectedError: "at least one of the flags in the group [name id] is required",
		},
		{
			Name: "server error handling",
			Args: []string{"repo", "get", "-n", repoName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)
				client.EXPECT().
					GetRepositoryByName(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "repository not found")).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			ExpectedError: "repository not found",
		},
	}

	cli.RunCmdTests(t, tests, RepoCmd)
}
