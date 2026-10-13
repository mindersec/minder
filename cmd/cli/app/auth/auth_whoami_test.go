// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestWhoamiCommand(t *testing.T) {
	setupMockUser := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()
		client := mockv1.NewMockUserServiceClient(ctrl)
		mockResponse := &minderv1.GetUserResponse{}
		cli.LoadFixture(t, "mock_user_response.json", mockResponse)

		client.EXPECT().
			GetUser(gomock.Any(), gomock.Any()).
			Return(mockResponse, nil)
		return cli.WithRPCClient[minderv1.UserServiceClient](context.Background(), client)
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "table output with data",
			Args:           []string{"auth", "whoami", "-o", app.Table},
			MockSetup:      setupMockUser,
			GoldenFileName: "whoami.table",
		},
		{
			Name:           "default output format (table)",
			Args:           []string{"auth", "whoami"},
			MockSetup:      setupMockUser,
			GoldenFileName: "whoami.table",
		},
		{
			Name:           "json output format",
			Args:           []string{"auth", "whoami", "-o", app.JSON},
			MockSetup:      setupMockUser,
			GoldenFileName: "whoami.json",
		},
		{
			Name:           "yaml output format",
			Args:           []string{"auth", "whoami", "-o", app.YAML},
			MockSetup:      setupMockUser,
			GoldenFileName: "whoami.yaml",
		},
		{
			Name: "grpc error handling",
			Args: []string{"auth", "whoami"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockUserServiceClient(ctrl)
				client.EXPECT().
					GetUser(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "internal server error"))
				return cli.WithRPCClient[minderv1.UserServiceClient](context.Background(), client)
			},
			ExpectedError: "internal server error",
		},
		{
			Name:          "invalid output format",
			Args:          []string{"auth", "whoami", "-o", "csv"},
			ExpectedError: "invalid argument",
		},
	}

	cli.RunCmdTests(t, tests, AuthCmd)
}
