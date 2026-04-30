// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestRegisterCommand(t *testing.T) {
	const (
		repoName = "mock-owner/mock-repo"
	)

	tests := []cli.CmdTestCase{
		{
			Name: "register single repo by name",
			Args: []string{"repo", "register", "-n", repoName, "-p", "github"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)

				mockRemoteResp := &minderv1.ListRemoteRepositoriesFromProviderResponse{}
				cli.LoadFixture(t, "mock_repo_register_remote.json", mockRemoteResp)

				client.EXPECT().
					ListRemoteRepositoriesFromProvider(gomock.Any(), gomock.Any()).
					Return(mockRemoteResp, nil).
					Times(1)

				mockRegisterResp := &minderv1.RegisterRepositoryResponse{}
				cli.LoadFixture(t, "mock_repo_register_success.json", mockRegisterResp)

				client.EXPECT().
					RegisterRepository(gomock.Any(), gomock.Any()).
					Return(mockRegisterResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			GoldenFileName: "register_single.table",
		},
		{
			Name:          "fails when using mutually exclusive flags",
			Args:          []string{"repo", "register", "-n", repoName, "--all"},
			ExpectedError: "cannot use --name and --all together",
		},
		{
			Name:          "fails on invalid repository name",
			Args:          []string{"repo", "register", "-n", "mock-repo"},
			ExpectedError: "invalid repository name",
		},
	}

	cli.RunCmdTests(t, tests, RepoCmd)
}
