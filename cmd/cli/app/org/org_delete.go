//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.role/licenses/LICENSE-2.0
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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

var org_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a organization within a mediator controlplane",
	Long: `The medic org delete subcommand lets you delete organizations within a
mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// delete the org via GRPC
		id := util.GetConfigValue("org-id", "org-id", cmd, int32(0)).(int32)
		force := util.GetConfigValue("force", "force", cmd, false).(bool)

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")

		defer conn.Close()

		client := pb.NewOrganizationServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		forcePtr := &force
		_, err = client.DeleteOrganization(ctx, &pb.DeleteOrganizationRequest{
			Id:    id,
			Force: forcePtr,
		})

		util.ExitNicelyOnError(err, "Error deleting organization")
		cmd.Println("Successfully deleted organization with id:", id)
	},
}

func init() {
	OrgCmd.AddCommand(org_deleteCmd)
	org_deleteCmd.Flags().Int32P("org-id", "o", 0, "id of organization to delete")
	org_deleteCmd.Flags().BoolP("force", "f", false,
		"Force deletion of organization, even if it has associated groups")
	err := org_deleteCmd.MarkFlagRequired("org-id")
	util.ExitNicelyOnError(err, "Error marking flag as required")
}
