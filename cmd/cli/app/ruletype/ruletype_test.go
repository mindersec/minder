// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"testing"

	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global state
func TestRuleTypeRootCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:           "ruletype root command shows help",
			Args:           []string{"ruletype"},
			GoldenFileName: "ruletype_root.help",
		},
	}

	cli.RunCmdTests(t, tests, ruleTypeCmd)
}
