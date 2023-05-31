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

package user

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
)

var user_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a user within a mediator controlplane",
	Long: `The medctl user delete subcommand lets you delete users within a
mediator control plane.`,
	Run: func(cmd *cobra.Command, args []string) {
		// delete the user via GRPC
		id := util.GetConfigValue("user-id", "user-id", cmd, int32(0)).(int32)
		force := util.GetConfigValue("force", "force", cmd, false).(bool)

		conn, err := util.GetGrpcConnection(cmd)
		defer conn.Close()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting grpc connection: %s\n", err)
			os.Exit(1)
		}

		client := pb.NewUserServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		forcePtr := &force
		_, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{
			Id:    id,
			Force: forcePtr,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting user: %s\n", err)
			os.Exit(1)
		}
		cmd.Println("Successfully deleted user with id:", id)
	},
}

func init() {
	UserCmd.AddCommand(user_deleteCmd)
	user_deleteCmd.PersistentFlags().Int32P("user-id", "u", 0, "id of user to delete")
	user_deleteCmd.PersistentFlags().BoolP("force", "f", false,
		"Force deletion of user, even if it's protected (WARNING: removing a protected user may cause loss of mediator access and data)")
	if err := user_deleteCmd.MarkPersistentFlagRequired("user-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

	if err := viper.BindPFlags(user_deleteCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
