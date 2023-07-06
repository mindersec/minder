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

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewPolicyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		id := viper.GetInt32("id")
		policy, err := client.GetPolicyById(ctx, &pb.GetPolicyByIdRequest{Id: id})
		util.ExitNicelyOnError(err, "Error getting policy")

		json, err := json.MarshalIndent(policy.Policy, "", "  ")
		util.ExitNicelyOnError(err, "Error marshalling policy")
		fmt.Println(string(json))
	},
}

func init() {
	PolicyCmd.AddCommand(policy_getCmd)
	policy_getCmd.Flags().Int32P("id", "i", 0, "ID for the policy to query")
	if err := policy_getCmd.MarkFlagRequired("id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

}
