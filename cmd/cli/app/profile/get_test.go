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
func TestGetCommand(t *testing.T) {
	testName := "test-profile"

	tests := []cli.CmdTestCase{
		{
			Name: "get by name table success",
			Args: []string{"profile", "get", "-n", testName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)
				mockProf := &minderv1.Profile{}
				cli.LoadFixture(t, "mock_profile_get.json", mockProf)

				client.EXPECT().
					GetProfileByName(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetProfileByNameResponse{Profile: mockProf}, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "get_by_name_table.txt",
		},
		{
			Name:          "failure missing id and name",
			Args:          []string{"profile", "get"},
			ExpectedError: "id or name required",
		},
		{
			Name:          "failure invalid format",
			Args:          []string{"profile", "get", "-n", testName, "-o", "invalid"},
			ExpectedError: "invalid argument",
		},
		{
			Name: "failure profile not found",
			Args: []string{"profile", "get", "-n", "non-existent"},
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
	}

	cli.RunCmdTests(t, tests, ProfileCmd)
}
