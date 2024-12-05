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

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List data sources",
	Long:  `The datasource list subcommand lets you list all data sources within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(listCommand),
}

// listCommand is the datasource list subcommand
func listCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewDataSourceServiceClient(conn)

	project := viper.GetString("project")
	format := viper.GetString("output")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	resp, err := client.ListDataSources(ctx, &minderv1.ListDataSourcesRequest{
		Context: &minderv1.ContextV2{
			ProjectId: project,
		},
	})
	if err != nil {
		return cli.MessageAndError("Failed to list data sources", err)
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
	case app.Table:
		t := table.New(table.Simple, layouts.Default, []string{"ID", "Name", "Type"})
		for _, ds := range resp.GetDataSources() {
			t.AddRow(ds.Id, ds.Name, ds.GetDriverType())
		}
		t.Render()
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	return nil
}

func init() {
	DataSourceCmd.AddCommand(listCmd)

	listCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
