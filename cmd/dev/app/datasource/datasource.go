// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package datasource provides the root command for the datasource subcommands
package datasource

import "github.com/spf13/cobra"

// CmdDataSource is the root command for the datasource subcommands
func CmdDataSource() *cobra.Command {
	var rtCmd = &cobra.Command{
		Use:   "datasource",
		Short: "datasource provides utilities for testing and working with data sources",
	}

	rtCmd.AddCommand(CmdGenerate())

	return rtCmd
}
