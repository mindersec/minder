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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

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
	Run: func(cmd *cobra.Command, args []string) {
		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		format := viper.GetString("output")

		if format != app.JSON && format != app.YAML && format != "" {
			fmt.Fprintf(os.Stderr, "Error: invalid format: %s\n", format)
		}

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewPolicyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		id := viper.GetInt32("id")
		status := util.GetConfigValue("status", "status", cmd, false).(bool)
		if status {
			resp, err := client.GetPolicyStatusById(ctx, &pb.GetPolicyStatusByIdRequest{PolicyId: id})
			util.ExitNicelyOnError(err, "Error getting policy status")

			// print results
			if format == "" {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Policy type", "Repo ID", "Repo owner", "Repo Name", "Status", "Last updated"})

				for _, v := range resp.PolicyRepoStatus {
					row := []string{
						v.PolicyType,
						fmt.Sprintf("%d", v.RepoId),
						v.RepoOwner,
						v.RepoName,
						v.PolicyStatus,
						v.GetLastUpdated().AsTime().Format(time.RFC3339),
					}
					table.Append(row)
				}
				table.Render()
			} else if format == app.JSON {
				output, err := json.MarshalIndent(resp.PolicyRepoStatus, "", "  ")
				util.ExitNicelyOnError(err, "Error marshalling json")
				fmt.Println(string(output))
			} else if format == app.YAML {
				yamlData, err := yaml.Marshal(resp.PolicyRepoStatus)
				util.ExitNicelyOnError(err, "Error marshalling yaml")
				fmt.Println(string(yamlData))
			}

		} else {
			policy, err := client.GetPolicyById(ctx, &pb.GetPolicyByIdRequest{Id: id})
			util.ExitNicelyOnError(err, "Error getting policy")

			if format == app.YAML {
				yamlData, err := yaml.Marshal(policy.Policy)
				util.ExitNicelyOnError(err, "Error marshalling yaml")
				fmt.Println(string(yamlData))
			} else {
				json, err := json.MarshalIndent(policy.Policy, "", "  ")
				util.ExitNicelyOnError(err, "Error marshalling policy")
				fmt.Println(string(json))
			}
		}
	},
}

func init() {
	PolicyCmd.AddCommand(policy_getCmd)
	policy_getCmd.Flags().Int32P("id", "i", 0, "ID for the policy to query")
	policy_getCmd.Flags().BoolP("status", "s", false, "Only return the status of the policy for all the associated repos")
	policy_getCmd.Flags().StringP("output", "o", "", "Output format (json or yaml)")

	if err := policy_getCmd.MarkFlagRequired("id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

}
