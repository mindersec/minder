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

package org

import (
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

var org_listCmd = &cobra.Command{
	Use:   "list",
	Short: "List organizations within a mediator control plane",
	Long: `The medic org list subcommand lets you list organizations within a
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

		client := pb.NewOrganizationServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		limit := viper.GetInt32("limit")
		offset := viper.GetInt32("offset")
		format := viper.GetString("output")

		if format != "json" && format != "yaml" && format != "" {
			fmt.Fprintf(os.Stderr, "Error: invalid format: %s\n", format)
		}

		var limitPtr = &limit
		var offsetPtr = &offset

		resp, err := client.GetOrganizations(ctx, &pb.GetOrganizationsRequest{
			Limit:  limitPtr,
			Offset: offsetPtr,
		})
		util.ExitNicelyOnError(err, "Error getting organizations")

		// print output in a table
		if format == "" {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Id", "Name", "Company", "Created date", "Updated date"})

			for _, v := range resp.Organizations {
				row := []string{
					fmt.Sprintf("%d", v.Id),
					v.Name,
					v.Company,
					v.GetCreatedAt().AsTime().Format(time.RFC3339),
					v.GetUpdatedAt().AsTime().Format(time.RFC3339),
				}
				table.Append(row)
			}
			table.Render()
		} else if format == "json" {
			output, err := json.MarshalIndent(resp.Organizations, "", "  ")
			util.ExitNicelyOnError(err, "Error marshalling json")
			fmt.Println(string(output))
		} else if format == "yaml" {
			yamlData, err := yaml.Marshal(resp.Organizations)
			util.ExitNicelyOnError(err, "Error marshalling yaml")
			fmt.Println(string(yamlData))

		}
	},
}

func init() {
	OrgCmd.AddCommand(org_listCmd)
	org_listCmd.Flags().StringP("output", "o", "", "Output format (json or yaml)")
	org_listCmd.Flags().Int32P("limit", "l", -1, "Limit the number of results returned")
	org_listCmd.Flags().Int32P("offset", "f", 0, "Offset the results returned")
}
