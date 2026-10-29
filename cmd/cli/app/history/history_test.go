// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"testing"

	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global state
func TestHistoryRootCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:           "history root command shows help",
			Args:           []string{"history"},
			GoldenFileName: "history_root.help",
		},
	}

	cli.RunCmdTests(t, tests, historyCmd)
}
