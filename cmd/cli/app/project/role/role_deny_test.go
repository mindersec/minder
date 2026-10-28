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
func TestRoleDenyCommand(t *testing.T) {
	const (
		testProject = "00000000-0000-0000-0000-000000000000"
		testSubject = "sub-12345"
		testEmail   = "charlie@example.com"
		testRole    = "editor"
	)

	setupMockDenySubject := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()
		client := mockv1.NewMockPermissionsServiceClient(ctrl)
		mockResp := &minderv1.RemoveRoleResponse{}

		client.EXPECT().
			RemoveRole(gomock.Any(), gomock.Cond(func(req *minderv1.RemoveRoleRequest) bool {
				return req.GetContext().GetProject() == testProject &&
					req.GetRoleAssignment().GetRole() == testRole &&
					req.GetRoleAssignment().GetSubject() == testSubject
			})).
			Return(mockResp, nil)

		return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
	}

	setupMockDenyEmail := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()
		client := mockv1.NewMockPermissionsServiceClient(ctrl)
		mockResp := &minderv1.RemoveRoleResponse{}

		client.EXPECT().
			RemoveRole(gomock.Any(), gomock.Cond(func(req *minderv1.RemoveRoleRequest) bool {
				return req.GetContext().GetProject() == testProject &&
					req.GetRoleAssignment().GetRole() == "viewer" &&
					req.GetRoleAssignment().GetEmail() == testEmail
			})).
			Return(mockResp, nil)

		return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
	}

	tests := []cli.CmdTestCase{
		{
			Name:              "deny role from subject",
			Args:              []string{"project", "role", "deny", "-j", testProject, "-s", testSubject, "-r", testRole},
			MockSetup:         setupMockDenySubject,
			ExpectEmptyStdout: true,
			ExpectedStderr:    "Denied role successfully.\n",
		},
		{
			Name:              "delete invite by email",
			Args:              []string{"project", "role", "deny", "-j", testProject, "-e", testEmail, "-r", "viewer"},
			MockSetup:         setupMockDenyEmail,
			ExpectEmptyStdout: true,
			ExpectedStderr:    "Invite deleted successfully.\n",
		},
		{
			Name: "fails when required role flag is missing",
			Args: []string{"project", "role", "deny", "-j", testProject, "-s", testSubject},
			MockSetup: func(_ *testing.T, _ *gomock.Controller) context.Context {
				return context.Background()
			},
			ExpectedError: "required flag(s) \"role\" not set",
		},
		{
			Name: "fails when neither sub nor email provided",
			Args: []string{"project", "role", "deny", "-j", testProject, "-r", testRole},
			MockSetup: func(_ *testing.T, _ *gomock.Controller) context.Context {
				return context.Background()
			},
			ExpectedError: "at least one of the flags in the group [sub email] is required",
		},
		{
			Name: "fails when both sub and email are provided",
			Args: []string{"project", "role", "deny", "-j", testProject, "-s", testSubject, "-e", testEmail, "-r", testRole},
			MockSetup: func(_ *testing.T, _ *gomock.Controller) context.Context {
				return context.Background()
			},
			ExpectedError: "if any flags in the group [sub email] are set none of the others can be",
		},
		{
			Name: "server error handling on remove role from subject",
			Args: []string{"project", "role", "deny", "-j", testProject, "-s", testSubject, "-r", testRole},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)
				client.EXPECT().
					RemoveRole(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "internal server error"))
				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			ExpectedError: "internal server error",
		},
		{
			Name: "server error handling on delete invite",
			Args: []string{"project", "role", "deny", "-j", testProject, "-e", testEmail, "-r", "viewer"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)
				client.EXPECT().
					RemoveRole(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "invite not found"))
				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			ExpectedError: "invite not found",
		},
	}

	cli.RunCmdTests(t, tests, RoleCmd)
}
