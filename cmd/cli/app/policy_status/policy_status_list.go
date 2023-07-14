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

package policy_status

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
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
	Run: func(cmd *cobra.Command, args []string) {
		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewPolicyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		provider := viper.GetString("provider")
		group := viper.GetInt32("group-id")
		policy_id := viper.GetInt32("policy-id")
		repo_id := viper.GetInt32("repo-id")
		format := viper.GetString("output")

		if format != "json" && format != "yaml" && format != "" {
			fmt.Fprintf(os.Stderr, "Error: invalid format: %s\n", format)
		}

		flags_set := 0
		if policy_id != 0 {
			flags_set++
		}
		if repo_id != 0 {
			flags_set++
		}
		if provider != "" {
			flags_set++
		}
		if flags_set > 1 {
			fmt.Fprintf(os.Stderr, "Error: only one of policy-id, repo-id or provider/group-id can be set\n")
			os.Exit(1)
		}

		// if group is set, provider needs to be set
		if group != 0 && provider == "" {
			fmt.Fprintf(os.Stderr, "Error: provider and group-id must be set together\n")
			os.Exit(1)
		}

		// at least one of policy_id, repo-id or group needs to be set
		if policy_id == 0 && repo_id == 0 && group == 0 {
			fmt.Fprintf(os.Stderr, "Error: policy-id or provider/group-id must be set\n")
			os.Exit(1)
		}

		// check if we go via id or provider/group
		var status []*pb.PolicyRepoStatus
		if policy_id != 0 {
			resp, err := client.GetPolicyStatusById(ctx,
				&pb.GetPolicyStatusByIdRequest{PolicyId: policy_id})
			util.ExitNicelyOnError(err, "Error getting policy status")
			status = resp.PolicyRepoStatus
		} else if repo_id != 0 {
			resp, err := client.GetPolicyStatusByRepository(ctx, &pb.GetPolicyStatusByRepositoryRequest{RepositoryId: repo_id})
			util.ExitNicelyOnError(err, "Error getting policies")
			status = resp.PolicyRepoStatus
		} else {
			resp, err := client.GetPolicyStatusByGroup(ctx,
				&pb.GetPolicyStatusByGroupRequest{Provider: provider, GroupId: group})
			util.ExitNicelyOnError(err, "Error getting policies")
			status = resp.PolicyRepoStatus
		}

		// print output (only json or yaml due to the nature of the data)
		if format == "" {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Policy type", "Repo ID", "Repo owner", "Repo name", "Policy status", "Last updated"})

			for _, v := range status {
				row := []string{
					v.PolicyType,
					fmt.Sprintf("%d", v.GetRepoId()),
					v.RepoOwner,
					v.RepoName,
					v.PolicyStatus,
					v.LastUpdated.AsTime().Format(time.RFC3339),
				}
				table.Append(row)
			}
			table.Render()

		} else if format == "json" {
			output, err := json.MarshalIndent(status, "", "  ")
			util.ExitNicelyOnError(err, "Error marshalling json")
			fmt.Println(string(output))
		} else if format == "yaml" {
			yamlData, err := yaml.Marshal(status)
			util.ExitNicelyOnError(err, "Error marshalling yaml")
			fmt.Println(string(yamlData))

		}
	},
}

func init() {
	PolicyStatusCmd.AddCommand(policystatus_listCmd)
	policystatus_listCmd.Flags().StringP("provider", "p", "", "Provider to list policy violations for")
	policystatus_listCmd.Flags().Int32P("group-id", "g", 0, "group id to list policy violations for")
	policystatus_listCmd.Flags().Int32P("policy-id", "i", 0, "policy id to list policy violations for")
	policystatus_listCmd.Flags().Int32P("repo-id", "r", 0, "repo id to list policy violations for")
	policystatus_listCmd.Flags().StringP("output", "o", "", "Output format (json or yaml)")
}
