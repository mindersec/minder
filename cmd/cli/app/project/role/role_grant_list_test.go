// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
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
func TestRoleGrantListCommand(t *testing.T) {
	const mockProjectID = "12345678-1234-1234-1234-123456789012"

	tests := []cli.CmdTestCase{
		{
			Name: "grant list table output",
			Args: []string{"project", "role", "grant", "list", "-j", mockProjectID, "-o", "table"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)

				mockResp := &minderv1.ListRoleAssignmentsResponse{}
				cli.LoadFixture(t, "mock_role_grant_list.json", mockResp)

				client.EXPECT().
					ListRoleAssignments(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			GoldenFileName: "grant_list_table.txt",
		},
		{
			Name: "grant list json output",
			Args: []string{"project", "role", "grant", "list", "-j", mockProjectID, "-o", "json"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)

				mockResp := &minderv1.ListRoleAssignmentsResponse{}
				cli.LoadFixture(t, "mock_role_grant_list.json", mockResp)

				client.EXPECT().
					ListRoleAssignments(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			GoldenFileName: "grant_list_json.txt",
		},
		{
			Name: "grant list yaml output",
			Args: []string{"project", "role", "grant", "list", "-o", "yaml"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)

				mockResp := &minderv1.ListRoleAssignmentsResponse{}
				cli.LoadFixture(t, "mock_role_grant_list.json", mockResp)

				client.EXPECT().
					ListRoleAssignments(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			GoldenFileName: "grant_list_yaml.txt",
		},
		{
			Name: "server error handling",
			Args: []string{"project", "role", "grant", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)

				client.EXPECT().
					ListRoleAssignments(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "internal server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			ExpectedError: "internal server error",
		},
	}

	cli.RunCmdTests(t, tests, RoleCmd)
}
