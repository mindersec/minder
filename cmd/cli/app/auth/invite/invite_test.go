// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package invite

import (
	"testing"

	"github.com/mindersec/minder/cmd/cli/app/auth"
	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global state
func TestInviteRootCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:           "invite root command shows help",
			Args:           []string{"auth", "invite"},
			GoldenFileName: "invite_root.help",
		},
	}

	cli.RunCmdTests(t, tests, auth.AuthCmd)
}
