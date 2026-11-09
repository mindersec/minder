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
func TestRoleGrantListCommand(t *testing.T) {
	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		permissionsClient := mockv1.NewMockPermissionsServiceClient(ctrl)

		mockResp := &minderv1.ListRoleAssignmentsResponse{}
		cli.LoadFixture(t, "mock_role_grant_list.json", mockResp)

		permissionsClient.EXPECT().
			ListRoleAssignments(gomock.Any(), gomock.Any()).
			Return(mockResp, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), permissionsClient)
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "list role grants - table output",
			Args:           []string{"project", "role", "grant", "list", "-o", "table"},
			MockSetup:      setupSuccess,
			GoldenFileName: "role_grant_list.table",
		},
		{
			Name:           "list role grants - default output (table)",
			Args:           []string{"project", "role", "grant", "list"},
			MockSetup:      setupSuccess,
			GoldenFileName: "role_grant_list.table",
		},
		{
			Name:           "list role grants - json output",
			Args:           []string{"project", "role", "grant", "list", "-o", "json"},
			MockSetup:      setupSuccess,
			GoldenFileName: "role_grant_list.json",
		},
		{
			Name:           "list role grants - yaml output",
			Args:           []string{"project", "role", "grant", "list", "-o", "yaml"},
			MockSetup:      setupSuccess,
			GoldenFileName: "role_grant_list.yaml",
		},
		{
			Name: "list role grants - no pending invitations table output",
			Args: []string{"project", "role", "grant", "list", "-o", "table"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				permissionsClient := mockv1.NewMockPermissionsServiceClient(ctrl)

				noInvitesResp := &minderv1.ListRoleAssignmentsResponse{}
				cli.LoadFixture(t, "mock_role_grant_list_no_invites.json", noInvitesResp)

				permissionsClient.EXPECT().
					ListRoleAssignments(gomock.Any(), gomock.Any()).
					Return(noInvitesResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), permissionsClient)
			},
			GoldenFileName: "role_grant_list_no_invites.table",
		},
		{
			Name:          "fails on invalid output format",
			Args:          []string{"project", "role", "grant", "list", "-o", "invalid_format"},
			ExpectedError: "invalid argument",
		},
		{
			Name: "server error handling",
			Args: []string{"project", "role", "grant", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				permissionsClient := mockv1.NewMockPermissionsServiceClient(ctrl)

				permissionsClient.EXPECT().
					ListRoleAssignments(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), permissionsClient)
			},
			ExpectedError: "server error",
		},
	}

	cli.RunCmdTests(t, tests, RoleCmd)
}
