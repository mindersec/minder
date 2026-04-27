// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

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
	const testID = "00000000-0000-0000-0000-000000000001"

	tests := []cli.CmdTestCase{
		{
			Name: "delete success",
			Args: []string{"profile", "delete", "-i", testID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				client.EXPECT().
					DeleteProfile(gomock.Any(), gomock.Any()).
					Return(&minderv1.DeleteProfileResponse{}, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "delete_success.txt",
		},
		{
			Name: "delete failure not found",
			Args: []string{"profile", "delete", "-i", testID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				client.EXPECT().
					DeleteProfile(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "profile not found"))

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			ExpectedError: "profile not found",
		},
	}

	cli.RunCmdTests(t, tests, ProfileCmd)
}
