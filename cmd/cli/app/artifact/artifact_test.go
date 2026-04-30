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

// (no profile client injected; tests skip profile evaluation when RPC client is injected)

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

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestArtifactGetCommand(t *testing.T) {
	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()
		client := mockv1.NewMockArtifactServiceClient(ctrl)

		mockResp := &minderv1.GetArtifactByIdResponse{}
		cli.LoadFixture(t, "mock_artifact_get.json", mockResp)

		client.EXPECT().
			GetArtifactById(gomock.Any(), gomock.Any()).
			Return(mockResp, nil).
			Times(1)

		ctx := cli.WithRPCClient[minderv1.ArtifactServiceClient](context.Background(), client)
		return ctx
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "get artifact - table output",
			Args:           []string{"artifact", "get", "-i", "111", "-o", "table"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_get.table",
		},
		{
			Name:           "get artifact - json output",
			Args:           []string{"artifact", "get", "-i", "111", "-o", "json"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_get.json",
		},
		{
			Name:           "get artifact - yaml output",
			Args:           []string{"artifact", "get", "-i", "111", "-o", "yaml"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_get.yaml",
		},
		{
			Name: "server error handling",
			Args: []string{"artifact", "get", "-i", "111"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockArtifactServiceClient(ctrl)
				client.EXPECT().
					GetArtifactById(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "artifact not found")).
					Times(1)

				return cli.WithRPCClient[minderv1.ArtifactServiceClient](context.Background(), client)
			},
			ExpectedError: "artifact not found",
		},
	}

	cli.RunCmdTests(t, tests, ArtifactCmd)
}
