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
func TestInviteGetCommand(t *testing.T) {
	const testCode = "test-invite-code-123"

	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		inviteClient := mockv1.NewMockInviteServiceClient(ctrl)

		mockResp := &minderv1.GetInviteDetailsResponse{}
		cli.LoadFixture(t, "mock_invite_get.json", mockResp)

		inviteClient.EXPECT().
			GetInviteDetails(gomock.Any(), gomock.Cond(func(r *minderv1.GetInviteDetailsRequest) bool {
				return r.GetCode() == testCode
			})).
			Return(mockResp, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.InviteServiceClient](context.Background(), inviteClient)
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "get invite details - table output",
			Args:           []string{"auth", "invite", "get", testCode, "-o", "table"},
			MockSetup:      setupSuccess,
			GoldenFileName: "invite_get.table",
		},
		{
			Name:           "get invite details - default output (table)",
			Args:           []string{"auth", "invite", "get", testCode},
			MockSetup:      setupSuccess,
			GoldenFileName: "invite_get.table",
		},
		{
			Name:           "get invite details - json output",
			Args:           []string{"auth", "invite", "get", testCode, "-o", "json"},
			MockSetup:      setupSuccess,
			GoldenFileName: "invite_get.json",
		},
		{
			Name:           "get invite details - yaml output",
			Args:           []string{"auth", "invite", "get", testCode, "-o", "yaml"},
			MockSetup:      setupSuccess,
			GoldenFileName: "invite_get.yaml",
		},
		{
			Name:          "fails on invalid output format",
			Args:          []string{"auth", "invite", "get", testCode, "-o", "invalid_format"},
			ExpectedError: "invalid argument",
		},
		{
			Name:          "fails when code argument is missing",
			Args:          []string{"auth", "invite", "get"},
			ExpectedError: "accepts 1 arg(s), received 0",
		},
		{
			Name: "server error handling",
			Args: []string{"auth", "invite", "get", testCode},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				inviteClient := mockv1.NewMockInviteServiceClient(ctrl)

				inviteClient.EXPECT().
					GetInviteDetails(gomock.Any(), gomock.Cond(func(r *minderv1.GetInviteDetailsRequest) bool {
						return r.GetCode() == testCode
					})).
					Return(nil, status.Error(codes.NotFound, "invitation not found")).
					Times(1)

				return cli.WithRPCClient[minderv1.InviteServiceClient](context.Background(), inviteClient)
			},
			ExpectedError: "invitation not found",
		},
	}

	cli.RunCmdTests(t, tests, auth.AuthCmd)
}
