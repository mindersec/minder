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
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// Role_createCmd represents the role create command
var Role_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a role within a mediator control plane",
	Long: `The medic role create subcommand lets you create new roles for a group
within a mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		org := util.GetConfigValue("org-id", "org-id", cmd, int32(0))
		group := util.GetConfigValue("group-id", "group-id", cmd, int32(0))
		name := util.GetConfigValue("name", "name", cmd, "")
		isAdmin := viper.GetBool("is_admin")
		isProtected := viper.GetBool("is_protected")

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		util.ExitNicelyOnError(err, "Error getting grpc connection")

		client := pb.NewRoleServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		adminPtr := &isAdmin
		protectedPtr := &isProtected

		m := protojson.MarshalOptions{
			Indent: "  ",
		}
		if group == 0 {
			// create a role by org
			resp, err := client.CreateRoleByOrganization(ctx, &pb.CreateRoleByOrganizationRequest{
				OrganizationId: org.(int32),
				Name:           name.(string),
				IsAdmin:        adminPtr,
				IsProtected:    protectedPtr,
			})
			util.ExitNicelyOnError(err, "Error creating role")

			output, err := m.Marshal(resp)
			util.ExitNicelyOnError(err, "error marshalling json")
			fmt.Println(string(output))
		} else {
			// create a role by group
			resp, err := client.CreateRoleByGroup(ctx, &pb.CreateRoleByGroupRequest{
				OrganizationId: org.(int32),
				GroupId:        group.(int32),
				Name:           name.(string),
				IsAdmin:        adminPtr,
				IsProtected:    protectedPtr,
			})
			util.ExitNicelyOnError(err, "Error creating role")
			output, err := m.Marshal(resp)
			util.ExitNicelyOnError(err, "error marshalling json")
			fmt.Println(string(output))
		}

	},
}

func init() {
	RoleCmd.AddCommand(Role_createCmd)
	Role_createCmd.Flags().StringP("name", "n", "", "Name of the role")
	Role_createCmd.Flags().BoolP("is_protected", "i", false, "Is the role protected")
	Role_createCmd.Flags().BoolP("is_admin", "a", false, "Is it an admin role")
	Role_createCmd.Flags().Int32P("org-id", "o", 0, "ID of the organization which owns the role")
	Role_createCmd.Flags().Int32P("group-id", "g", 0, "ID of the group which owns the role")
	if err := Role_createCmd.MarkFlagRequired("name"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
	if err := Role_createCmd.MarkFlagRequired("org-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
}
