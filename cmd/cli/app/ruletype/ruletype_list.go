// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

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

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List rule types",
	Long:  `The ruletype list subcommand lets you list rule type within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(listCommand),
}

// listCommand is the ruletype list subcommand
func listCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewRuleTypeServiceClient(conn)

	project := viper.GetString("project")
	format := viper.GetString("output")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	resp, err := client.ListRuleTypes(ctx, &minderv1.ListRuleTypesRequest{
		Context: &minderv1.Context{Project: &project},
	})
	if err != nil {
		return cli.MessageAndError("Error listing rule types", err)
	}

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
		table := initializeTableForList()
		table.SeparateRows()
		for _, rt := range resp.RuleTypes {
			table.AddRow(
				appendRuleTypePropertiesToName(rt),
				rt.GetDef().GetInEntity(),
				cli.RenderMarkdown(rt.Description, cli.WidthFraction(0.5)),
			)
		}
		table.Render()
	}
	// this is unreachable
	return nil
}

func init() {
	ruleTypeCmd.AddCommand(listCmd)
	// Flags
	listCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
