// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package quickstart

import (
	"testing"

	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestQuickstartCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:          "rejects unexpected positional arguments",
			Args:          []string{"quickstart", "unexpected-arg"},
			ExpectedError: `unknown command "unexpected-arg" for "minder quickstart"`,
		},
	}

	cli.RunCmdTests(t, tests, cmd)
}
