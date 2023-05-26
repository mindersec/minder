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
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
)

var role_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a role within a mediator control plane",
	Long: `The medctl role create subcommand lets you create new roles for an organization
within a mediator control plane.`,
	Run: func(cmd *cobra.Command, args []string) {
		// create the role via GRPC
		group := util.GetConfigValue("group", "group", cmd, nil)
		name := util.GetConfigValue("name", "name", cmd, "")
		isAdmin := util.GetConfigValue("is_admin", "is_admin", cmd, false)
		isProtected := util.GetConfigValue("is_protected", "is_protected", cmd, false)

		conn, err := util.GetGrpcConnection(cmd)
		defer conn.Close()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting grpc connection: %s\n", err)
			os.Exit(1)
		}

		client := pb.NewRoleServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		resp, err := client.CreateRole(ctx, &pb.CreateRoleRequest{
			GroupId:     group.(int32),
			Name:        name.(string),
			IsAdmin:     isAdmin.(*bool),
			IsProtected: isProtected.(*bool),
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating role: %s\n", err)
			os.Exit(1)
		}
		cmd.Println("Created role:", resp.Name)
	},
}

func init() {
	RoleCmd.AddCommand(role_createCmd)
	role_createCmd.Flags().Uint64P("group", "g", 0, "ID of the group which owns the role")
	role_createCmd.Flags().StringP("name", "n", "", "Name of the role")
	role_createCmd.Flags().BoolP("is_admin", "a", false, "Is it an admin role")
	role_createCmd.Flags().BoolP("is_protected", "p", false, "Is it a protected role")
	err := role_createCmd.MarkPersistentFlagRequired("group")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
	}
	err = role_createCmd.MarkPersistentFlagRequired("name")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
	}
	if err := viper.BindPFlags(role_createCmd.PersistentFlags()); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
	}
}
