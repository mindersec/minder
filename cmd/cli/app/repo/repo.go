// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package repo contains the repo logic for the control plane
package repo

import (
	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/cli/app"
)

// RepoCmd is the root command for the repo subcommands
var RepoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories within a Minder project",
	Long: `Manage repositories within a Minder project.

This command allows you to list, add, and manage repositories
connected to Minder for security analysis and policy enforcement.`,
	Example: `
  # List repositories
    minder repo list

  # Register a repository
    minder repo register --name my-repo --provider github

  # Delete a repository
    minder repo delete --name my-repo
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(RepoCmd)
	// Flags for all subcommands
	RepoCmd.PersistentFlags().StringP("provider", "p", "", "Name of the provider, i.e. github")
	RepoCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}
