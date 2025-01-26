// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasource

import (
	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/cli/app"
)

// DataSourceCmd is the root command for the data source subcommands
var DataSourceCmd = &cobra.Command{
	Use:   "datasource",
	Short: "Manage data sources within a minder control plane",
	Long:  "The data source subcommand allows the management of data sources within Minder.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(DataSourceCmd)
	// Flags for all subcommands
	DataSourceCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}
