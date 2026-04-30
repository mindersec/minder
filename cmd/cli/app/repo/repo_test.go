// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"testing"

	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestRootCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:           "repo root command prints usage",
			Args:           []string{"repo"},
			GoldenFileName: "repo_root_usage.txt",
		},
		{
			Name:           "repo help flag prints usage",
			Args:           []string{"repo", "-h"},
			GoldenFileName: "repo_help.txt",
		},
	}

	cli.RunCmdTests(t, tests, RepoCmd)
}
