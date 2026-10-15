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

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get data source details",
	Long:  `The datasource get subcommand lets you retrieve details for a data source within Minder.`,
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
	RunE: getCommand,
}

// getCommand is the datasource get subcommand
func getCommand(cmd *cobra.Command, _ []string) error {
	client, cleanup, err := cli.GetCLIClient(cmd, minderv1.NewDataSourceServiceClient)
	if err != nil {
		return err
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

	var ds *minderv1.DataSource

	if id != "" {
		resp, err := client.GetDataSourceById(cmd.Context(), &minderv1.GetDataSourceByIdRequest{
			Context: &minderv1.ContextV2{
				ProjectId: project,
			},
			Id: id,
		})
		if err != nil {
			return cli.MessageAndError("Failed to get data source", err)
		}
		ds = resp.GetDataSource()
	} else {
		resp, err := client.GetDataSourceByName(cmd.Context(), &minderv1.GetDataSourceByNameRequest{
			Context: &minderv1.ContextV2{
				ProjectId: project,
			},
			Name: name,
		})
		if err != nil {
			return cli.MessageAndError("Failed to get data source", err)
		}
		ds = resp.GetDataSource()
	}

	return outputDataSource(cmd, format, ds)
}

func outputDataSource(cmd *cobra.Command, format string, ds *minderv1.DataSource) error {
	switch format {
	case app.JSON, app.YAML:
		return cli.PrintProto(cmd, format, ds)
	case app.Table:
		t := table.New(
			table.Simple, layouts.Default, cmd.OutOrStdout(),
			[]string{"Name", "Type", "Functions"},
		).SetAutoMerge(true)
		var functions iter.Seq[string]
		switch driver := ds.Driver.(type) {
		case *minderv1.DataSource_Rest:
			// TODO: do we want to also include arguments?
			functions = maps.Keys(driver.Rest.GetDef())
		case *minderv1.DataSource_Structured:
			functions = maps.Keys(driver.Structured.GetDef())
		}
		t.AddRow(ds.Name, ds.GetDriverType(), strings.Join(slices.Sorted(functions), ", "))
		t.Render()
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
