// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

type ruleTypeGetter interface {
	protoreflect.ProtoMessage
	GetRuleType() *minderv1.RuleType
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details for a rule type",
	Long:  `The ruletype get subcommand lets you retrieve details for a rule type within Minder.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}

		id, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")
		format, _ := cmd.Flags().GetString("output")

		if id != "" && name != "" {
			return fmt.Errorf("please provide either the --id or --name flag, but not both")
		}

		// Ensure the output format is supported
		if !app.IsOutputFormatSupported(format) {
			return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
		}

		return nil
	},
	RunE: getCommand,
}

func getCommand(cmd *cobra.Command, _ []string) error {
	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, closeConn, err := GetRuleTypeClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer closeConn()

	project := viper.GetString("project")
	format := viper.GetString("output")
	id := viper.GetString("id")
	name := viper.GetString("name")

	var rtype ruleTypeGetter

	if id != "" {
		rtype, err = client.GetRuleTypeById(cmd.Context(), &minderv1.GetRuleTypeByIdRequest{
			Context: &minderv1.Context{Project: &project},
			Id:      id,
		})
	} else {
		rtype, err = client.GetRuleTypeByName(cmd.Context(), &minderv1.GetRuleTypeByNameRequest{
			Context: &minderv1.Context{Project: &project},
			Name:    name,
		})
	}
	if err != nil {
		return cli.MessageAndError("Error getting rule type", err)
	}

	// handle output formatting
	switch format {
	case app.YAML:
		out, err := util.GetYamlFromProto(rtype.GetRuleType())
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	case app.JSON:
		out, err := util.GetJsonFromProto(rtype)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.Table:
		// initialize and render the table
		table := initializeTableForOne(cmd.OutOrStdout())
		rt := rtype.GetRuleType()
		oneRuleTypeToRows(table, rt)
		// add the rule type to the table rows
		table.Render()
	}
	return nil

}

func init() {
	ruleTypeCmd.AddCommand(getCmd)
	// Flags
	getCmd.Flags().StringP("id", "i", "", "ID for the rule type to query")
	getCmd.Flags().StringP("name", "n", "", "Name for the rule type to query")
	getCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))

	getCmd.MarkFlagsMutuallyExclusive("id", "name")
	getCmd.MarkFlagsOneRequired("id", "name")
}
