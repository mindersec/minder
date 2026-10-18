// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestRunCmdTests_StdoutAndStderrSeparation(t *testing.T) {

	rootCmd := &cobra.Command{Use: "root"}
	subCmd := &cobra.Command{
		Use: "testcmd",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && args[0] == "error" {
				cmd.PrintErrln("standard error output")
				return errors.New("command failed")
			}
			cmd.Println("standard out output")
			cmd.PrintErrln("warning message on stderr")
			return nil
		},
	}
	rootCmd.AddCommand(subCmd)

	tests := []CmdTestCase{
		{
			Name:           "stdout and stderr captured independently",
			Args:           []string{"testcmd"},
			ExpectedStderr: "warning message on stderr",
		},
		{
			Name:           "command returns error and stderr",
			Args:           []string{"testcmd", "error"},
			ExpectedStderr: "standard error output",
			ExpectedError:  "command failed",
		},
	}

	RunCmdTests(t, tests, rootCmd)
}
