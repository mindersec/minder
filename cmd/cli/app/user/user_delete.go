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

package user

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

var user_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a user within a mediator controlplane",
	Long: `The medic user delete subcommand lets you delete users within a
mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// delete the user via GRPC
		id := util.GetConfigValue("user-id", "user-id", cmd, int32(0)).(int32)
		force := util.GetConfigValue("force", "force", cmd, false).(bool)

		conn, err := util.GrpcForCommand(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")

		defer conn.Close()

		client := pb.NewUserServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		forcePtr := &force
		_, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{
			Id:    id,
			Force: forcePtr,
		})

		util.ExitNicelyOnError(err, "Error deleting user")
		cmd.Println("Successfully deleted user with id:", id)
	},
}

func init() {
	UserCmd.AddCommand(user_deleteCmd)
	user_deleteCmd.Flags().Int32P("user-id", "u", 0, "id of user to delete")
	user_deleteCmd.Flags().BoolP("force", "f", false,
		"Force deletion of user, even if it's protected "+
			"(WARNING: removing a protected user may cause loss of mediator access and data)")
	err := user_deleteCmd.MarkFlagRequired("user-id")
	util.ExitNicelyOnError(err, "Error marking flag as required")
}
