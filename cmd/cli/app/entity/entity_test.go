// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"testing"

	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global state
func TestEntityRootCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:           "entity root command shows help",
			Args:           []string{"entity"},
			GoldenFileName: "entity_root.help",
		},
	}

	cli.RunCmdTests(t, tests, EntityCmd)
}
