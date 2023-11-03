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

package rule_type

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/util"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var ruleType_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details for a rule type within a minder control plane",
	Long: `The minder rule_type get subcommand lets you retrieve details for a rule type within a
minder control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := viper.GetString("provider")
		format := viper.GetString("output")

		switch format {
		case app.JSON:
		case app.YAML:
		case app.Table:
		default:
			return fmt.Errorf("error: invalid format: %s", format)
		}

		conn, err := util.GrpcForCommand(cmd, viper.GetViper())
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := minderv1.NewProfileServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		id := viper.GetString("id")

		rtype, err := client.GetRuleTypeById(ctx, &minderv1.GetRuleTypeByIdRequest{
			Context: &minderv1.Context{
				Provider: provider,
				// TODO set up project if specified
				// Currently it's inferred from the authorization token
			},
			Id: id,
		})
		if err != nil {
			return fmt.Errorf("error getting rule type: %w", err)
		}

		switch format {
		case app.YAML:
			out, err := util.GetYamlFromProto(rtype)
			util.ExitNicelyOnError(err, "Error getting json from proto")
			fmt.Println(out)
		case app.JSON:
			out, err := util.GetJsonFromProto(rtype)
			util.ExitNicelyOnError(err, "Error getting json from proto")
			fmt.Println(out)
		case app.Table:
			handleGetTableOutput(cmd, rtype.GetRuleType())
		}
		return nil
	},
}

func init() {
	ruleTypeCmd.AddCommand(ruleType_getCmd)
	ruleType_getCmd.Flags().StringP("id", "i", "", "ID for the profile to query")
	ruleType_getCmd.Flags().StringP("output", "o", app.Table, "Output format (json, yaml or table)")
	ruleType_getCmd.Flags().StringP("provider", "p", "github", "Provider for the profile")
	// TODO set up project if specified

	if err := ruleType_getCmd.MarkFlagRequired("id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

}

func handleGetTableOutput(cmd *cobra.Command, rtype *minderv1.RuleType) {
	table := initializeTable(cmd)

	renderRuleTypeTable(rtype, table)

	table.Render()
}
