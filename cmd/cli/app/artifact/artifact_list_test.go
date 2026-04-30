// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package artifact

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
func TestArtifactListCommand(t *testing.T) {
	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()
		client := mockv1.NewMockArtifactServiceClient(ctrl)

		mockResp := &minderv1.ListArtifactsResponse{}
		cli.LoadFixture(t, "mock_artifact_list.json", mockResp)

		client.EXPECT().
			ListArtifacts(gomock.Any(), gomock.Any()).
			Return(mockResp, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.ArtifactServiceClient](context.Background(), client)
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "list artifacts - table output",
			Args:           []string{"artifact", "list", "-o", "table"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_list.table",
		},
		{
			Name:           "list artifacts - json output",
			Args:           []string{"artifact", "list", "-o", "json"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_list.json",
		},
		{
			Name:           "list artifacts - yaml output",
			Args:           []string{"artifact", "list", "-o", "yaml"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_list.yaml",
		},
		{
			Name:           "list artifacts with from filter",
			Args:           []string{"artifact", "list", "--from", "repository=org/repo", "-o", "table"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_list_from.table",
		},
		{
			Name: "server error handling",
			Args: []string{"artifact", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockArtifactServiceClient(ctrl)
				client.EXPECT().
					ListArtifacts(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "internal server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.ArtifactServiceClient](context.Background(), client)
			},
			ExpectedError: "internal server error",
		},
	}

	cli.RunCmdTests(t, tests, ArtifactCmd)
}
