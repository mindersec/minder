//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ruletype

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details for a rule type",
	Long:  `The ruletype get subcommand lets you retrieve details for a rule type within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(getCommand),
}

type ruleTypeGetter interface {
	protoreflect.ProtoMessage
	GetRuleType() *minderv1.RuleType
}

// getCommand is the ruletype get subcommand
func getCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewRuleTypeServiceClient(conn)

	project := viper.GetString("project")
	format := viper.GetString("output")
	id := viper.GetString("id")
	name := viper.GetString("name")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	var rtype ruleTypeGetter
	var err error

	if id != "" {
		rtype, err = client.GetRuleTypeById(ctx, &minderv1.GetRuleTypeByIdRequest{
			Context: &minderv1.Context{Project: &project},
			Id:      id,
		})
	} else {
		rtype, err = client.GetRuleTypeByName(ctx, &minderv1.GetRuleTypeByNameRequest{
			Context: &minderv1.Context{Project: &project},
			Name:    name,
		})
	}
	if err != nil {
		return cli.MessageAndError("Error getting rule type", err)
	}

	switch format {
	case app.YAML:
		out, err := util.GetYamlFromProto(rtype)
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
		// Initialize the table
		table := initializeTableForOne()
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
