// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
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
	Short: "List the providers available in a specific project",
	Long:  `The minder provider list command lists the providers available in a specific project.`,
	RunE:  listProviderCommand,
}

func init() {
	ProviderCmd.AddCommand(listCmd)

	listCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))

	// TODO: implement pagination in CLI
}

// listProviderCommand lists the providers available in a specific project
func listProviderCommand(cmd *cobra.Command, _ []string) error {
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return fmt.Errorf("error binding flags: %w", err)
	}

	cmd.SilenceUsage = true

	client, cleanup, err := cli.GetCLIClient(cmd, minderv1.NewProvidersServiceClient)
	if err != nil {
		return err
	}
	defer cleanup()

	project := viper.GetString("project")
	format := viper.GetString("output")

	cursor := ""
	out := &minderv1.ListProvidersResponse{}

	for {
		resp, err := client.ListProviders(cmd.Context(), &minderv1.ListProvidersRequest{
			Context: &minderv1.Context{
				Project: &project,
			},
			Cursor: cursor,
		})
		if err != nil {
			return err
		}

		out.Providers = append(out.Providers, resp.Providers...)

		if resp.Cursor == "" {
			break
		}

		cursor = resp.Cursor
	}

	switch format {
	case app.JSON, app.YAML:
		return cli.PrintProto(cmd, format, out)
	case app.Table:
		t := table.New(table.Simple, layouts.Default, cmd.OutOrStdout(),
			[]string{"Name", "Version", "Implements"}).
			SetAutoMerge(true)

		for _, v := range out.Providers {
			impls := getImplementsAsStrings(v)
			t.AddRow(v.GetName(), v.GetVersion(), strings.Join(impls, ", "))
		}
		t.Render()
		return nil
	default:
		return fmt.Errorf("output format %s not supported", format)
	}
}
