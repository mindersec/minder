// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package role is the root command for the role subcommands
package role

import (
	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/cli/app/project"
)

// RoleCmd is the root command for the project subcommands
var RoleCmd = &cobra.Command{
	Use:   "role",
	Short: "Manage roles within a minder control plane",
	Long:  `The minder role commands manage permissions within a minder control plane.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

// inviteBearerWarning is printed to stderr when an invitation code is displayed.
const inviteBearerWarning = "Warning: Invitation codes are one-time bearer tokens. " +
	"Anyone with access to this code or URL can use it to join the project. " +
	"Share it only over trusted channels."

func init() {
	project.ProjectCmd.AddCommand(RoleCmd)
	RoleCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}
