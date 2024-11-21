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
	"github.com/mindersec/minder/internal/util"
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

	// No longer print usage on returned error, since we've parsed our inputs
	cmd.SilenceUsage = true

	resp, err := client.DeleteDataSource(ctx, &minderv1.DeleteDataSourceRequest{
		Context: &minderv1.ContextV2{
			ProjectId: project,
		},
		Id: id,
	})
	if err != nil {
		return cli.MessageAndError("Failed to delete data source", err)
	}

	switch format {
	case app.JSON:
		out, err := util.GetJsonFromProto(resp)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(resp)
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	default:
		cmd.Printf("Successfully deleted data source with ID: %s\n", id)
	}

	return nil
}

func init() {
	DataSourceCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	deleteCmd.Flags().StringP("id", "i", "", "ID of the data source to delete")

	if err := deleteCmd.MarkFlagRequired("id"); err != nil {
		panic(err)
	}
}
