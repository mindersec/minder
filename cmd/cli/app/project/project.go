// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package project is the root command for the project subcommands
package project

import (
	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/cli/app"
)

// ProjectCmd is the root command for the project subcommands
var ProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage project within a minder control plane",
	Long:  `The minder project commands manage projects within a minder control plane.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(ProjectCmd)
}
