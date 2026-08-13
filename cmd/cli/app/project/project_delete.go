// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package project

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// projectDeleteCmd is the command for deleting sub-projects
var projectDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a sub-project within a minder control plane",
	Long:  `Delete a sub-project within a minder control plane`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return cli.MessageAndError("Error binding flags", err)
		}
		return nil
	},
	RunE: deleteCommand,
}

// deleteCommand is the command for listing projects
func deleteCommand(cmd *cobra.Command, _ []string) error {
	client, cleanup, err := GetProjectsClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error getting client", err)
	}
	defer cleanup()

	project := viper.GetString("project")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	resp, err := client.DeleteProject(cmd.Context(), &minderv1.DeleteProjectRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
	})
	if err != nil {
		return cli.MessageAndError("Error deleting sub-project", err)
	}

	cmd.Println("Successfully deleted project with id:", resp.ProjectId)

	return nil
}

func init() {
	ProjectCmd.AddCommand(projectDeleteCmd)

	projectDeleteCmd.Flags().StringP("project", "j", "", "The sub-project to delete")
	// mark as required
	if err := projectDeleteCmd.MarkFlagRequired("project"); err != nil {
		panic(err)
	}
}
