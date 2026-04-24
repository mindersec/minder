// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a repository",
	Long:  `The repo delete subcommand is used to delete a registered repository within Minder.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}
		return nil
	},
	RunE: deleteCommand,
}

// deleteCommand is the repo delete subcommand
func deleteCommand(cmd *cobra.Command, _ []string) error {
	provider := viper.GetString("provider")
	project := viper.GetString("project")
	repoID := viper.GetString("id")
	name := viper.GetString("name")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, cleanup, err := getRepoClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer cleanup()

	// delete repo by id
	if repoID != "" {
		resp, err := client.DeleteRepositoryById(cmd.Context(), &minderv1.DeleteRepositoryByIdRequest{
			Context:      &minderv1.Context{Provider: &provider, Project: &project},
			RepositoryId: repoID,
		})
		if err != nil {
			return cli.MessageAndError("Error deleting repo by id", err)
		}
		cmd.Println("Successfully deleted repo with id:", resp.RepositoryId)
	} else {
		// delete repo by name
		resp, err := client.DeleteRepositoryByName(cmd.Context(), &minderv1.DeleteRepositoryByNameRequest{
			Context: &minderv1.Context{Provider: &provider, Project: &project},
			Name:    name,
		})
		if err != nil {
			return cli.MessageAndError("Error deleting repo by name", err)
		}
		cmd.Println("Successfully deleted repo with name:", resp.Name)
	}
	return nil
}

func init() {
	RepoCmd.AddCommand(deleteCmd)
	// Flags
	deleteCmd.Flags().StringP("name", "n", "", "Name of the repository (owner/name format) to delete")
	deleteCmd.Flags().StringP("id", "i", "", "ID of the repo to delete")
	// Required
	deleteCmd.MarkFlagsOneRequired("name", "id")
	deleteCmd.MarkFlagsMutuallyExclusive("name", "id")
}
