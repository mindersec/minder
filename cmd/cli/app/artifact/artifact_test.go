// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package artifact

import (
	"testing"

	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global state
func TestArtifactRootCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:           "artifact root command shows help",
			Args:           []string{"artifact"},
			GoldenFileName: "artifact_root.help",
		},
	}

	cli.RunCmdTests(t, tests, ArtifactCmd)
}
