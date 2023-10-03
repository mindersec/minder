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

package policy_status

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/cmd/cli/app"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

var policystatus_listCmd = &cobra.Command{
	Use:   "list",
	Short: "List policy status within a mediator control plane",
	Long: `The medic policy_status list subcommand lets you list policy status within a
mediator control plane for an specific provider/group or policy id.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := util.GrpcForCommand(cmd)
		if err != nil {
			return fmt.Errorf("error getting grpc connection: %w", err)
		}
		defer conn.Close()

		client := pb.NewPolicyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		provider := viper.GetString("provider")
		group := viper.GetString("group")
		policyId := viper.GetString("policy")
		format := viper.GetString("output")
		all := viper.GetBool("detailed")
		rule := viper.GetString("rule")

		switch format {
		case app.JSON, app.YAML, app.Table:
		default:
			return fmt.Errorf("error: invalid format: %s", format)
		}

		if provider == "" {
			return fmt.Errorf("provider must be set")
		}

		req := &pb.GetPolicyStatusByIdRequest{
			Context: &pb.Context{
				Provider: provider,
			},
			PolicyId: policyId,
			All:      all,
			Rule:     rule,
		}

		if group != "" {
			req.Context.Group = &group
		}

		resp, err := client.GetPolicyStatusById(ctx, req)
		if err != nil {
			return fmt.Errorf("error getting policy status: %w", err)
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
			handlePolicyStatusListTable(cmd, resp)

			if all {
				handleRuleEvaluationStatusListTable(cmd, resp)
			}
		}

		return nil
	},
}

func init() {
	PolicyStatusCmd.AddCommand(policystatus_listCmd)
	policystatus_listCmd.Flags().StringP("provider", "p", "github", "Provider to list policy status for")
	policystatus_listCmd.Flags().StringP("group", "g", "", "group id to list policy status for")
	policystatus_listCmd.Flags().StringP("policy", "i", "", "policy id to list policy status for")
	policystatus_listCmd.Flags().StringP("output", "o", app.Table, "Output format (json, yaml or table)")
	policystatus_listCmd.Flags().BoolP("detailed", "d", false, "List all policy violations")
	policystatus_listCmd.Flags().StringP("rule", "r", "", "Filter policy status list by rule")

	if err := policystatus_listCmd.MarkFlagRequired("policy"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
}

func handlePolicyStatusListTable(cmd *cobra.Command, resp *pb.GetPolicyStatusByIdResponse) {
	table := initializePolicyStatusTable(cmd)

	renderPolicyStatusTable(resp.PolicyStatus, table)

	table.Render()
}

func handleRuleEvaluationStatusListTable(cmd *cobra.Command, resp *pb.GetPolicyStatusByIdResponse) {
	table := initializeRuleEvaluationStatusTable(cmd)

	for idx := range resp.RuleEvaluationStatus {
		reval := resp.RuleEvaluationStatus[idx]
		renderRuleEvaluationStatusTable(reval, table)
	}

	table.Render()
}
