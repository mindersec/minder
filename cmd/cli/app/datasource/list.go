// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasource

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/internal/util/cli/table"
	"github.com/mindersec/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List data sources",
	Long:  `The datasource list subcommand lets you list all data sources within Minder.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}

		format := viper.GetString("output")

		return app.IsOutputFormatSupported(format)
	},
	RunE: listCommand,
}

// listCommand is the datasource list subcommand
func listCommand(cmd *cobra.Command, _ []string) error {
	client, cleanup, err := cli.GetCLIClient(cmd, minderv1.NewDataSourceServiceClient)
	if err != nil {
		return err
	}
	defer cleanup()

	project := viper.GetString("project")
	format := viper.GetString("output")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	resp, err := client.ListDataSources(cmd.Context(), &minderv1.ListDataSourcesRequest{
		Context: &minderv1.ContextV2{
			ProjectId: project,
		},
	})
	if err != nil {
		return cli.MessageAndError("Failed to list data sources", err)
	}

	switch format {
	case app.JSON, app.YAML:
		return cli.PrintProto(cmd, format, resp)
	case app.Table:
		t := table.New(
			table.Simple, layouts.Default, cmd.OutOrStdout(),
			[]string{"Name", "Type", "Functions"},
		).SetAutoMerge(true)
		for _, ds := range resp.GetDataSources() {
			var functions iter.Seq[string]
			switch driver := ds.Driver.(type) {
			case *minderv1.DataSource_Rest:
				// TODO: do we want to also include arguments?
				functions = maps.Keys(driver.Rest.GetDef())
			case *minderv1.DataSource_Structured:
				functions = maps.Keys(driver.Structured.GetDef())
			}
			t.AddRow(ds.Name, ds.GetDriverType(), strings.Join(slices.Sorted(functions), "\n"))
		}
		t.Render()
	}

	return nil
}

func init() {
	DataSourceCmd.AddCommand(listCmd)

	listCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
