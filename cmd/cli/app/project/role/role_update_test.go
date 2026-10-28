// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package role

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
func TestRoleUpdateCommand(t *testing.T) {
	const (
		testProject = "00000000-0000-0000-0000-000000000000"
		testSubject = "sub-12345"
		testEmail   = "charlie@example.com"
		testRole    = "admin"
	)

	setupMockUpdateSubject := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()
		client := mockv1.NewMockPermissionsServiceClient(ctrl)
		mockResp := &minderv1.UpdateRoleResponse{}
		cli.LoadFixture(t, "mock_role_update_sub.json", mockResp)

		client.EXPECT().
			UpdateRole(gomock.Any(), gomock.Cond(func(req *minderv1.UpdateRoleRequest) bool {
				return req.GetContext().GetProject() == testProject &&
					len(req.GetRoles()) == 1 && req.GetRoles()[0] == testRole &&
					req.GetSubject() == testSubject
			})).
			Return(mockResp, nil)

		return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
	}

	setupMockUpdateEmail := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()
		client := mockv1.NewMockPermissionsServiceClient(ctrl)
		mockResp := &minderv1.UpdateRoleResponse{}
		cli.LoadFixture(t, "mock_role_update_email.json", mockResp)

		client.EXPECT().
			UpdateRole(gomock.Any(), gomock.Cond(func(req *minderv1.UpdateRoleRequest) bool {
				return req.GetContext().GetProject() == testProject &&
					len(req.GetRoles()) == 1 && req.GetRoles()[0] == testRole &&
					req.GetEmail() == testEmail
			})).
			Return(mockResp, nil)

		return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "update role for subject - table output",
			Args:           []string{"project", "role", "update", "-j", testProject, "-s", testSubject, "-r", testRole, "-o", app.Table},
			MockSetup:      setupMockUpdateSubject,
			GoldenFileName: "role_update_sub.table",
		},
		{
			Name:           "update role for subject - default output (table)",
			Args:           []string{"project", "role", "update", "-j", testProject, "-s", testSubject, "-r", testRole},
			MockSetup:      setupMockUpdateSubject,
			GoldenFileName: "role_update_sub.table",
		},
		{
			Name:           "update role for subject - json output",
			Args:           []string{"project", "role", "update", "-j", testProject, "-s", testSubject, "-r", testRole, "-o", app.JSON},
			MockSetup:      setupMockUpdateSubject,
			GoldenFileName: "role_update_sub.json",
		},
		{
			Name:           "update role for subject - yaml output",
			Args:           []string{"project", "role", "update", "-j", testProject, "-s", testSubject, "-r", testRole, "-o", app.YAML},
			MockSetup:      setupMockUpdateSubject,
			GoldenFileName: "role_update_sub.yaml",
		},
		{
			Name:           "update role for invite email - table output with invite details",
			Args:           []string{"project", "role", "update", "-j", testProject, "-e", testEmail, "-r", testRole, "-o", app.Table},
			MockSetup:      setupMockUpdateEmail,
			GoldenFileName: "role_update_email.table",
		},
		{
			Name: "fails on invalid output format",
			Args: []string{"project", "role", "update", "-j", testProject, "-s", testSubject, "-r", testRole, "-o", "invalid-format"},
			MockSetup: func(_ *testing.T, _ *gomock.Controller) context.Context {
				return context.Background()
			},
			ExpectedError: "invalid argument",
		},
		{
			Name: "fails when neither sub nor email provided",
			Args: []string{"project", "role", "update", "-j", testProject, "-r", testRole},
			MockSetup: func(_ *testing.T, _ *gomock.Controller) context.Context {
				return context.Background()
			},
			ExpectedError: "at least one of the flags in the group [sub email] is required",
		},
		{
			Name: "fails when both sub and email are provided",
			Args: []string{"project", "role", "update", "-j", testProject, "-s", testSubject, "-e", testEmail, "-r", testRole},
			MockSetup: func(_ *testing.T, _ *gomock.Controller) context.Context {
				return context.Background()
			},
			ExpectedError: "if any flags in the group [sub email] are set none of the others can be",
		},
		{
			Name: "server error handling on update role for subject",
			Args: []string{"project", "role", "update", "-j", testProject, "-s", testSubject, "-r", testRole},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)
				client.EXPECT().
					UpdateRole(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "internal server error"))
				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			ExpectedError: "internal server error",
		},
		{
			Name: "server error handling on update invite",
			Args: []string{"project", "role", "update", "-j", testProject, "-e", testEmail, "-r", testRole},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)
				client.EXPECT().
					UpdateRole(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "invite not found"))
				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			ExpectedError: "invite not found",
		},
	}

	cli.RunCmdTests(t, tests, RoleCmd)
}
