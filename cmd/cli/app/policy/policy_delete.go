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

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
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
		id := viper.GetString("id")
		provider := viper.GetString("provider")

		conn, err := util.GrpcForCommand(cmd)

		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewPolicyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		_, err = client.DeletePolicy(ctx, &pb.DeletePolicyRequest{
			Context: &pb.Context{
				Provider: provider,
			},
			Id: id,
		})

		util.ExitNicelyOnError(err, "Error deleting policy")
		cmd.Println("Successfully deleted policy with id:", id)
	},
}

func init() {
	PolicyCmd.AddCommand(policy_deleteCmd)
	policy_deleteCmd.Flags().StringP("id", "i", "", "id of policy to delete")
	policy_deleteCmd.Flags().StringP("provider", "p", "github", "Provider for the policy")
	err := policy_deleteCmd.MarkFlagRequired("id")
	util.ExitNicelyOnError(err, "Error marking flag as required")
	// TODO: add a flag for the policy name
}
