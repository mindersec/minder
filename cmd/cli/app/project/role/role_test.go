// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package role

import (
	"testing"

	"github.com/mindersec/minder/cmd/cli/app/project"
	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global state
func TestRoleRootCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:           "role root command shows help",
			Args:           []string{"project", "role"},
			GoldenFileName: "role_root.help",
		},
	}

	cli.RunCmdTests(t, tests, project.ProjectCmd)
}
