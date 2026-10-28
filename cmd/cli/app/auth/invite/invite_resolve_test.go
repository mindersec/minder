// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package invite

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/cmd/cli/app/auth"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestInviteAcceptCommand(t *testing.T) {
	const testCode = "test-invite-code-123"

	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		userClient := mockv1.NewMockUserServiceClient(ctrl)

		userClient.EXPECT().
			ResolveInvitation(gomock.Any(), gomock.Cond(func(r *minderv1.ResolveInvitationRequest) bool {
				return r.GetAccept() && r.GetCode() == testCode
			})).
			Return(&minderv1.ResolveInvitationResponse{
				Email:          "alice@example.com",
				Role:           "admin",
				ProjectDisplay: "test-project",
			}, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.UserServiceClient](context.Background(), userClient)
	}

	tests := []cli.CmdTestCase{
		{
			Name:              "accept invitation successfully",
			Args:              []string{"auth", "invite", "accept", testCode},
			MockSetup:         setupSuccess,
			ExpectEmptyStdout: true,
			ExpectedStderr:    "Invitation test-invite-code-123 for alice@example.com to become admin of project test-project was accepted!\n",
		},
		{
			Name:          "fails when code argument is missing",
			Args:          []string{"auth", "invite", "accept"},
			ExpectedError: "accepts 1 arg(s), received 0",
		},
		{
			Name: "server error handling",
			Args: []string{"auth", "invite", "accept", testCode},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				userClient := mockv1.NewMockUserServiceClient(ctrl)

				userClient.EXPECT().
					ResolveInvitation(gomock.Any(), gomock.Cond(func(r *minderv1.ResolveInvitationRequest) bool {
						return r.GetAccept() && r.GetCode() == testCode
					})).
					Return(nil, status.Error(codes.NotFound, "invitation not found")).
					Times(1)

				return cli.WithRPCClient[minderv1.UserServiceClient](context.Background(), userClient)
			},
			ExpectedError: "invitation not found",
		},
	}

	cli.RunCmdTests(t, tests, auth.AuthCmd)
}

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestInviteDeclineCommand(t *testing.T) {
	const testCode = "test-invite-code-456"

	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		userClient := mockv1.NewMockUserServiceClient(ctrl)

		userClient.EXPECT().
			ResolveInvitation(gomock.Any(), gomock.Cond(func(r *minderv1.ResolveInvitationRequest) bool {
				return !r.GetAccept() && r.GetCode() == testCode
			})).
			Return(&minderv1.ResolveInvitationResponse{
				Email:          "bob@example.com",
				Role:           "editor",
				ProjectDisplay: "test-project",
			}, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.UserServiceClient](context.Background(), userClient)
	}

	tests := []cli.CmdTestCase{
		{
			Name:              "decline invitation successfully",
			Args:              []string{"auth", "invite", "decline", testCode},
			MockSetup:         setupSuccess,
			ExpectEmptyStdout: true,
			ExpectedStderr:    "Invitation test-invite-code-456 for bob@example.com to become editor of project test-project was declined!\n",
		},
		{
			Name:          "fails when code argument is missing",
			Args:          []string{"auth", "invite", "decline"},
			ExpectedError: "accepts 1 arg(s), received 0",
		},
		{
			Name: "server error handling",
			Args: []string{"auth", "invite", "decline", testCode},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				userClient := mockv1.NewMockUserServiceClient(ctrl)

				userClient.EXPECT().
					ResolveInvitation(gomock.Any(), gomock.Cond(func(r *minderv1.ResolveInvitationRequest) bool {
						return !r.GetAccept() && r.GetCode() == testCode
					})).
					Return(nil, status.Error(codes.NotFound, "invitation not found")).
					Times(1)

				return cli.WithRPCClient[minderv1.UserServiceClient](context.Background(), userClient)
			},
			ExpectedError: "invitation not found",
		},
	}

	cli.RunCmdTests(t, tests, auth.AuthCmd)
}
