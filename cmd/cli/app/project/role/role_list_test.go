// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package role

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
func TestRoleListCommand(t *testing.T) {
	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		permissionsClient := mockv1.NewMockPermissionsServiceClient(ctrl)

		mockResp := &minderv1.ListRolesResponse{}
		cli.LoadFixture(t, "mock_role_list.json", mockResp)

		permissionsClient.EXPECT().
			ListRoles(gomock.Any(), gomock.Any()).
			Return(mockResp, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), permissionsClient)
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "list roles - table output",
			Args:           []string{"project", "role", "list", "-o", "table"},
			MockSetup:      setupSuccess,
			GoldenFileName: "role_list.table",
		},
		{
			Name:           "list roles - default output (table)",
			Args:           []string{"project", "role", "list"},
			MockSetup:      setupSuccess,
			GoldenFileName: "role_list.table",
		},
		{
			Name:           "list roles - json output",
			Args:           []string{"project", "role", "list", "-o", "json"},
			MockSetup:      setupSuccess,
			GoldenFileName: "role_list.json",
		},
		{
			Name:           "list roles - yaml output",
			Args:           []string{"project", "role", "list", "-o", "yaml"},
			MockSetup:      setupSuccess,
			GoldenFileName: "role_list.yaml",
		},
		{
			Name: "list roles - empty table output",
			Args: []string{"project", "role", "list", "-o", "table"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				permissionsClient := mockv1.NewMockPermissionsServiceClient(ctrl)

				emptyResp := &minderv1.ListRolesResponse{}
				cli.LoadFixture(t, "mock_role_list_empty.json", emptyResp)

				permissionsClient.EXPECT().
					ListRoles(gomock.Any(), gomock.Any()).
					Return(emptyResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), permissionsClient)
			},
			GoldenFileName: "role_list_empty.table",
		},
		{
			Name:          "fails on invalid output format",
			Args:          []string{"project", "role", "list", "-o", "invalid_format"},
			ExpectedError: "invalid argument",
		},
		{
			Name: "server error handling",
			Args: []string{"project", "role", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				permissionsClient := mockv1.NewMockPermissionsServiceClient(ctrl)

				permissionsClient.EXPECT().
					ListRoles(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), permissionsClient)
			},
			ExpectedError: "server error",
		},
	}

	cli.RunCmdTests(t, tests, RoleCmd)
}
