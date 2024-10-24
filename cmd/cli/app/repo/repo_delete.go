// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/util/cli"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a repository",
	Long:  `The repo delete subcommand is used to delete a registered repository within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(deleteCommand),
}

// deleteCommand is the repo delete subcommand
func deleteCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewRepositoryServiceClient(conn)

	provider := viper.GetString("provider")
	project := viper.GetString("project")
	repoID := viper.GetString("id")
	name := viper.GetString("name")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// delete repo by id
	if repoID != "" {
		resp, err := client.DeleteRepositoryById(ctx, &minderv1.DeleteRepositoryByIdRequest{
			Context:      &minderv1.Context{Provider: &provider, Project: &project},
			RepositoryId: repoID,
		})
		if err != nil {
			return cli.MessageAndError("Error deleting repo by id", err)
		}
		cmd.Println("Successfully deleted repo with id:", resp.RepositoryId)
	} else {
		// delete repo by name
		resp, err := client.DeleteRepositoryByName(ctx, &minderv1.DeleteRepositoryByNameRequest{
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
