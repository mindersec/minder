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
	"encoding/json"
	"fmt"
	"os"

	"github.com/stacklok/mediator/cmd/cli/app"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type output struct {
	Group *pb.GroupRecord  `json:"group"`
	Roles []*pb.RoleRecord `json:"roles"`
	Users []*pb.UserRecord `json:"users"`
}

func printGroup(group *pb.GroupRecord, roles []*pb.RoleRecord, users []*pb.UserRecord, format string) {
	output := output{
		Group: group,
		Roles: roles,
		Users: users,
	}
	if format == app.JSON {
		output, err := json.MarshalIndent(output, "", "  ")
		util.ExitNicelyOnError(err, "Error marshalling json")
		fmt.Println(string(output))
	} else if format == app.YAML {
		yamlData, err := yaml.Marshal(output)
		util.ExitNicelyOnError(err, "Error marshalling yaml")
		fmt.Println(string(yamlData))

	}
}

var group_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details for an group within a mediator control plane",
	Long: `The medic group get subcommand lets you retrieve details for a group within a
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

		client := pb.NewGroupServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		id := viper.GetInt32("id")
		name := viper.GetString("name")
		format := util.GetConfigValue("output", "output", cmd, "").(string)
		if format == "" {
			format = app.JSON
		}
		if format != app.JSON && format != app.YAML && format != "" {
			fmt.Fprintf(os.Stderr, "Error: invalid format: %s\n", format)
		}

		// check for required options
		if id == 0 && name == "" {
			fmt.Fprintf(os.Stderr, "Error: must specify one of id or name\n")
			os.Exit(1)
		}

		if id > 0 && name != "" {
			fmt.Fprintf(os.Stderr, "Error: must specify either one of id or name\n")
			os.Exit(1)
		}

		// get by id
		if id > 0 {
			group, err := client.GetGroupById(ctx, &pb.GetGroupByIdRequest{
				GroupId: id,
			})
			util.ExitNicelyOnError(err, "Error getting group")
			printGroup(group.Group, group.Roles, group.Users, format)
		} else if name != "" {
			// get by name
			group, err := client.GetGroupByName(ctx, &pb.GetGroupByNameRequest{
				Name: name,
			})
			util.ExitNicelyOnError(err, "Error getting group")
			printGroup(group.Group, group.Roles, group.Users, format)
		}
	},
}

func init() {
	GroupCmd.AddCommand(group_getCmd)
	group_getCmd.Flags().Int32P("id", "i", 0, "ID for the role to query")
	group_getCmd.Flags().Int32P("group-id", "g", 0, "Group ID")
	group_getCmd.Flags().StringP("name", "n", "", "Name for the role to query")
	group_getCmd.Flags().StringP("output", "o", "", "Output format (json or yaml)")
}
