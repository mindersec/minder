// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

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
func TestDeleteCommand(t *testing.T) {
	const entityID = "00000000-0000-0000-0000-000000000001"

	tests := []cli.CmdTestCase{
		{
			Name: "delete by id - success",
			Args: []string{"entity", "delete", "--id", entityID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				client.EXPECT().
					DeleteEntityById(gomock.Any(), gomock.Any()).
					Return(&minderv1.DeleteEntityByIdResponse{Id: entityID}, nil)
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			GoldenFileName: "delete_by_id.txt",
		},
		{
			Name:          "missing required id flag",
			Args:          []string{"entity", "delete"},
			ExpectedError: "required flag(s) \"id\" not set",
		},
		{
			Name: "grpc error",
			Args: []string{"entity", "delete", "--id", entityID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockEntityInstanceServiceClient(ctrl)
				client.EXPECT().
					DeleteEntityById(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "entity not found"))
				return cli.WithRPCClient[minderv1.EntityInstanceServiceClient](context.Background(), client)
			},
			ExpectedError: "entity not found",
		},
	}

	cli.RunCmdTests(t, tests, EntityCmd)
}
