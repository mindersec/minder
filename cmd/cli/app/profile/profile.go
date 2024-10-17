// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package profile provides the CLI subcommand for managing profiles
package profile

import (
	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/cli/app"
)

// ProfileCmd is the root command for the profile subcommands
var ProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage profiles",
	Long:  `The profile subcommands allows the management of profiles within Minder.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(ProfileCmd)
	// Flags for all subcommands
	ProfileCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}
