// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List rule types",
	Long:  `The ruletype list subcommand lets you list rule type within Minder.`,
	Args:  cobra.NoArgs,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}

		format, err := cmd.Flags().GetString("output")
		if err != nil {
			return cli.MessageAndError("Error parsing output flag", err)
		}

		// Ensure the output format is supported
		if !app.IsOutputFormatSupported(format) {
			return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
		}

		return nil
	},
	RunE: listCommand,
}

func listCommand(cmd *cobra.Command, _ []string) error {
	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, closeConn, err := getRuleTypeClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer closeConn()

	project := viper.GetString("project")

	format := viper.GetString("output")

	resp, err := client.ListRuleTypes(cmd.Context(), &minderv1.ListRuleTypesRequest{
		Context: &minderv1.Context{Project: &project},
	})
	if err != nil {
		return cli.MessageAndError("Error listing rule types", err)
	}

	// handle output formatting
	switch format {
	case app.JSON:
		out, err := util.GetJsonFromProto(resp)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		for _, rt := range resp.GetRuleTypes() {
			out, err := util.GetYamlFromProto(rt)
			if err != nil {
				return cli.MessageAndError("Error getting yaml from proto", err)
			}
			// Print YAML separator between each rule type
			cmd.Println("---")
			cmd.Println(out)
		}
	case app.Table:
		// Sort by Entity Type first to ensure AutoMerge works correctly,
		// then by Name within those groups.
		slices.SortFunc(resp.RuleTypes, func(a, b *minderv1.RuleType) int {
			if a.GetDef().GetInEntity() != b.GetDef().GetInEntity() {
				return strings.Compare(a.GetDef().GetInEntity(), b.GetDef().GetInEntity())
			}
			return strings.Compare(a.GetName(), b.GetName())
		})

		table := initializeTableForList(cmd.OutOrStdout())

		for _, rt := range resp.RuleTypes {
			table.AddRow(
				appendRuleTypePropertiesToName(rt),
				rt.GetDef().GetInEntity(),
				rt.Description,
			)
		}
		table.Render()
	}

	return nil
}

func init() {
	ruleTypeCmd.AddCommand(listCmd)
	// Flags
	listCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
