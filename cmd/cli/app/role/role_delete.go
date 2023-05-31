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

package role

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

var role_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a role within a mediator controlplane",
	Long: `The medctl role delete subcommand lets you delete roles within a
mediator control plane.`,
	Run: func(cmd *cobra.Command, args []string) {
		// delete the role via GRPC
		id := util.GetConfigValue("role-id", "role-id", cmd, int32(0)).(int32)
		force := util.GetConfigValue("force", "force", cmd, false).(bool)

		conn, err := util.GetGrpcConnection(cmd)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting grpc connection: %s\n", err)
			os.Exit(1)
		}
		defer conn.Close()

		client := pb.NewRoleServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		forcePtr := &force
		_, err = client.DeleteRole(ctx, &pb.DeleteRoleRequest{
			Id:    id,
			Force: forcePtr,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting role: %s\n", err)
			os.Exit(1)
		}
		cmd.Println("Successfully deleted role with id:", id)
	},
}

func init() {
	RoleCmd.AddCommand(role_deleteCmd)
	role_deleteCmd.PersistentFlags().Int32P("role-id", "r", 0, "id of role to delete")
	role_deleteCmd.PersistentFlags().BoolP("force", "f", false,
		"Force deletion of role, even if it's protected or has associated users "+
			"(WARNING: removing a protected role may cause loosing mediator access)")
	if err := role_deleteCmd.MarkPersistentFlagRequired("role-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

	if err := viper.BindPFlags(role_deleteCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
