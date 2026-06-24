// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

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
			Name: "list profiles table success",
			Args: []string{"profile", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockResp := &minderv1.ListProfilesResponse{}
				cli.LoadFixture(t, "mock_profile_list.json", mockResp)

				client.EXPECT().
					ListProfiles(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_profiles_table.txt",
		},
		{
			Name: "list profiles json success",
			Args: []string{"profile", "list", "-o", "json"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockResp := &minderv1.ListProfilesResponse{}
				cli.LoadFixture(t, "mock_profile_list.json", mockResp)

				client.EXPECT().
					ListProfiles(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_profiles.json",
		},
		{
			Name: "list profiles yaml success",
			Args: []string{"profile", "list", "-o", "yaml"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockResp := &minderv1.ListProfilesResponse{}
				cli.LoadFixture(t, "mock_profile_list.json", mockResp)

				client.EXPECT().
					ListProfiles(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_profiles.yaml",
		},
		{
			Name: "list profiles with label filter",
			Args: []string{"profile", "list", "-l", "test-label"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockResp := &minderv1.ListProfilesResponse{Profiles: []*minderv1.Profile{}}

				client.EXPECT().
					ListProfiles(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_profiles_filtered.txt",
		},
		{
			Name:          "failure invalid format",
			Args:          []string{"profile", "list", "-o", "invalid"},
			ExpectedError: "invalid argument",
		},
		{
			Name: "failure server error",
			Args: []string{"profile", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				client.EXPECT().
					ListProfiles(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "database connection failed"))

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			ExpectedError: "database connection failed",
		},
	}

	cli.RunCmdTests(t, tests, ProfileCmd)
}
