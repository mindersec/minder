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

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var role_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details for an role within a mediator control plane",
	Long: `The medctl role get subcommand lets you retrieve details for an role within a
mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := util.GetGrpcConnection(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewRoleServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		id := viper.GetInt32("id")
		group_id := viper.GetInt32("group-id")
		name := viper.GetString("name")

		// check for required options
		if id == 0 && name == "" && group_id == 0 {
			fmt.Fprintf(os.Stderr, "Error: must specify one of id or group_id+name\n")
			os.Exit(1)
		}

		if id > 0 && (name != "" || group_id > 0) {
			fmt.Fprintf(os.Stderr, "Error: must specify either one of id or group_id+name\n")
			os.Exit(1)
		}

		// if name is specified, group_id must also be specified
		if (name != "" && group_id == 0) || (name == "" && group_id > 0) {
			fmt.Fprintf(os.Stderr, "Error: must specify both group_id and name\n")
			os.Exit(1)
		}

		var roleRecord *pb.RoleRecord
		// get by id
		if id > 0 {
			role, _ := client.GetRoleById(ctx, &pb.GetRoleByIdRequest{
				Id: id,
			})
			if role != nil {
				roleRecord = role.Role
			}
		} else if name != "" {
			// get by name
			role, _ := client.GetRoleByName(ctx, &pb.GetRoleByNameRequest{
				GroupId: group_id,
				Name:    name,
			})
			if role != nil {
				roleRecord = role.Role
			}
		}

		if roleRecord == nil {
			fmt.Fprintf(os.Stderr, "Error getting role\n")
			os.Exit(1)
		}
		json, err := json.Marshal(roleRecord)
		util.ExitNicelyOnError(err, "Error marshalling role")
		fmt.Println(string(json))
	},
}

func init() {
	RoleCmd.AddCommand(role_getCmd)
	role_getCmd.Flags().Int32P("id", "i", 0, "ID for the role to query")
	role_getCmd.Flags().Int32P("group-id", "g", 0, "Group ID")
	role_getCmd.Flags().StringP("name", "n", "", "Name for the role to query")
}
