// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package testutils provides shared helpers for CLI tests.
package testutils

import (
	"bytes"

	"github.com/spf13/cobra"
)

// RunCommand executes a cobra command and captures output.
func RunCommand(cmd *cobra.Command, args ...string) (string, error) {
	buf := new(bytes.Buffer)
	root := cmd.Root()
	if root != nil {
		// Some help/usage paths write via the root command streams.
		root.SetOut(buf)
		root.SetErr(buf)
	}

	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return buf.String(), err
}
