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

package project

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// Project_createCmd is the command for creating a project
var Project_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a project within a mediator control plane",
	Long: `The medic project create subcommand lets you create new projects within
a mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {

		name := util.GetConfigValue("name", "name", cmd, "")
		description := util.GetConfigValue("description", "description", cmd, "")
		organization := viper.GetString("org-id")
		isProtected := viper.GetBool("is_protected")

		conn, err := util.GrpcForCommand(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewProjectServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		protectedPtr := &isProtected

		resp, err := client.CreateProject(ctx, &pb.CreateProjectRequest{
			Name:           name.(string),
			Description:    description.(string),
			OrganizationId: organization,
			IsProtected:    protectedPtr,
		})
		util.ExitNicelyOnError(err, "Error creating project")
		out, err := util.GetJsonFromProto(resp)
		util.ExitNicelyOnError(err, "Error getting json from proto")
		fmt.Println(out)
	},
}

func init() {
	ProjectCmd.AddCommand(Project_createCmd)
	Project_createCmd.Flags().StringP("name", "n", "", "Name of the project")
	Project_createCmd.Flags().StringP("description", "d", "", "Description of the project")
	Project_createCmd.Flags().String("org-id", "", "Organization ID")
	Project_createCmd.Flags().BoolP("is_protected", "i", false, "Is the project protected")

	if err := Project_createCmd.MarkFlagRequired("name"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
	if err := Project_createCmd.MarkFlagRequired("description"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
	if err := Project_createCmd.MarkFlagRequired("org-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
}
