// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package artifact

// JSON output is not tested because protojson formatting is not stable across environments.
// This can cause flaky tests due to spacing differences.
// See maintainer discussion in PR #6417.

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

const testArtifactID = "3f2504e0-4f89-41d3-9a0c-0305e82c3301"

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestArtifactGetCommand(t *testing.T) {
	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		artifactClient := mockv1.NewMockArtifactServiceClient(ctrl)
		profileClient := mockv1.NewMockProfileServiceClient(ctrl)

		artifactResp := &minderv1.GetArtifactByIdResponse{}
		cli.LoadFixture(t, "mock_artifact_get.json", artifactResp)

		artifactClient.EXPECT().
			GetArtifactById(gomock.Any(), gomock.Cond(func(req *minderv1.GetArtifactByIdRequest) bool {
				return req.GetId() == testArtifactID
			})).
			Return(artifactResp, nil).
			Times(1)

		listProfilesResp := &minderv1.ListProfilesResponse{}
		cli.LoadFixture(t, "mock_list_profiles.json", listProfilesResp)
		profileClient.EXPECT().
			ListProfiles(gomock.Any(), gomock.Any()).
			Return(listProfilesResp, nil).
			Times(1)

		statusResp := &minderv1.GetProfileStatusByNameResponse{}
		cli.LoadFixture(t, "mock_profile_status.json", statusResp)
		profileClient.EXPECT().
			GetProfileStatusByName(gomock.Any(), gomock.Cond(func(req *minderv1.GetProfileStatusByNameRequest) bool {
				return req.GetEntity().GetId() == testArtifactID
			})).
			Return(statusResp, nil).
			Times(1)

		ctx := cli.WithRPCClient[minderv1.ArtifactServiceClient](context.Background(), artifactClient)
		ctx = cli.WithRPCClient[minderv1.ProfileServiceClient](ctx, profileClient)
		return ctx
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "get artifact - table output",
			Args:           []string{"artifact", "get", "-i", testArtifactID, "-o", "table"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_get.table",
		},
		{
			Name:           "get artifact - yaml output",
			Args:           []string{"artifact", "get", "-i", testArtifactID, "-o", "yaml"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_get.yaml",
		},
		{
			Name: "server error handling",
			Args: []string{"artifact", "get", "-i", testArtifactID},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				artifactClient := mockv1.NewMockArtifactServiceClient(ctrl)
				profileClient := mockv1.NewMockProfileServiceClient(ctrl)

				artifactClient.EXPECT().
					GetArtifactById(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "artifact not found")).
					Times(1)

				ctx := cli.WithRPCClient[minderv1.ArtifactServiceClient](context.Background(), artifactClient)
				ctx = cli.WithRPCClient[minderv1.ProfileServiceClient](ctx, profileClient)
				return ctx
			},
			ExpectedError: "artifact not found",
		},
	}

	cli.RunCmdTests(t, tests, ArtifactCmd)
}
