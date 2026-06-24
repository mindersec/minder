// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"testing"

	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestProfileRootCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:           "profile root command prints usage",
			Args:           []string{"profile"},
			GoldenFileName: "profile_root.txt",
		},
		{
			Name:           "profile root command with help flag",
			Args:           []string{"profile", "--help"},
			GoldenFileName: "profile_root_help.txt",
		},
	}

	cli.RunCmdTests(t, tests, ProfileCmd)
}
