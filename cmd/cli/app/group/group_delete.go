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

package group

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

var group_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a group within a mediator controlplane",
	Long: `The medic group delete subcommand lets you delete groups within a
mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// delete the group via GRPC
		id := util.GetConfigValue("group-id", "group-id", cmd, int32(0)).(int32)
		force := util.GetConfigValue("force", "force", cmd, false).(bool)

		conn, err := util.GrpcForCommand(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")

		defer conn.Close()

		client := pb.NewGroupServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		forcePtr := &force
		_, err = client.DeleteGroup(ctx, &pb.DeleteGroupRequest{
			Id:    id,
			Force: forcePtr,
		})
		util.ExitNicelyOnError(err, "Error deleting group")

		cmd.Println("Successfully deleted group with id:", id)
	},
}

func init() {
	GroupCmd.AddCommand(group_deleteCmd)
	group_deleteCmd.Flags().Int32P("group-id", "g", 0, "id of group to delete")
	group_deleteCmd.Flags().BoolP("force", "f", false,
		"Force deletion of group, even if it's protected or has associated roles "+
			"(WARNING: removing a protected group may cause loosing mediator access)")

	err := group_deleteCmd.MarkFlagRequired("group-id")
	util.ExitNicelyOnError(err, "Error marking flag as required")
}
