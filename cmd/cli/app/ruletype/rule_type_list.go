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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var ruleType_listCmd = &cobra.Command{
	Use:   "list",
	Short: "List rule types within a minder control plane",
	Long: `The minder ruletype list subcommand lets you list rule type within a
minder control plane for an specific project.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: cli.GRPCClientWrapRunE(func(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
		format := viper.GetString("output")

		client := minderv1.NewProfileServiceClient(conn)

		provider := viper.GetString("provider")

		switch format {
		case app.JSON:
		case app.YAML:
		case app.Table:
		default:
			fmt.Fprintf(os.Stderr, "Error: invalid format: %s\n", format)
		}

		resp, err := client.ListRuleTypes(ctx, &minderv1.ListRuleTypesRequest{
			Context: &minderv1.Context{
				Provider: &provider,
				// TODO set up project if specified
				// Currently it's inferred from the authorization token
			},
		})
		if err != nil {
			return fmt.Errorf("error getting profiles: %w", err)
		}

		switch format {
		case app.JSON:
			out, err := util.GetJsonFromProto(resp)
			util.ExitNicelyOnError(err, "Error getting json from proto")
			fmt.Println(out)
		case app.YAML:
			out, err := util.GetYamlFromProto(resp)
			util.ExitNicelyOnError(err, "Error getting yaml from proto")
			fmt.Println(out)
		case app.Table:
			handleListTableOutput(cmd, resp)
		}

		// this is unreachable
		return nil
	}),
}

func init() {
	ruleTypeCmd.AddCommand(ruleType_listCmd)
	ruleType_listCmd.Flags().StringP("provider", "p", "", "Provider to list rule types for")
	ruleType_listCmd.Flags().StringP("output", "o", app.Table, "Output format (json, yaml or table)")
	// TODO: Take project ID into account
	// ruleType_listCmd.Flags().Int32P("project-id", "g", 0, "project id to list roles for")

	if err := ruleType_listCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
}

func handleListTableOutput(cmd *cobra.Command, resp *minderv1.ListRuleTypesResponse) {
	table := initializeTable(cmd)

	for _, v := range resp.RuleTypes {
		renderRuleTypeTable(v, table)
	}
	table.Render()
}
