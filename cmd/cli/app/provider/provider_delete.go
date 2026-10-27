// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a given provider available in a specific project",
	Long:  `The minder provider delete command deletes a given provider available in a specific project.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}
		return nil
	},
	RunE: deleteProviderCommand,
}

func init() {
	ProviderCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().StringP("name", "n", "", "Name of the provider to delete")
	deleteCmd.Flags().StringP("id", "i", "", "ID of the provider to delete")
	deleteCmd.MarkFlagsMutuallyExclusive("name", "id")
	deleteCmd.MarkFlagsOneRequired("name", "id")
}

// deleteProviderCommand deletes the provider in a specific project
func deleteProviderCommand(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	client, cleanup, err := cli.GetCLIClient(cmd, minderv1.NewProvidersServiceClient)
	if err != nil {
		return err
	}
	defer cleanup()

	project := viper.GetString("project")
	name := viper.GetString("name")
	id := viper.GetString("id")

	if id != "" {
		resp, err := client.DeleteProviderByID(cmd.Context(), &minderv1.DeleteProviderByIDRequest{
			Context: &minderv1.Context{
				Project: &project,
			},
			Id: id,
		})
		if err != nil {
			return cli.MessageAndError("Error deleting provider by id", err)
		}
		cmd.PrintErrln("Successfully deleted provider with id:", resp.Id)
	} else {
		resp, err := client.DeleteProvider(cmd.Context(), &minderv1.DeleteProviderRequest{
			Context: &minderv1.Context{Provider: &name, Project: &project},
		})
		if err != nil {
			return cli.MessageAndError("Error deleting provider by name", err)
		}
		cmd.PrintErrln("Successfully deleted provider with name:", resp.Name)
	}

	return nil
}
