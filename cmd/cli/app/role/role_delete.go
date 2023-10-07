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

package role

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

var role_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a role within a mediator control plane",
	Long: `The medic role delete subcommand lets you delete roles within a
mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// delete the role via GRPC
		id := util.GetConfigValue("role-id", "role-id", cmd, int32(0)).(int32)
		force := util.GetConfigValue("force", "force", cmd, false).(bool)

		conn, err := util.GrpcForCommand(cmd)

		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewRoleServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		forcePtr := &force
		_, err = client.DeleteRole(ctx, &pb.DeleteRoleRequest{
			Id:    id,
			Force: forcePtr,
		})

		util.ExitNicelyOnError(err, "Error deleting role")
		cmd.Println("Successfully deleted role with id:", id)
	},
}

func init() {
	RoleCmd.AddCommand(role_deleteCmd)
	role_deleteCmd.Flags().Int32P("role-id", "r", 0, "ID of role to delete")
	role_deleteCmd.Flags().BoolP("force", "f", false,
		"Force deletion of role, even if it's protected or has associated users "+
			"(WARNING: removing a protected role may cause loosing mediator access)")
	err := role_deleteCmd.MarkFlagRequired("role-id")
	util.ExitNicelyOnError(err, "Error marking flag as required")
}
