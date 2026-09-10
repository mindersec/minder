// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package project

import (
	"testing"

	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestProjectRootCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:           "project root command prints usage",
			Args:           []string{"project"},
			GoldenFileName: "project_root_usage.txt",
		},
		{
			Name:           "project help flag prints usage",
			Args:           []string{"project", "-h"},
			GoldenFileName: "project_help_usage.txt",
		},
	}

	cli.RunCmdTests(t, tests, ProjectCmd)
}
