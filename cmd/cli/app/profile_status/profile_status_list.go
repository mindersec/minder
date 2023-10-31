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

package profile_status

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/cmd/cli/app"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

var profilestatus_listCmd = &cobra.Command{
	Use:   "list",
	Short: "List profile status within a minder control plane",
	Long: `The minder profile_status list subcommand lets you list profile status within a
minder control plane for an specific provider/project or profile id.`,
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

		client := pb.NewProfileServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		provider := viper.GetString("provider")
		project := viper.GetString("project")
		profileName := viper.GetString("profile")
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

		req := &pb.GetProfileStatusByNameRequest{
			Context: &pb.Context{
				Provider: provider,
			},
			Name: profileName,
			All:  all,
			Rule: rule,
		}

		if project != "" {
			req.Context.Project = &project
		}

		resp, err := client.GetProfileStatusByName(ctx, req)
		if err != nil {
			return fmt.Errorf("error getting profile status: %w", err)
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
			handleProfileStatusListTable(cmd, resp)

			if all {
				handleRuleEvaluationStatusListTable(cmd, resp)
			}
		}

		return nil
	},
}

func init() {
	ProfileStatusCmd.AddCommand(profilestatus_listCmd)
	profilestatus_listCmd.Flags().StringP("provider", "p", "github", "Provider to list profile status for")
	profilestatus_listCmd.Flags().StringP("project", "g", "", "Project ID to list profile status for")
	profilestatus_listCmd.Flags().StringP("profile", "i", "", "Profile name to list profile status for")
	profilestatus_listCmd.Flags().StringP("output", "o", app.Table, "Output format (json, yaml or table)")
	profilestatus_listCmd.Flags().BoolP("detailed", "d", false, "List all profile violations")
	profilestatus_listCmd.Flags().StringP("rule", "r", "", "Filter profile status list by rule")

	if err := profilestatus_listCmd.MarkFlagRequired("profile"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
}

func handleProfileStatusListTable(cmd *cobra.Command, resp *pb.GetProfileStatusByNameResponse) {
	table := initializeProfileStatusTable(cmd)

	renderProfileStatusTable(resp.ProfileStatus, table)

	table.Render()
}

func handleRuleEvaluationStatusListTable(cmd *cobra.Command, resp *pb.GetProfileStatusByNameResponse) {
	table := initializeRuleEvaluationStatusTable(cmd)

	for idx := range resp.RuleEvaluationStatus {
		reval := resp.RuleEvaluationStatus[idx]
		renderRuleEvaluationStatusTable(reval, table)
	}

	table.Render()
}
