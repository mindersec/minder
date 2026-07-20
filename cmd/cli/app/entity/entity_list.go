// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/internal/util/cli/table"
	"github.com/mindersec/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List entities",
	Long:  `The entity list subcommand is used to list entity instances within Minder.`,
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
	RunE: listCommand,
}

// listCommand is the entity list subcommand
func listCommand(cmd *cobra.Command, _ []string) error {
	project := viper.GetString("project")
	provider := viper.GetString("provider")
	format := viper.GetString("output")
	entityTypeStr := viper.GetString("type")
	properties := viper.GetStringSlice("property")

	entityType := minderv1.EntityFromString(entityTypeStr)
	if entityType == minderv1.Entity_ENTITY_UNSPECIFIED {
		return fmt.Errorf("invalid or unspecified entity type %q", entityTypeStr)
	}

	client, closeConn, err := cli.GetCLIClient(cmd, minderv1.NewEntityInstanceServiceClient)
	if err != nil {
		return cli.MessageAndError("Error creating gRPC client", err)
	}
	defer closeConn()

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	resp, err := client.ListEntities(cmd.Context(), &minderv1.ListEntitiesRequest{
		Context: &minderv1.ContextV2{
			ProjectId: project,
			Provider:  provider,
		},
		EntityType: entityType,
	})
	if err != nil {
		return cli.MessageAndError("Error listing entities", err)
	}

	switch format {
	case app.Table:
		emoji := viper.GetBool("emoji")
		header := []string{"Type", "Name", "Provider"}
		if len(properties) == 0 {
			header = append(header, "ID")
		}
		header = append(header, properties...)
		t := table.New(table.Simple, layouts.Default, cmd.OutOrStdout(), header)

		for _, e := range resp.GetResults() {
			typeIcon := table.GetEntityTypeIcon(e.GetType().String(), emoji)
			row := []layouts.ColoredColumn{
				typeIcon,
				layouts.NoColor(e.GetName()),
				layouts.NoColor(e.GetContext().GetProvider()),
			}

			if len(properties) == 0 {
				row = append(row, layouts.NoColor(e.GetId()))
			}

			for _, p := range properties {
				val := e.GetProperties().GetFields()[p]
				if val == nil {
					row = append(row, layouts.NoColor(""))
				} else {
					row = append(row, layouts.NoColor(fmt.Sprintf("%v", val.AsInterface())))
				}
			}

			t.AddRowWithColor(row...)
		}
		t.Render()
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
	}

	return nil
}

func init() {
	EntityCmd.AddCommand(listCmd)
	// Flags
	listCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	listCmd.Flags().StringP("type", "t", "", "Type of entity to list (e.g. repository, artifact, pull_request)")
	listCmd.Flags().Bool("emoji", true, "Use emojis in the output")
	listCmd.Flags().StringSlice("property", []string{}, "Properties to include in the output table")
	// Required
	if err := listCmd.MarkFlagRequired("type"); err != nil {
		panic(err)
	}
}
