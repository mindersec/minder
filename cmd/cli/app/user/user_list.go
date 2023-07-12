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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var user_listCmd = &cobra.Command{
	Use:   "list",
	Short: "List users within a mediator control plane",
	Long: `The medic user list subcommand lets you list users within a
mediator control plane for an specific role.`,
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

		client := pb.NewUserServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		org := viper.GetInt32("org-id")
		group := viper.GetInt32("group-id")
		limit := viper.GetInt32("limit")
		offset := viper.GetInt32("offset")
		format := viper.GetString("output")

		if format != "json" && format != "yaml" && format != "" {
			fmt.Fprintf(os.Stderr, "Error: invalid format: %s\n", format)
		}

		// need to set either group or org
		if org == 0 && group == 0 {
			fmt.Fprintf(os.Stderr, "Error: must set either org-id or group-id\n")
			os.Exit(1)
		}

		// if group id is set, org id cannot be set
		if (org != 0) && (group != 0) {
			fmt.Fprintf(os.Stderr, "Error: cannot set both org-id and group-id\n")
			os.Exit(1)
		}

		var limitPtr = &limit
		var offsetPtr = &offset

		// call depending on parameters
		var users []*pb.UserRecord

		if org != 0 {
			resp, err := client.GetUsersByOrganization(ctx,
				&pb.GetUsersByOrganizationRequest{OrganizationId: org, Limit: limitPtr, Offset: offsetPtr})
			util.ExitNicelyOnError(err, "Error getting users")
			users = resp.Users
		} else if group != 0 {
			resp, err := client.GetUsersByGroup(ctx, &pb.GetUsersByGroupRequest{GroupId: group, Limit: limitPtr, Offset: offsetPtr})
			util.ExitNicelyOnError(err, "Error getting users")
			users = resp.Users
		}

		// print output in a table
		if format == "" {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Id", "Role", "Username", "Email", "First name",
				"Last name", "Is protected", "Created date", "Updated date"})

			for _, v := range users {
				if v.IsProtected == nil {
					*v.IsProtected = false
				}

				row := []string{
					fmt.Sprintf("%d", v.Id),
					fmt.Sprintf("%d", v.OrganizationId),
					v.Username,
					*v.Email,
					*v.FirstName,
					*v.LastName,
					fmt.Sprintf("%t", *v.IsProtected),
					v.GetCreatedAt().AsTime().Format(time.RFC3339),
					v.GetUpdatedAt().AsTime().Format(time.RFC3339),
				}
				table.Append(row)
			}
			table.Render()
		} else if format == "json" {
			output, err := json.MarshalIndent(users, "", "  ")
			util.ExitNicelyOnError(err, "Error marshalling json")
			fmt.Println(string(output))
		} else if format == "yaml" {
			yamlData, err := yaml.Marshal(users)
			util.ExitNicelyOnError(err, "Error marshalling yaml")
			fmt.Println(string(yamlData))

		}
	},
}

func init() {
	UserCmd.AddCommand(user_listCmd)
	user_listCmd.Flags().Int32P("org-id", "i", 0, "org id to list users for")
	user_listCmd.Flags().Int32P("group-id", "g", 0, "group id to list users for")
	user_listCmd.Flags().StringP("output", "o", "", "Output format (json or yaml)")
	user_listCmd.Flags().Int32P("limit", "l", -1, "Limit the number of results returned")
	user_listCmd.Flags().Int32P("offset", "f", 0, "Offset the results returned")
}
