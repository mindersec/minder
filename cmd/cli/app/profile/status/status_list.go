// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/cmd/cli/app/profile"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List profile status",
	Long:  `The profile status list subcommand lets you list profile status within Minder.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %s", err)
		}
		return nil
	},
	RunE: listCommand,
}

func listCommand(cmd *cobra.Command, _ []string) error {
	project := viper.GetString("project")
	profileName := viper.GetString("name")
	detailed := viper.GetBool("detailed")
	ruleType := viper.GetString("ruleType")
	ruleName := viper.GetString("ruleName")

	cmd.SilenceUsage = true

	format := viper.GetString("output")

	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	client, closer, err := profile.GetProfileClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer closer()

	resp, err := client.GetProfileStatusByName(cmd.Context(), &minderv1.GetProfileStatusByNameRequest{
		Context:  &minderv1.Context{Project: &project},
		Name:     profileName,
		All:      detailed,
		RuleType: ruleType,
		RuleName: ruleName,
	})
	if err != nil {
		return cli.MessageAndError("Error getting profile status", err)
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
		table := profile.NewProfileStatusTable(cmd.OutOrStdout())
		profile.RenderProfileStatusTable(resp.ProfileStatus, table, viper.GetBool("emoji"))
		table.Render()

		if detailed {
			cmd.Println()
			table = profile.NewRuleEvaluationsTable(cmd.OutOrStdout())
			table.SeparateRows()
			profile.RenderRuleEvaluationStatusTable(resp.RuleEvaluationStatus, table, viper.GetBool("emoji"))
			table.Render()
		}
	}
	return nil
}

func init() {
	profileStatusCmd.AddCommand(listCmd)
	// Flags
	listCmd.Flags().BoolP("detailed", "d", false, "List all profile violations")
	listCmd.Flags().StringP("ruleType", "r", "", "Filter profile status list by rule type")
	listCmd.Flags().String("ruleName", "", "Filter profile status list by rule name")

	listCmd.Flags().StringP("name", "n", "", "Profile name to list status for")
	listCmd.Flags().Bool("emoji", true, "Use emojis in the output")

	if err := listCmd.MarkFlagRequired("name"); err != nil {
		listCmd.Printf("Error marking flag required: %s", err)
		os.Exit(1)
	}
}
