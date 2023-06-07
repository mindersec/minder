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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
)

// Role_createCmd represents the role create command
var Role_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a role within a mediator control plane",
	Long: `The medctl role create subcommand lets you create new roles for a group
within a mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// create the role via GRPC
		group := util.GetConfigValue("group-id", "group-id", cmd, int32(0)).(int32)
		name := util.GetConfigValue("name", "name", cmd, "")
		isAdmin := util.GetConfigValue("is_admin", "is_admin", cmd, false).(bool)
		isProtected := util.GetConfigValue("is_protected", "is_protected", cmd, false).(bool)

		conn, err := util.GetGrpcConnection(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting grpc connection: %s\n", err)
			os.Exit(1)
		}
		defer conn.Close()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting grpc connection: %s\n", err)
			os.Exit(1)
		}

		client := pb.NewRoleServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		adminPtr := &isAdmin
		protectedPtr := &isProtected

		resp, err := client.CreateRole(ctx, &pb.CreateRoleRequest{
			GroupId:     group,
			Name:        name.(string),
			IsAdmin:     adminPtr,
			IsProtected: protectedPtr,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating role: %s\n", err)
			os.Exit(1)
		}
		role, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			cmd.Println("Created role: ", resp.Name)
		} else {
			cmd.Println("Created role:", string(role))
		}
	},
}

func init() {
	RoleCmd.AddCommand(Role_createCmd)
	Role_createCmd.Flags().StringP("name", "n", "", "Name of the role")
	Role_createCmd.Flags().BoolP("is_protected", "i", false, "Is the role protected")
	Role_createCmd.Flags().BoolP("is_admin", "a", false, "Is it an admin role")
	Role_createCmd.Flags().Int32P("group-id", "g", 0, "ID of the group which owns the role")
	if err := Role_createCmd.MarkFlagRequired("name"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
	if err := Role_createCmd.MarkFlagRequired("group-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
}
