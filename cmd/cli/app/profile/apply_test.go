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
func TestApplyCommand(t *testing.T) {
	fixtureAttestation := filepath.Join("fixture", "artifact_attestation.yaml")
	fixtureDependabot := filepath.Join("fixture", "dependabot_go.yaml")

	tests := []cli.CmdTestCase{
		{
			Name: "apply create new profile via flag",
			Args: []string{"profile", "apply", "-f", fixtureAttestation},
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
			GoldenFileName: "apply_create.table",
		},
		{
			Name: "apply update existing profile via positional arg",
			Args: []string{"profile", "apply", fixtureDependabot},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockProfile := &minderv1.Profile{}
				cli.LoadFixture(t, "mock_profile_dependabot.json", mockProfile)

				client.EXPECT().
					CreateProfile(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.AlreadyExists, "already exists"))

				client.EXPECT().
					UpdateProfile(gomock.Any(), gomock.Any()).
					Return(&minderv1.UpdateProfileResponse{Profile: mockProfile}, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "apply_update.table",
		},
		{
			Name: "apply multiple files (create and update)",
			Args: []string{"profile", "apply", "-f", fixtureAttestation, fixtureDependabot},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProfileServiceClient(ctrl)

				mockProfile1 := &minderv1.Profile{}
				cli.LoadFixture(t, "mock_profile_attestation.json", mockProfile1)

				mockProfile2 := &minderv1.Profile{}
				cli.LoadFixture(t, "mock_profile_dependabot.json", mockProfile2)

				client.EXPECT().
					CreateProfile(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateProfileResponse{Profile: mockProfile1}, nil)

				client.EXPECT().
					CreateProfile(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.AlreadyExists, "already exists"))

				client.EXPECT().
					UpdateProfile(gomock.Any(), gomock.Any()).
					Return(&minderv1.UpdateProfileResponse{Profile: mockProfile2}, nil)

				return cli.WithRPCClient[minderv1.ProfileServiceClient](context.Background(), client)
			},
			GoldenFileName: "apply_multiple.table",
		},
		{
			Name:          "no files specified",
			Args:          []string{"profile", "apply"},
			ExpectedError: "no files specified",
		},
	}

	cli.RunCmdTests(t, tests, ProfileCmd)
}
