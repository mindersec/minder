// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package role

import (
	"testing"

	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestRoleRootCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:           "role root command prints usage",
			Args:           []string{"project", "role"},
			GoldenFileName: "role_root_usage.txt",
		},
		{
			Name:           "role help flag prints usage",
			Args:           []string{"project", "role", "-h"},
			GoldenFileName: "role_help_usage.txt",
		},
	}

	cli.RunCmdTests(t, tests, RoleCmd)
}
