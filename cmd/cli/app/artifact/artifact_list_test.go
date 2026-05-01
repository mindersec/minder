// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package artifact

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Uses global viper/Stdout state
func TestArtifactListCommand(t *testing.T) {
	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		artifactClient := mockv1.NewMockArtifactServiceClient(ctrl)

		listResp := &minderv1.ListArtifactsResponse{}
		cli.LoadFixture(t, "mock_artifact_list.json", listResp)

		artifactClient.EXPECT().
			ListArtifacts(gomock.Any(), gomock.Any()).
			Return(listResp, nil).
			Times(1)

		ctx := cli.WithCLIClient[minderv1.ArtifactServiceClient](context.Background(), artifactClient)
		return ctx
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "list artifacts - json output",
			Args:           []string{"artifact", "list", "-o", "json"},
			MockSetup:      setupSuccess,
			GoldenFileName: "artifact_list.json",
		},
	}

	cli.RunCmdTests(t, tests, ArtifactCmd)
}
