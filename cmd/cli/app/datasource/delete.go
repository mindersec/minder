// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasource

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a data source",
	Long:  `The datasource delete subcommand lets you delete a data source within Minder.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}

		format := viper.GetString("output")

		// Ensure the output format is supported
		if !app.IsOutputFormatSupported(format) {
			return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
		}

		return nil
	},
	RunE: deleteCommand,
}

// deleteCommand is the datasource delete subcommand
func deleteCommand(cmd *cobra.Command, _ []string) error {
	client, cleanup, err := cli.GetCLIClient(cmd, minderv1.NewDataSourceServiceClient)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer cleanup()

	project := viper.GetString("project")
	format := viper.GetString("output")
	id := viper.GetString("id")
	name := viper.GetString("name")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	if id == "" && name == "" {
		return fmt.Errorf("either id or name must be specified")
	}

	if id != "" {
		_, err = client.DeleteDataSourceById(cmd.Context(), &minderv1.DeleteDataSourceByIdRequest{
			Context: &minderv1.ContextV2{
				ProjectId: project,
			},
			Id: id,
		})
	} else {
		_, err = client.DeleteDataSourceByName(cmd.Context(), &minderv1.DeleteDataSourceByNameRequest{
			Context: &minderv1.ContextV2{
				ProjectId: project,
			},
			Name: name,
		})
	}

	if err != nil {
		return cli.MessageAndError("Failed to delete data source", err)
	}

	return outputDeleteResult(cmd, format, id, name)
}

func outputDeleteResult(cmd *cobra.Command, format, id, name string) error {
	switch format {
	case app.JSON:
		fmt.Fprintln(cmd.OutOrStdout(), `{"status": "success"}`)
	case app.YAML:
		fmt.Fprintln(cmd.OutOrStdout(), "status: success")
	default:
		if id != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully deleted data source with ID: %s\n", id)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully deleted data source with Name: %s\n", name)
		}
	}
	return nil
}

func init() {
	DataSourceCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	deleteCmd.Flags().StringP("id", "i", "", "ID of the data source to delete")
	deleteCmd.Flags().StringP("name", "n", "", "Name of the data source to delete")

	// Ensure at least one of id or name is required
	deleteCmd.MarkFlagsOneRequired("id", "name")
}
