// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"testing"

	"github.com/mindersec/minder/cmd/cli/app/profile"
	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestProfileStatusRootCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:           "profile status root command prints usage",
			Args:           []string{"profile", "status"},
			GoldenFileName: "status_root.help",
		},
		{
			Name:           "profile status root command with help flag",
			Args:           []string{"profile", "status", "--help"},
			GoldenFileName: "status_root_help.help",
		},
	}

	cli.RunCmdTests(t, tests, profile.ProfileCmd)
}
