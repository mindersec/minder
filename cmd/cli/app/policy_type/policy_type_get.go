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

package policy_type

import (
	"encoding/json"
	"fmt"
	"os"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/internal/util"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var policy_type_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details for a policy type within a mediator control plane",
	Long: `The medic policy_type get subcommand lets you retrieve details for a policy type within a
mediator control plane.`,
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
		policy_type := viper.GetString("type")
		schema := util.GetConfigValue("schema", "schema", cmd, false).(bool)
		default_schema := util.GetConfigValue("default_schema", "default_schema", cmd, false).(bool)

		policy, err := client.GetPolicyType(ctx, &pb.GetPolicyTypeRequest{Provider: provider, Type: policy_type})
		util.ExitNicelyOnError(err, "Error getting policy")

		var data map[string]json.RawMessage
		err = json.Unmarshal([]byte(policy.PolicyType.JsonSchema), &data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting policy type schema")
			os.Exit(1)
		}

		jsonS, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting policy type schema")
			os.Exit(1)
		}

		// format schema to look nice
		if schema {
			fmt.Printf("%+v\n", string(jsonS))
		} else if default_schema {
			fmt.Printf("%+v\n", string(policy.PolicyType.DefaultSchema))
		} else {
			policy.PolicyType.JsonSchema = string(jsonS)

			json, err := json.MarshalIndent(policy.PolicyType, "", "  ")
			util.ExitNicelyOnError(err, "Error marshalling policy")
			fmt.Printf("%+v\n", string(json))
		}
	},
}

func init() {
	PolicyTypeCmd.AddCommand(policy_type_getCmd)
	policy_type_getCmd.Flags().StringP("provider", "p", "", "Provider for the policy type")
	policy_type_getCmd.Flags().StringP("type", "t", "", "Type of the policy")
	policy_type_getCmd.Flags().BoolP("schema", "s", false, "Only get the json schema in a readable format")
	policy_type_getCmd.Flags().BoolP("default_schema", "d", false, "Only get the default schema in a readable format")
	if err := policy_type_getCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
	if err := policy_type_getCmd.MarkFlagRequired("type"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
}
