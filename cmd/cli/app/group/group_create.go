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

var group_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a group within a mediator control plane",
	Long: `The medctl group create subcommand lets you create new groups within
a mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {

		name := util.GetConfigValue("name", "name", cmd, "")
		description := util.GetConfigValue("description", "description", cmd, "")
		organisation := util.GetConfigValue("org-id", "org-id", cmd, int32(0)).(int32)
		isProtected := util.GetConfigValue("is_protected", "is_protected", cmd, false).(bool)

		conn, err := util.GetGrpcConnection(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting grpc connection: %s\n", err)
			os.Exit(1)
		}
		defer conn.Close()

		client := pb.NewGroupServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		protectedPtr := &isProtected

		resp, err := client.CreateGroup(ctx, &pb.CreateGroupRequest{
			Name:           name.(string),
			Description:    description.(string),
			OrganisationId: organisation,
			IsProtected:    protectedPtr,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating group: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Group created: %s\n", resp.String())

	},
}

func init() {
	GroupCmd.AddCommand(group_createCmd)
	group_createCmd.Flags().StringP("name", "n", "", "Name of the group")
	group_createCmd.Flags().StringP("description", "d", "", "Description of the group")
	group_createCmd.Flags().Int32("org-id", 0, "Organisation ID")
	group_createCmd.Flags().BoolP("is_protected", "i", false, "Is the group protected")

	if err := group_createCmd.MarkFlagRequired("name"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
	if err := group_createCmd.MarkFlagRequired("description"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
	if err := group_createCmd.MarkFlagRequired("org-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
}
