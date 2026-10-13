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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestArtifactListCommand(t *testing.T) {
	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		artifactClient := mockv1.NewMockArtifactServiceClient(ctrl)

		artifactListResp := &minderv1.ListArtifactsResponse{}
		cli.LoadFixture(t, "mock_artifact_list.json", artifactListResp)

		artifactClient.EXPECT().
			ListArtifacts(gomock.Any(), gomock.Any()).
			Return(artifactListResp, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.ArtifactServiceClient](context.Background(), artifactClient)
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "list artifacts - table output",
			Args:           []string{"artifact", "list", "-o", "table"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_list.table",
		},
		{
			Name:           "list artifacts - default output (table)",
			Args:           []string{"artifact", "list"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_list.table",
		},
		{
			Name:           "list artifacts - yaml output",
			Args:           []string{"artifact", "list", "-o", "yaml"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_list.yaml",
		},
		{
			Name: "list artifacts - empty table output",
			Args: []string{"artifact", "list", "-o", "table"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				artifactClient := mockv1.NewMockArtifactServiceClient(ctrl)

				emptyResp := &minderv1.ListArtifactsResponse{}
				cli.LoadFixture(t, "mock_artifact_list_empty.json", emptyResp)

				artifactClient.EXPECT().
					ListArtifacts(gomock.Any(), gomock.Any()).
					Return(emptyResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.ArtifactServiceClient](context.Background(), artifactClient)
			},
			GoldenFileName: "artifact_list_empty.table",
		},
		{
			Name: "list artifacts - with from filter",
			Args: []string{"artifact", "list", "--from", "repository=org/repo"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				artifactClient := mockv1.NewMockArtifactServiceClient(ctrl)

				artifactListResp := &minderv1.ListArtifactsResponse{}
				cli.LoadFixture(t, "mock_artifact_list.json", artifactListResp)

				artifactClient.EXPECT().
					ListArtifacts(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *minderv1.ListArtifactsRequest, _ ...grpc.CallOption) (*minderv1.ListArtifactsResponse, error) {
						if req.GetFrom() != "repository=org/repo" {
							t.Fatalf("expected from filter repository=org/repo, got %s", req.GetFrom())
						}
						return artifactListResp, nil
					}).
					Times(1)

				return cli.WithRPCClient[minderv1.ArtifactServiceClient](context.Background(), artifactClient)
			},
			GoldenFileName: "artifact_list.table",
		},
		{
			Name:          "fails on invalid output format",
			Args:          []string{"artifact", "list", "-o", "invalid_format"},
			ExpectedError: "invalid argument",
		},
		{
			Name: "server error handling",
			Args: []string{"artifact", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				artifactClient := mockv1.NewMockArtifactServiceClient(ctrl)

				artifactClient.EXPECT().
					ListArtifacts(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.ArtifactServiceClient](context.Background(), artifactClient)
			},
			ExpectedError: "server error",
		},
	}

	cli.RunCmdTests(t, tests, ArtifactCmd)
}
