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
func TestInviteListCommand(t *testing.T) {
	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		userClient := mockv1.NewMockUserServiceClient(ctrl)

		mockResp := &minderv1.ListInvitationsResponse{}
		cli.LoadFixture(t, "mock_invite_list.json", mockResp)

		userClient.EXPECT().
			ListInvitations(gomock.Any(), gomock.Any()).
			Return(mockResp, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.UserServiceClient](context.Background(), userClient)
	}

	setupEmpty := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		userClient := mockv1.NewMockUserServiceClient(ctrl)

		userClient.EXPECT().
			ListInvitations(gomock.Any(), gomock.Any()).
			Return(&minderv1.ListInvitationsResponse{}, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.UserServiceClient](context.Background(), userClient)
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "list invitations - table output",
			Args:           []string{"auth", "invite", "list", "-o", "table"},
			MockSetup:      setupSuccess,
			GoldenFileName: "invite_list.table",
		},
		{
			Name:           "list invitations - default output (table)",
			Args:           []string{"auth", "invite", "list"},
			MockSetup:      setupSuccess,
			GoldenFileName: "invite_list.table",
		},
		{
			Name:           "list invitations - json output",
			Args:           []string{"auth", "invite", "list", "-o", "json"},
			MockSetup:      setupSuccess,
			GoldenFileName: "invite_list.json",
		},
		{
			Name:           "list invitations - yaml output",
			Args:           []string{"auth", "invite", "list", "-o", "yaml"},
			MockSetup:      setupSuccess,
			GoldenFileName: "invite_list.yaml",
		},
		{
			Name:           "list invitations - empty table output",
			Args:           []string{"auth", "invite", "list", "-o", "table"},
			MockSetup:      setupEmpty,
			GoldenFileName: "invite_list_empty.table",
		},
		{
			Name:          "fails on invalid output format",
			Args:          []string{"auth", "invite", "list", "-o", "invalid_format"},
			ExpectedError: "invalid argument",
		},
		{
			Name: "server error handling",
			Args: []string{"auth", "invite", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				userClient := mockv1.NewMockUserServiceClient(ctrl)

				userClient.EXPECT().
					ListInvitations(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.UserServiceClient](context.Background(), userClient)
			},
			ExpectedError: "server error",
		},
	}

	cli.RunCmdTests(t, tests, auth.AuthCmd)
}
