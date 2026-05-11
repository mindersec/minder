// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package entity contains the entity logic for the control plane
package entity

import (
	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/cli/app"
)

// EntityCmd is the root command for the entity subcommands
var EntityCmd = &cobra.Command{
	Use:   "entity",
	Short: "Manage entities within a Minder project",
	Long: `Manage entities within a Minder project.

This command allows you to list, get, register, and delete entity instances
connected to Minder for security analysis and policy enforcement.`,
	Example: `
  # List entities
    minder entity list --type repository

  # Get an entity by ID
    minder entity get --id <entity-id>

  # Register an entity
    minder entity register --type repository --property github/repo_owner=owner --property github/repo_name=name

  # Delete an entity
    minder entity delete --id <entity-id>
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(EntityCmd)
	// Flags for all subcommands
	EntityCmd.PersistentFlags().StringP("provider", "p", "", "Name of the provider, i.e. github")
	EntityCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}
