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

package policy

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/cmd/cli/app"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

var policy_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details for a policy within a mediator control plane",
	Long: `The medic policy get subcommand lets you retrieve details for a policy within a
mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := viper.GetString("provider")
		format := viper.GetString("output")

		if format != app.JSON && format != app.YAML && format != app.Table {
			return fmt.Errorf("error: invalid format: %s", format)
		}

		conn, err := util.GrpcForCommand(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewPolicyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		id := viper.GetInt32("id")
		policy, err := client.GetPolicyById(ctx, &pb.GetPolicyByIdRequest{
			Context: &pb.Context{
				Provider: provider,
				// TODO set up group if specified
				// Currently it's inferred from the authorization token
			},
			Id: id,
		})
		util.ExitNicelyOnError(err, "Error getting policy")

		switch format {
		case app.YAML:
			out, err := util.GetYamlFromProto(policy)
			util.ExitNicelyOnError(err, "Error getting yaml from proto")
			fmt.Println(out)
		case app.JSON:
			out, err := util.GetJsonFromProto(policy)
			util.ExitNicelyOnError(err, "Error getting json from proto")
			fmt.Println(out)
		case app.Table:
			p := policy.GetPolicy()
			handleGetTableOutput(cmd, p)
		}

		return nil
	},
}

func init() {
	PolicyCmd.AddCommand(policy_getCmd)
	policy_getCmd.Flags().Int32P("id", "i", 0, "ID for the policy to query")
	policy_getCmd.Flags().StringP("output", "o", app.Table, "Output format (json, yaml or table)")
	policy_getCmd.Flags().StringP("provider", "p", "github", "Provider for the policy")
	// TODO set up group if specified

	if err := policy_getCmd.MarkFlagRequired("id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

}

func handleGetTableOutput(cmd *cobra.Command, policy *pb.PipelinePolicy) {
	table := initializeTable(cmd)

	renderPolicyTable(policy, table)

	table.Render()
}
