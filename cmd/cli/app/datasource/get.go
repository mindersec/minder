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
	"github.com/mindersec/minder/internal/util/cli/table"
	"github.com/mindersec/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get data source details",
	Long:  `The datasource get subcommand lets you retrieve details for a data source within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(getCommand),
}

// getCommand is the datasource get subcommand
func getCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewDataSourceServiceClient(conn)

	project := viper.GetString("project")
	format := viper.GetString("output")
	id := viper.GetString("id")
	name := viper.GetString("name")

	// No longer print usage on returned error, since we've parsed our inputs
	cmd.SilenceUsage = true

	var err error
	var ds *minderv1.DataSource

	if id == "" && name == "" {
		return fmt.Errorf("either id or name must be specified")
	}

	var resp interface {
		GetDataSource() *minderv1.DataSource
	}

	if id != "" {
		resp, err = client.GetDataSourceById(ctx, &minderv1.GetDataSourceByIdRequest{
			Context: &minderv1.ContextV2{
				ProjectId: project,
			},
			Id: id,
		})
	} else {
		resp, err = client.GetDataSourceByName(ctx, &minderv1.GetDataSourceByNameRequest{
			Context: &minderv1.ContextV2{
				ProjectId: project,
			},
			Name: name,
		})
	}

	if err != nil {
		return cli.MessageAndError("Failed to get data source", err)
	}

	ds = resp.GetDataSource()
	return outputDataSource(cmd, format, ds)
}

func outputDataSource(cmd *cobra.Command, format string, ds *minderv1.DataSource) error {
	switch format {
	case app.JSON:
		out, err := util.GetJsonFromProto(ds)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(ds)
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	case app.Table:
		t := table.New(table.Simple, layouts.Default, []string{"ID", "Name", "Type"})
		t.AddRow(ds.Id, ds.Name, getDataSourceType(ds))
		t.Render()
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
	return nil
}

func init() {
	DataSourceCmd.AddCommand(getCmd)

	getCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	getCmd.Flags().StringP("id", "i", "", "ID of the data source to get info from")
	getCmd.Flags().StringP("name", "n", "", "Name of the data source to get info from")

	// Ensure at least one of id or name is required
	getCmd.MarkFlagsOneRequired("id", "name")
}
