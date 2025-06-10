// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package image provides the root command for the image subcommands
package image

import "github.com/spf13/cobra"

// CmdImage is the root command for the container subcommands
func CmdImage() *cobra.Command {
	var rtCmd = &cobra.Command{
		Use:   "image",
		Short: "image provides utilities to test minder container image support",
	}

	rtCmd.AddCommand(CmdVerify())
	rtCmd.AddCommand(CmdList())

	return rtCmd
}
