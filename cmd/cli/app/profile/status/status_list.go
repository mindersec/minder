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

package status

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/cmd/cli/app/profile"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List profile status",
	Long:  `The profile status list subcommand lets you list profile status within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(listCommand),
}

// listCommand is the profile "list" subcommand
func listCommand(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewProfileServiceClient(conn)

	project := viper.GetString("project")
	profileName := viper.GetString("name")
	format := viper.GetString("output")
	detailed := viper.GetBool("detailed")
	ruleType := viper.GetString("ruleType")
	ruleName := viper.GetString("ruleName")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	resp, err := client.GetProfileStatusByName(ctx, &minderv1.GetProfileStatusByNameRequest{
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
		table := profile.NewProfileStatusTable()
		profile.RenderProfileStatusTable(resp.ProfileStatus, table)
		table.Render()
		if detailed {
			table = profile.NewRuleEvaluationsTable()
			profile.RenderRuleEvaluationStatusTable(resp.RuleEvaluationStatus, table)
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
}
