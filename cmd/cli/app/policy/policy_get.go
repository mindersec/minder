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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/encoding/protojson"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)
		provider := viper.GetString("provider")
		format := viper.GetString("output")

		if format != app.JSON && format != app.YAML && format != "" {
			return fmt.Errorf("error: invalid format: %s", format)
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
			resp, err := client.GetPolicyStatusById(ctx, &pb.GetPolicyStatusByIdRequest{
				Context: &pb.Context{
					Provider: provider,
					// TODO set up group if specified
					// Currently it's inferred from the authorization token
				},
				PolicyId: id,
			})
			util.ExitNicelyOnError(err, "Error getting policy status")

			// print results
			m := protojson.MarshalOptions{
				Indent: "  ",
			}

			if format == "" {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"Policy ID", "Policy Name", "Status", "Last updated"})

				st := resp.GetPolicyStatus()
				row := []string{
					fmt.Sprintf("%d", st.PolicyId),
					st.PolicyName,
					st.PolicyStatus,
					st.GetLastUpdated().AsTime().Format(time.RFC3339),
				}
				table.Append(row)
				table.Render()
			} else if format == app.JSON {
				output, err := m.Marshal(resp)
				util.ExitNicelyOnError(err, "Error marshalling json")
				fmt.Println(string(output))
			} else if format == app.YAML {
				output, err := m.Marshal(resp)
				util.ExitNicelyOnError(err, "Error marshalling json")

				var rawMsg json.RawMessage
				err = json.Unmarshal(output, &rawMsg)
				util.ExitNicelyOnError(err, "Error unmarshalling json")
				yamlResult, err := util.ConvertJsonToYaml(rawMsg)
				util.ExitNicelyOnError(err, "Error converting json to yaml")
				fmt.Println(string(yamlResult))
			}

			return nil
		}

		policy, err := client.GetPolicyById(ctx, &pb.GetPolicyByIdRequest{
			Context: &pb.Context{
				Provider: provider,
				// TODO set up group if specified
				// Currently it's inferred from the authorization token
			},
			Id: id,
		})
		util.ExitNicelyOnError(err, "Error getting policy")

		if format == app.YAML {
			yamlData, err := yaml.Marshal(policy.Policy)
			if err != nil {
				return fmt.Errorf("error marshalling yaml: %w", err)
			}
			fmt.Println(string(yamlData))
		}

		json, err := json.MarshalIndent(policy.Policy, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshalling json: %w", err)
		}
		fmt.Println(string(json))

		return nil
	},
}

func init() {
	PolicyCmd.AddCommand(policy_getCmd)
	policy_getCmd.Flags().Int32P("id", "i", 0, "ID for the policy to query")
	policy_getCmd.Flags().BoolP("status", "s", false, "Only return the status of the policy for all the associated repos")
	policy_getCmd.Flags().StringP("output", "o", "", "Output format (json or yaml)")
	policy_getCmd.Flags().StringP("provider", "p", "github", "Provider for the policy")
	// TODO set up group if specified

	if err := policy_getCmd.MarkFlagRequired("id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

}
