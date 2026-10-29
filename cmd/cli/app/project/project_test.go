// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package project

import (
	"testing"

	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global state
func TestProjectRootCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:           "project root command shows help",
			Args:           []string{"project"},
			GoldenFileName: "project_root.help",
		},
	}

	cli.RunCmdTests(t, tests, ProjectCmd)
}
