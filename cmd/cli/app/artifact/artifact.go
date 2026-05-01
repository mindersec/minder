// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package artifact provides the artifact subcommands
package artifact

import (
	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/cli/app"
)

// ArtifactCmd is the artifact subcommand
var ArtifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Manage artifacts within a minder control plane",
	Long:  `The minder artifact commands allow the management of artifacts within a minder control plane`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(ArtifactCmd)
	// Flags for all subcommands
	ArtifactCmd.PersistentFlags().StringP("provider", "p", "", "Name of the provider, i.e. github")
	ArtifactCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}
