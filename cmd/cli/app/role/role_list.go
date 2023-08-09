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
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func printRoles(rolesById *pb.GetRolesByGroupResponse, rolesByGroup *pb.GetRolesResponse, format string) {
	// print output in a table
	if format == "" {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Id", "Organization", "Group", "Name", "Is admin", "Is protected", "Created date", "Updated date"})

		var roles []*pb.RoleRecord
		if rolesById != nil {
			roles = rolesById.Roles
		} else {
			roles = rolesByGroup.Roles
		}

		for _, v := range roles {
			row := []string{
				fmt.Sprintf("%d", v.Id),
				fmt.Sprintf("%d", v.OrganizationId),
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
		var roles protoreflect.ProtoMessage
		if rolesById != nil {
			roles = rolesById
		} else {
			roles = rolesByGroup
		}
		out, err := util.GetJsonFromProto(roles)
		util.ExitNicelyOnError(err, "Error getting json from proto")
		fmt.Println(out)
	} else if format == "yaml" {
		var roles protoreflect.ProtoMessage
		if rolesById != nil {
			roles = rolesById
		} else {
			roles = rolesByGroup
		}
		out, err := util.GetYamlFromProto(roles)
		util.ExitNicelyOnError(err, "Error getting yaml from proto")
		fmt.Println(out)
	}

}

var role_listCmd = &cobra.Command{
	Use:   "list",
	Short: "List roles within a mediator control plane",
	Long: `The medic role list subcommand lets you list roles within a
mediator control plane for an specific group.`,
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

		org := viper.GetInt32("org-id")
		group := viper.GetInt32("group-id")
		limit := viper.GetInt32("limit")
		offset := viper.GetInt32("offset")
		format := viper.GetString("output")

		if format != "json" && format != "yaml" && format != "" {
			fmt.Fprintf(os.Stderr, "Error: invalid format: %s\n", format)
		}

		var limitPtr = &limit
		var offsetPtr = &offset

		// we need to set either org or group
		if org == 0 && group == 0 {
			fmt.Fprintf(os.Stderr, "Error: must set either org or group\n")
			os.Exit(1)
		}
		// if group is set , org cannot be set
		if group != 0 && org != 0 {
			fmt.Fprintf(os.Stderr, "Error: cannot set both org and group\n")
			os.Exit(1)
		}

		if group != 0 {
			resp, err := client.GetRolesByGroup(ctx, &pb.GetRolesByGroupRequest{GroupId: group, Limit: limitPtr, Offset: offsetPtr})
			util.ExitNicelyOnError(err, "Error getting roles")
			printRoles(resp, nil, format)
		} else if org != 0 {
			resp, err := client.GetRoles(ctx,
				&pb.GetRolesRequest{OrganizationId: org, Limit: limitPtr, Offset: offsetPtr})
			util.ExitNicelyOnError(err, "Error getting roles")
			printRoles(nil, resp, format)
		}
	},
}

func init() {
	RoleCmd.AddCommand(role_listCmd)
	role_listCmd.Flags().Int32P("org-id", "i", 0, "org id to list roles for")
	role_listCmd.Flags().Int32P("group-id", "g", 0, "group id to list roles for")
	role_listCmd.Flags().StringP("output", "o", "", "Output format (json or yaml)")
	role_listCmd.Flags().Int32P("limit", "l", -1, "Limit the number of results returned")
	role_listCmd.Flags().Int32P("offset", "f", 0, "Offset the results returned")
}
