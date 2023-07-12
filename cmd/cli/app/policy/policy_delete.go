//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.role/licenses/LICENSE-2.0
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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/internal/util"
)

var policy_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a policy within a mediator controlplane",
	Long: `The medic policy delete subcommand lets you delete policies within a
mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// delete the policy via GRPC
		id := util.GetConfigValue("id", "id", cmd, int32(0)).(int32)

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)

		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewPolicyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		_, err = client.DeletePolicy(ctx, &pb.DeletePolicyRequest{
			Id: id,
		})

		util.ExitNicelyOnError(err, "Error deleting policy")
		cmd.Println("Successfully deleted policy with id:", id)
	},
}

func init() {
	PolicyCmd.AddCommand(policy_deleteCmd)
	policy_deleteCmd.Flags().Int32P("id", "i", 0, "id of policy to delete")
	err := policy_deleteCmd.MarkFlagRequired("id")
	util.ExitNicelyOnError(err, "Error marking flag as required")
}
