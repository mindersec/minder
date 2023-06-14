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

var group_listCmd = &cobra.Command{
	Use:   "list",
	Short: "medctl group list",
	Long: `The medctl group list subcommand lets you list groups within
a mediator control plane.`,
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

		client := pb.NewGroupServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		org := viper.GetInt32("org-id")
		limit := viper.GetInt32("limit")
		offset := viper.GetInt32("offset")
		format := viper.GetString("output")

		if format != "json" && format != "yaml" && format != "" {
			fmt.Fprintf(os.Stderr, "Error: invalid format: %s\n", format)
		}

		resp, err := client.GetGroups(ctx, &pb.GetGroupsRequest{
			OrganizationId: org,
			Limit:          limit,
			Offset:         offset,
		})
		util.ExitNicelyOnError(err, "Error getting groups")

		// print output in a table
		if format == "" {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Id", "Organization", "Name", "Is protected", "Created date", "Updated date"})

			for _, v := range resp.Groups {
				row := []string{
					fmt.Sprintf("%d", v.GroupId),
					fmt.Sprintf("%d", v.OrganizationId),
					v.Name,
					fmt.Sprintf("%t", v.IsProtected),
					v.GetCreatedAt().AsTime().Format(time.RFC3339),
					v.GetUpdatedAt().AsTime().Format(time.RFC3339),
				}
				table.Append(row)
			}
			table.Render()
		} else if format == "json" {
			output, err := json.MarshalIndent(resp.Groups, "", "  ")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshalling json: %s\n", err)
				os.Exit(1)
			}
			fmt.Println(string(output))
		} else if format == "yaml" {
			yamlData, err := yaml.Marshal(resp.Groups)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshalling yaml: %s\n", err)
				os.Exit(1)
			}
			fmt.Println(string(yamlData))

		}
	},
}

func init() {
	GroupCmd.AddCommand(group_listCmd)
	group_listCmd.Flags().Int32P("org-id", "i", 0, "org id to list groups for")
	group_listCmd.Flags().StringP("output", "o", "", "Output format")
	group_listCmd.Flags().Int32P("limit", "l", -1, "Limit the number of results returned")
	group_listCmd.Flags().Int32P("offset", "f", 0, "Offset the results returned")
	if err := group_listCmd.MarkFlagRequired("org-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
}
