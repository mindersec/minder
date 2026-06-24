// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"context"
	"path/filepath"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestCreateCommand(t *testing.T) {
	fixtureFile := filepath.Join("fixture", "artifact_attestation.yaml")

	tests := []cli.CmdTestCase{
		{
			Name: "create success",
			Args: []string{"profile", "create", "-f", fixtureFile},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockProfile := &minderv1.Profile{}
				cli.LoadFixture(t, "mock_profile_attestation.json", mockProfile)

				client.EXPECT().
					CreateProfile(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateProfileResponse{Profile: mockProfile}, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "create_success.table",
		},
		{
			Name: "create failure connection error",
			Args: []string{"profile", "create", "-f", fixtureFile},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				client.EXPECT().
					CreateProfile(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "server error"))

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			ExpectedError: "server error",
		},
	}

	cli.RunCmdTests(t, tests, ProfileCmd)
}
