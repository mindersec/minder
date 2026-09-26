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
func TestRoleGrantCommand(t *testing.T) {
	const (
		projectSub    = "00000000-0000-0000-0000-000000000001"
		mockProjectID = "12345678-1234-1234-1234-123456789012"
	)

	tests := []cli.CmdTestCase{
		{
			Name: "grant role table output",
			Args: []string{"project", "role", "grant", "-s", projectSub, "-r", "admin", "-j", mockProjectID, "-o", "table"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)

				mockResp := &minderv1.AssignRoleResponse{}
				cli.LoadFixture(t, "mock_role_grant.json", mockResp)

				client.EXPECT().
					AssignRole(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			GoldenFileName: "grant_table.txt",
		},
		{
			Name: "grant role json output",
			Args: []string{"project", "role", "grant", "-s", projectSub, "-r", "admin", "-j", mockProjectID, "-o", "json"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)

				mockResp := &minderv1.AssignRoleResponse{}
				cli.LoadFixture(t, "mock_role_grant.json", mockResp)

				client.EXPECT().
					AssignRole(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			GoldenFileName: "grant_json.txt",
		},
		{
			Name: "grant role yaml output",
			Args: []string{"project", "role", "grant", "-s", projectSub, "-r", "admin", "-j", mockProjectID, "-o", "yaml"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)

				mockResp := &minderv1.AssignRoleResponse{}
				cli.LoadFixture(t, "mock_role_grant.json", mockResp)

				client.EXPECT().
					AssignRole(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			GoldenFileName: "grant_yaml.txt",
		},
		{
			Name: "grant role via email invite table output",
			Args: []string{"project", "role", "grant", "-e", "test@example.com", "-r", "viewer", "-j", mockProjectID, "-o", "table"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)

				mockResp := &minderv1.AssignRoleResponse{}
				cli.LoadFixture(t, "mock_role_grant_invite.json", mockResp)

				client.EXPECT().
					AssignRole(gomock.Any(), gomock.Any()).
					Return(mockResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			GoldenFileName: "grant_invite_table.txt",
		},
		{
			Name:          "fails when missing required flags (sub or email)",
			Args:          []string{"project", "role", "grant", "-r", "admin"},
			ExpectedError: "at least one of the flags in the group [sub email] is required",
		},
		{
			Name: "server error handling",
			Args: []string{"project", "role", "grant", "-s", projectSub, "-r", "admin", "-j", mockProjectID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)

				client.EXPECT().
					AssignRole(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "internal server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			ExpectedError: "internal server error",
		},
	}

	cli.RunCmdTests(t, tests, RoleCmd)
}
