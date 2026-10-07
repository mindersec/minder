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
func TestDeleteCommand(t *testing.T) {
	const testID = "00000000-0000-0000-0000-000000000001"
	const testName = "test-profile"
	testIDPtr := func() *string { s := testID; return &s }()

	tests := []cli.CmdTestCase{
		{
			Name: "delete success by id",
			Args: []string{"profile", "delete", "-i", testID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				client.EXPECT().
					DeleteProfile(gomock.Any(), gomock.Any()).
					Return(&minderv1.DeleteProfileResponse{}, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "delete_success.txt",
		},
		{
			Name: "delete failure not found",
			Args: []string{"profile", "delete", "-i", testID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				client.EXPECT().
					DeleteProfile(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "profile not found"))

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			ExpectedError: "profile not found",
		},
		{
			Name: "delete success by name",
			Args: []string{"profile", "delete", "-n", testName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				client.EXPECT().
					GetProfileByName(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetProfileByNameResponse{
						Profile: &minderv1.Profile{
							Id: testIDPtr,
						},
					}, nil)

				client.EXPECT().
					DeleteProfile(gomock.Any(), gomock.Any()).
					Return(&minderv1.DeleteProfileResponse{}, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "delete_success.txt",
		},
		{
			Name: "delete by name returns nil profile",
			Args: []string{"profile", "delete", "-n", testName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				client.EXPECT().
					GetProfileByName(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetProfileByNameResponse{Profile: nil}, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			ExpectedError: "profile not found",
		},
		{
			Name: "delete by name not found",
			Args: []string{"profile", "delete", "-n", testName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				client.EXPECT().
					GetProfileByName(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "profile not found"))

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			ExpectedError: "profile not found",
		},
		{
			Name:          "delete fails with both id and name",
			Args:          []string{"profile", "delete", "-i", testID, "-n", testName},
			MockSetup:     func(_ *testing.T, _ *gomock.Controller) context.Context { return context.Background() },
			ExpectedError: "if any flags in the group [id name] are set none of the others can be",
		},
		{
			Name:          "delete fails with neither id nor name",
			Args:          []string{"profile", "delete"},
			MockSetup:     func(_ *testing.T, _ *gomock.Controller) context.Context { return context.Background() },
			ExpectedError: "at least one of the flags in the group [id name] is required",
		},
	}

	cli.RunCmdTests(t, tests, ProfileCmd)
}
