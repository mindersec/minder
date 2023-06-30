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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
)

// Org_createCmd is the command for creating an organization
var Org_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an organization within a mediator control plane",
	Long: `The medic org create subcommand lets you create new organizations
within a mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// create the organization via GRPC
		name := util.GetConfigValue("name", "name", cmd, "")
		company := util.GetConfigValue("company", "company", cmd, "")
		create := util.GetConfigValue("create-default-records", "create-default-records", cmd, false)

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		util.ExitNicelyOnError(err, "Error getting grpc connection")

		client := pb.NewOrganizationServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		resp, err := client.CreateOrganization(ctx, &pb.CreateOrganizationRequest{
			Name:                 name.(string),
			Company:              company.(string),
			CreateDefaultRecords: create.(bool),
		})
		util.ExitNicelyOnError(err, "Error creating organization")

		org, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			cmd.Println("Created organization: ", resp.Name)
		} else {
			cmd.Println("Created organization:", string(org))
		}
	},
}

func init() {
	OrgCmd.AddCommand(Org_createCmd)
	Org_createCmd.Flags().StringP("name", "n", "", "Name of the organization")
	Org_createCmd.Flags().StringP("company", "c", "", "Company name of the organization")
	Org_createCmd.Flags().BoolP("create-default-records", "d", false, "Create default records for the organization")

	if err := Org_createCmd.MarkFlagRequired("name"); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
	}
	if err := Org_createCmd.MarkFlagRequired("company"); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
	}
}
