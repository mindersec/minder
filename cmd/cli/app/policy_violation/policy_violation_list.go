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

package policy_violation

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var policyviolation_listCmd = &cobra.Command{
	Use:   "list",
	Short: "List policy violations within a mediator control plane",
	Long: `The medic policy_violation list subcommand lets you list policy violatins within a
mediator control plane for an specific provider/group or id.`,
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
		limit := viper.GetInt32("limit")
		offset := viper.GetInt32("offset")
		format := viper.GetString("output")

		if format != "json" && format != "yaml" {
			fmt.Fprintf(os.Stderr, "Error: invalid format: %s\n", format)
		}

		// if policy_id is set, provider and group cannot be set
		if policy_id != 0 && (provider != "" || group != 0) {
			fmt.Fprintf(os.Stderr, "Error: policy-id cannot be set with provider or group-id\n")
			os.Exit(1)
		}
		// if provider is set, group needs to be set
		if (provider != "" && group == 0) || (provider == "" && group != 0) {
			fmt.Fprintf(os.Stderr, "Error: provider and group-id must be set together\n")
			os.Exit(1)
		}
		// at least one of policy_id or provider/group needs to be set
		if policy_id == 0 && provider == "" && group == 0 {
			fmt.Fprintf(os.Stderr, "Error: policy-id or provider/group-id must be set\n")
			os.Exit(1)
		}

		var limitPtr = &limit
		var offsetPtr = &offset

		// check if we go via id or provider/group
		var violations []*pb.PolicyViolation
		if policy_id != 0 {
			resp, err := client.GetPolicyViolationsById(ctx,
				&pb.GetPolicyViolationsByIdRequest{Id: policy_id, Limit: limitPtr, Offset: offsetPtr})
			util.ExitNicelyOnError(err, "Error getting policies")
			violations = resp.PolicyViolation
		} else {
			resp, err := client.GetPolicyViolations(ctx,
				&pb.GetPolicyViolationsRequest{Provider: provider, GroupId: group, Limit: limitPtr, Offset: offsetPtr})
			util.ExitNicelyOnError(err, "Error getting policies")
			violations = resp.PolicyViolation
		}

		// print output (only json or yaml due to the nature of the data)
		if format == "json" {
			output, err := json.MarshalIndent(violations, "", "  ")
			util.ExitNicelyOnError(err, "Error marshalling json")
			fmt.Println(string(output))
		} else if format == "yaml" {
			yamlData, err := yaml.Marshal(violations)
			util.ExitNicelyOnError(err, "Error marshalling yaml")
			fmt.Println(string(yamlData))

		}
	},
}

func init() {
	PolicyViolationCmd.AddCommand(policyviolation_listCmd)
	policyviolation_listCmd.Flags().StringP("provider", "p", "", "Provider to list policy violations for")
	policyviolation_listCmd.Flags().Int32P("group-id", "g", 0, "group id to list policy violations for")
	policyviolation_listCmd.Flags().Int32P("policy-id", "i", 0, "policy id to list policy violations for")
	policyviolation_listCmd.Flags().StringP("output", "o", "json", "Output format (json or yaml)")
	policyviolation_listCmd.Flags().Int32P("limit", "l", -1, "Limit the number of results returned")
	policyviolation_listCmd.Flags().Int32P("offset", "f", 0, "Offset the results returned")
}
