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

	"github.com/olekukonko/tablewriter"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var role_listCmd = &cobra.Command{
	Use:   "list",
	Short: "List roles within a mediator control plane",
	Long: `The medctl role list subcommand lets you list roles within a
mediator control plane for an specific group.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := util.GetGrpcConnection(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting grpc connection: %s\n", err)
			os.Exit(1)
		}
		defer conn.Close()

		client := pb.NewRoleServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		group := viper.GetInt32("group-id")
		limit := viper.GetInt32("limit")
		offset := viper.GetInt32("offset")
		format := viper.GetString("output")

		if format != "json" && format != "yaml" && format != "" {
			fmt.Fprintf(os.Stderr, "Error: invalid format: %s\n", format)
		}

		var limitPtr = &limit
		var offsetPtr = &offset

		resp, err := client.GetRoles(ctx, &pb.GetRolesRequest{
			GroupId: group,
			Limit:   limitPtr,
			Offset:  offsetPtr,
		})
		util.ExitNicelyOnError(err, "Error getting roles")

		// print output in a table
		if format == "" {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Id", "Group", "Name", "Is admin", "Is protected", "Created date", "Updated date"})

			for _, v := range resp.Roles {
				row := []string{
					fmt.Sprintf("%d", v.Id),
					fmt.Sprintf("%d", v.GroupId),
					v.Name,
					fmt.Sprintf("%t", v.IsAdmin),
					fmt.Sprintf("%t", v.IsProtected),
					v.GetCreatedAt().AsTime().Format(time.RFC3339),
					v.GetUpdatedAt().AsTime().Format(time.RFC3339),
				}
				table.Append(row)
			}
			table.Render()
		} else if format == "json" {
			output, err := json.MarshalIndent(resp.Roles, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshalling json: %s\n", err)
				os.Exit(1)
			}
			fmt.Println(string(output))
		} else if format == "yaml" {
			yamlData, err := yaml.Marshal(resp.Roles)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshalling yaml: %s\n", err)
				os.Exit(1)
			}
			fmt.Println(string(yamlData))

		}
	},
}

func init() {
	RoleCmd.AddCommand(role_listCmd)
	role_listCmd.Flags().Int32P("group-id", "g", 0, "group id to list roles for")
	role_listCmd.Flags().StringP("output", "o", "", "Output format (json or yaml)")
	role_listCmd.Flags().Int32P("limit", "l", -1, "Limit the number of results returned")
	role_listCmd.Flags().Int32P("offset", "f", 0, "Offset the results returned")
	if err := role_listCmd.MarkFlagRequired("group-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
}
