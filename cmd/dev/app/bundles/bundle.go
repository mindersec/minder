// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package bundles contains logic relating to mindpak bundles
package bundles

import "github.com/spf13/cobra"

// CmdBundle is the root command for the container subcommands
func CmdBundle() *cobra.Command {
	var rtCmd = &cobra.Command{
		Use:   "bundle",
		Short: "container provides utilities to create mindpak bundles",
	}

	rtCmd.AddCommand(CmdBuild())

	return rtCmd
}
