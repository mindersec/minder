// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/cmd/cli/app/profile"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestStatusListCommand(t *testing.T) {
	testName := "test-profile"

	tests := []cli.CmdTestCase{
		{
			Name: "status list table success (standard)",
			Args: []string{"profile", "status", "list", "-n", testName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockResp := &minderv1.GetProfileStatusByNameResponse{}
				cli.LoadFixture(t, "mock_profile_status.json", mockResp)

				client.EXPECT().
					GetProfileStatusByName(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "status_list_table.txt",
		},
		{
			Name: "status list table success (detailed flag: -d)",
			Args: []string{"profile", "status", "list", "-n", testName, "-d"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockResp := &minderv1.GetProfileStatusByNameResponse{}
				cli.LoadFixture(t, "mock_profile_status.json", mockResp)

				client.EXPECT().
					GetProfileStatusByName(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "status_list_table_detailed.txt",
		},
		{
			Name: "status list table success (detailed no emoji)",
			Args: []string{"profile", "status", "list", "-n", testName, "-d", "--emoji=false"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockResp := &minderv1.GetProfileStatusByNameResponse{}
				cli.LoadFixture(t, "mock_profile_status.json", mockResp)

				client.EXPECT().
					GetProfileStatusByName(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "status_list_table_detailed_no_emoji.txt",
		},
		{
			Name: "status list table success (emoji disabled)",
			Args: []string{"profile", "status", "list", "-n", testName, "--emoji=false"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockResp := &minderv1.GetProfileStatusByNameResponse{}
				cli.LoadFixture(t, "mock_profile_status.json", mockResp)

				client.EXPECT().
					GetProfileStatusByName(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "status_list_table_no_emoji.txt",
		},
		{
			Name: "status list filter by ruleName",
			Args: []string{"profile", "status", "list", "-n", testName, "--ruleName", "secret_scanning"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockResp := &minderv1.GetProfileStatusByNameResponse{}
				cli.LoadFixture(t, "mock_profile_status.json", mockResp)

				client.EXPECT().
					GetProfileStatusByName(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "status_list_filter_rulename.txt",
		},
		{
			Name: "status list filter by ruleType",
			Args: []string{"profile", "status", "list", "-n", testName, "-r", "secret_scanning_configured"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockResp := &minderv1.GetProfileStatusByNameResponse{}
				cli.LoadFixture(t, "mock_profile_status.json", mockResp)

				client.EXPECT().
					GetProfileStatusByName(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "status_list_filter_ruletype.txt",
		},
		{
			Name: "status list yaml success",
			Args: []string{"profile", "status", "list", "-n", testName, "-o", "yaml"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockResp := &minderv1.GetProfileStatusByNameResponse{}
				cli.LoadFixture(t, "mock_profile_status.json", mockResp)

				client.EXPECT().
					GetProfileStatusByName(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "status_list.yaml",
		},
		{
			Name:          "failure missing required name flag",
			Args:          []string{"profile", "status", "list"},
			ExpectedError: `required flag(s) "name" not set`,
		},
		{
			Name:          "failure invalid format",
			Args:          []string{"profile", "status", "list", "-n", testName, "-o", "invalid"},
			ExpectedError: "invalid argument",
		},
		{
			Name: "failure server error",
			Args: []string{"profile", "status", "list", "-n", testName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				client.EXPECT().
					GetProfileStatusByName(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "profile status not found"))

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			ExpectedError: "profile status not found",
		},
	}

	cli.RunCmdTests(t, tests, profile.ProfileCmd)
}
