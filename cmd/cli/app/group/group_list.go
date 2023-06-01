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
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
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

		// org-id is required (later we could change this based on the users
		// session by mapping user-id to org-id). It may still be useful though
		// as a user might be able to belong to multiple organisations.
		if !cmd.Flags().Changed("org-id") {
			fmt.Fprintf(os.Stderr, "Error: --org-id must be set\n")
			os.Exit(1)
		}

		name := util.GetConfigValue("name", "name", cmd, "").(string)
		groupID := util.GetConfigValue("group-id", "group-id", cmd, int(0)).(int)
		organisation := util.GetConfigValue("org-id", "org-id", cmd, int(0)).(int)
		limit := util.GetConfigValue("limit", "limit", cmd, int(0)).(int)
		offset := util.GetConfigValue("offset", "offset", cmd, int(0)).(int)

		switch {
		case groupID != 0:
			resp, err := client.GetGroupById(ctx, &pb.GetGroupByIdRequest{
				GroupId: int32(groupID),
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting group: %s\n", err)
				os.Exit(1)
			}
			fmt.Printf("Group: %v\n", resp.Name)

		case name != "":
			resp, err := client.GetGroupByName(ctx, &pb.GetGroupByNameRequest{
				Name: name,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting group: %s\n", err)
				os.Exit(1)
			}
			fmt.Printf("Group ID: %v\n", resp.GroupId)

		default:
			resp, err := client.GetGroups(ctx, &pb.GetGroupsRequest{
				OrganisationId: int32(organisation),
				Limit:          int32(limit),
				Offset:         int32(offset),
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting groups: %s\n", err)
				os.Exit(1)
			}
			fmt.Printf("Groups: %v\n", resp)
		}
	},
}

func init() {
	GroupCmd.AddCommand(group_listCmd)
	group_listCmd.Flags().Int("group-id", 0, "Group ID")
	group_listCmd.Flags().StringP("name", "n", "", "List group values by a name")
	group_listCmd.Flags().Int("org-id", 0, "Organisation ID")
	group_listCmd.Flags().Int("limit", 10, "Limit number of results")
	group_listCmd.Flags().Int("offset", 0, "Offset number of results")
}
