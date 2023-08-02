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
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

var role_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details for an role within a mediator control plane",
	Long: `The medic role get subcommand lets you retrieve details for an role within a
mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewRoleServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		id := viper.GetInt32("id")
		org_id := viper.GetInt32("org-id")
		name := viper.GetString("name")

		// check for required options
		if id == 0 && name == "" && org_id == 0 {
			fmt.Fprintf(os.Stderr, "Error: must specify one of id or org_id+name\n")
			os.Exit(1)
		}

		if id > 0 && (name != "" || org_id > 0) {
			fmt.Fprintf(os.Stderr, "Error: must specify either one of id or org_id+name\n")
			os.Exit(1)
		}

		// if name is specified, org_id must also be specified
		if (name != "" && org_id == 0) || (name == "" && org_id > 0) {
			fmt.Fprintf(os.Stderr, "Error: must specify both org_id and name\n")
			os.Exit(1)
		}

		var roleRecord *pb.RoleRecord
		// get by id
		if id > 0 {
			role, err := client.GetRoleById(ctx, &pb.GetRoleByIdRequest{
				Id: id,
			})
			util.ExitNicelyOnError(err, "Error getting role")
			if role != nil {
				roleRecord = role.Role
			}
		} else if name != "" {
			// get by name
			role, err := client.GetRoleByName(ctx, &pb.GetRoleByNameRequest{
				OrganizationId: org_id,
				Name:           name,
			})
			util.ExitNicelyOnError(err, "Error getting role")
			roleRecord = role.GetRole()
		}
		json, err := json.Marshal(roleRecord)
		util.ExitNicelyOnError(err, "Error marshalling role")
		fmt.Println(string(json))
	},
}

func init() {
	RoleCmd.AddCommand(role_getCmd)
	role_getCmd.Flags().Int32P("id", "i", 0, "ID for the role to query")
	role_getCmd.Flags().Int32P("org-id", "o", 0, "Organization ID")
	role_getCmd.Flags().StringP("name", "n", "", "Name for the role to query")
}
