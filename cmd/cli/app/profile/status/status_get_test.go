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
func TestStatusGetCommand(t *testing.T) {
	testId := "11111111-1111-1111-1111-111111111111"
	testName := "test-profile"
	testEntityName := "test-repo"
	testEntityUUID := "22222222-2222-2222-2222-222222222222"
	testEntityType := "repository"

	tests := []cli.CmdTestCase{
		{
			Name: "status get by id success",
			Args: []string{"profile", "status", "get", "-i", testId, "-e", testEntityName, "-t", testEntityType},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockResp := &minderv1.GetProfileStatusByIdResponse{}
				cli.LoadFixture(t, "mock_profile_status.json", mockResp)

				client.EXPECT().
					GetProfileStatusById(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "status_get_by_id_table.txt",
		},
		{
			Name: "status get by name success",
			Args: []string{"profile", "status", "get", "-n", testName, "-e", testEntityName, "-t", testEntityType},
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
			GoldenFileName: "status_get_by_name_table.txt",
		},
		{
			Name: "status get success with UUID entity format",
			Args: []string{"profile", "status", "get", "-n", testName, "-e", testEntityUUID, "-t", testEntityType},
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
			GoldenFileName: "status_get_uuid_entity.txt",
		},
		{
			Name: "status get json output",
			Args: []string{"profile", "status", "get", "-i", testId, "-e", testEntityName, "-t", testEntityType, "-o", "json"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockResp := &minderv1.GetProfileStatusByIdResponse{}
				cli.LoadFixture(t, "mock_profile_status.json", mockResp)

				client.EXPECT().
					GetProfileStatusById(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "status_get.json",
		},
		{
			Name: "status get yaml output",
			Args: []string{"profile", "status", "get", "-n", testName, "-e", testEntityName, "-t", testEntityType, "-o", "yaml"},
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
			GoldenFileName: "status_get.yaml",
		},
		{
			Name: "status get without emoji",
			Args: []string{"profile", "status", "get", "-i", testId, "-e", testEntityName, "-t", testEntityType, "--emoji=false"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockResp := &minderv1.GetProfileStatusByIdResponse{}
				cli.LoadFixture(t, "mock_profile_status.json", mockResp)

				client.EXPECT().
					GetProfileStatusById(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "status_get_no_emoji.txt",
		},
		{
			Name:          "failure missing required entity flag",
			Args:          []string{"profile", "status", "get", "-i", testId, "-t", testEntityType},
			ExpectedError: `required flag(s) "entity" not set`,
		},
		{
			Name:          "failure missing required entity-type flag",
			Args:          []string{"profile", "status", "get", "-i", testId, "-e", testEntityName},
			ExpectedError: `required flag(s) "entity-type" not set`,
		},
		{
			Name:          "failure missing both id and name",
			Args:          []string{"profile", "status", "get", "-e", testEntityName, "-t", testEntityType},
			ExpectedError: `at least one of the flags in the group [id name] is required`,
		},
		{
			Name: "failure server error",
			Args: []string{"profile", "status", "get", "-i", testId, "-e", testEntityName, "-t", testEntityType},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				client.EXPECT().
					GetProfileStatusById(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "database connection failed"))

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			ExpectedError: "database connection failed",
		},
	}

	cli.RunCmdTests(t, tests, profile.ProfileCmd)
}
