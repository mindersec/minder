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
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
)

var org_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a organisation within a mediator controlplane",
	Long: `The medctl org delete subcommand lets you delete organisations within a
mediator control plane.`,
	Run: func(cmd *cobra.Command, args []string) {
		// delete the org via GRPC
		id := util.GetConfigValue("org-id", "org-id", cmd, int32(0)).(int32)
		force := util.GetConfigValue("force", "force", cmd, false).(bool)

		conn, err := util.GetGrpcConnection(cmd)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting grpc connection: %s\n", err)
			os.Exit(1)
		}
		defer conn.Close()

		client := pb.NewOrganisationServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		forcePtr := &force
		_, err = client.DeleteOrganisation(ctx, &pb.DeleteOrganisationRequest{
			Id:    id,
			Force: forcePtr,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting organisation: %s\n", err)
			os.Exit(1)
		}
		cmd.Println("Successfully deleted organisation with id:", id)
	},
}

func init() {
	OrgCmd.AddCommand(org_deleteCmd)
	org_deleteCmd.Flags().Int32P("org-id", "o", 0, "id of organisation to delete")
	org_deleteCmd.Flags().BoolP("force", "f", false,
		"Force deletion of organisation, even if it has associated groups")
	if err := org_deleteCmd.MarkFlagRequired("org-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

	if err := viper.BindPFlags(org_deleteCmd.Flags()); err != nil {
		log.Fatal(err)
	}
}
