// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasource

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a data source",
	Long:  `The datasource delete subcommand lets you delete a data source within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(deleteCommand),
}

// deleteCommand is the datasource delete subcommand
func deleteCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewDataSourceServiceClient(conn)

	project := viper.GetString("project")
	format := viper.GetString("output")
	id := viper.GetString("id")
	name := viper.GetString("name")

	// No longer print usage on returned error, since we've parsed our inputs
	cmd.SilenceUsage = true

	if id == "" && name == "" {
		return fmt.Errorf("either id or name must be specified")
	}

	var err error

	if id != "" {
		_, err = client.DeleteDataSourceById(ctx, &minderv1.DeleteDataSourceByIdRequest{
			Context: &minderv1.ContextV2{
				ProjectId: project,
			},
			Id: id,
		})
	} else {
		_, err = client.DeleteDataSourceByName(ctx, &minderv1.DeleteDataSourceByNameRequest{
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
		cmd.Println(`{"status": "success"}`)
	case app.YAML:
		cmd.Println("status: success")
	default:
		if id != "" {
			cmd.Printf("Successfully deleted data source with ID: %s\n", id)
		} else {
			cmd.Printf("Successfully deleted data source with Name: %s\n", name)
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
