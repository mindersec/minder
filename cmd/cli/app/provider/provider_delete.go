// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a given provider available in a specific project",
	Long:  `The minder provider delete command deletes a given provider available in a specific project.`,
	RunE:  cli.GRPCClientWrapRunE(DeleteProviderCommand),
}

func init() {
	ProviderCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().StringP("name", "n", "", "Name of the provider to delete")
	deleteCmd.Flags().StringP("id", "i", "", "ID of the provider to delete")
	// We allow deleting by name or ID but not both. One of them must be specified.
	deleteCmd.MarkFlagsMutuallyExclusive("name", "id")
	deleteCmd.MarkFlagsOneRequired("name", "id")
}

// DeleteProviderCommand deletes the provider in a specific project
func DeleteProviderCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewProvidersServiceClient(conn)

	project := viper.GetString("project")
	name := viper.GetString("name")
	id := viper.GetString("id")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	if id != "" {
		resp, err := client.DeleteProviderByID(ctx, &minderv1.DeleteProviderByIDRequest{
			Context: &minderv1.Context{
				Project: &project,
			},
			Id: id,
		})
		if err != nil {
			return cli.MessageAndError("Error deleting provider by id", err)
		}
		cmd.Println("Successfully deleted provider with id:", resp.Id)
	} else {
		// delete provider by name
		resp, err := client.DeleteProvider(ctx, &minderv1.DeleteProviderRequest{
			Context: &minderv1.Context{Provider: &name, Project: &project},
		})
		if err != nil {
			return cli.MessageAndError("Error deleting provider by name", err)
		}
		cmd.Println("Successfully deleted provider with name:", resp.Name)
	}

	return nil
}
