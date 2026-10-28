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
func TestRoleGrantCommand(t *testing.T) {
	const (
		testProject = "00000000-0000-0000-0000-000000000000"
		testSubject = "sub-12345"
		testEmail   = "charlie@example.com"
		testRole    = "editor"
	)

	setupMockGrantSubject := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()
		client := mockv1.NewMockPermissionsServiceClient(ctrl)
		mockResp := &minderv1.AssignRoleResponse{}
		cli.LoadFixture(t, "mock_role_grant_sub.json", mockResp)

		client.EXPECT().
			AssignRole(gomock.Any(), gomock.Cond(func(req *minderv1.AssignRoleRequest) bool {
				return req.GetContext().GetProject() == testProject &&
					req.GetRoleAssignment().GetRole() == testRole &&
					req.GetRoleAssignment().GetSubject() == testSubject
			})).
			Return(mockResp, nil)

		return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
	}

	setupMockGrantEmail := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()
		client := mockv1.NewMockPermissionsServiceClient(ctrl)
		mockResp := &minderv1.AssignRoleResponse{}
		cli.LoadFixture(t, "mock_role_grant_email.json", mockResp)

		client.EXPECT().
			AssignRole(gomock.Any(), gomock.Cond(func(req *minderv1.AssignRoleRequest) bool {
				return req.GetContext().GetProject() == testProject &&
					req.GetRoleAssignment().GetRole() == "viewer" &&
					req.GetRoleAssignment().GetEmail() == testEmail
			})).
			Return(mockResp, nil)

		return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "grant role to subject - table output",
			Args:           []string{"project", "role", "grant", "-j", testProject, "-s", testSubject, "-r", testRole, "-o", app.Table},
			MockSetup:      setupMockGrantSubject,
			GoldenFileName: "role_grant_sub.table",
		},
		{
			Name:           "grant role to subject - default output (table)",
			Args:           []string{"project", "role", "grant", "-j", testProject, "-s", testSubject, "-r", testRole},
			MockSetup:      setupMockGrantSubject,
			GoldenFileName: "role_grant_sub.table",
		},
		{
			Name:           "grant role to subject - json output",
			Args:           []string{"project", "role", "grant", "-j", testProject, "-s", testSubject, "-r", testRole, "-o", app.JSON},
			MockSetup:      setupMockGrantSubject,
			GoldenFileName: "role_grant_sub.json",
		},
		{
			Name:           "grant role to subject - yaml output",
			Args:           []string{"project", "role", "grant", "-j", testProject, "-s", testSubject, "-r", testRole, "-o", app.YAML},
			MockSetup:      setupMockGrantSubject,
			GoldenFileName: "role_grant_sub.yaml",
		},
		{
			Name:           "grant role via invite email - table output with invitation table",
			Args:           []string{"project", "role", "grant", "-j", testProject, "-e", testEmail, "-r", "viewer", "-o", app.Table},
			MockSetup:      setupMockGrantEmail,
			GoldenFileName: "role_grant_email.table",
		},
		{
			Name: "fails on invalid output format",
			Args: []string{"project", "role", "grant", "-j", testProject, "-s", testSubject, "-r", testRole, "-o", "invalid-format"},
			MockSetup: func(_ *testing.T, _ *gomock.Controller) context.Context {
				return context.Background()
			},
			ExpectedError: "invalid argument",
		},
		{
			Name: "fails when required role flag is missing",
			Args: []string{"project", "role", "grant", "-j", testProject, "-s", testSubject},
			MockSetup: func(_ *testing.T, _ *gomock.Controller) context.Context {
				return context.Background()
			},
			ExpectedError: "required flag(s) \"role\" not set",
		},
		{
			Name: "fails when neither sub nor email provided",
			Args: []string{"project", "role", "grant", "-j", testProject, "-r", testRole},
			MockSetup: func(_ *testing.T, _ *gomock.Controller) context.Context {
				return context.Background()
			},
			ExpectedError: "at least one of the flags in the group [sub email] is required",
		},
		{
			Name: "fails when both sub and email are provided",
			Args: []string{"project", "role", "grant", "-j", testProject, "-s", testSubject, "-e", testEmail, "-r", testRole},
			MockSetup: func(_ *testing.T, _ *gomock.Controller) context.Context {
				return context.Background()
			},
			ExpectedError: "if any flags in the group [sub email] are set none of the others can be",
		},
		{
			Name: "server error handling on assign role to subject",
			Args: []string{"project", "role", "grant", "-j", testProject, "-s", testSubject, "-r", testRole},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)
				client.EXPECT().
					AssignRole(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "internal server error"))
				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			ExpectedError: "internal server error",
		},
		{
			Name: "server error handling on create invite",
			Args: []string{"project", "role", "grant", "-j", testProject, "-e", testEmail, "-r", "viewer"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockPermissionsServiceClient(ctrl)
				client.EXPECT().
					AssignRole(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.AlreadyExists, "invitation already exists"))
				return cli.WithRPCClient[minderv1.PermissionsServiceClient](context.Background(), client)
			},
			ExpectedError: "invitation already exists",
		},
	}

	cli.RunCmdTests(t, tests, RoleCmd)
}
