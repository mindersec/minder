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
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/mediator/cmd/cli/app"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

func printProject(project protoreflect.ProtoMessage, format string) {
	if format == app.JSON {
		out, err := util.GetJsonFromProto(project)
		util.ExitNicelyOnError(err, "Error getting json from proto")
		fmt.Println(out)
	} else if format == app.YAML {
		out, err := util.GetYamlFromProto(project)
		util.ExitNicelyOnError(err, "Error getting yaml from proto")
		fmt.Println(out)
	}
}

var project_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details for an project within a mediator control plane",
	Long: `The medic project get subcommand lets you retrieve details for a project within a
mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		conn, err := util.GrpcForCommand(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewProjectServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		id := viper.GetString("id")
		name := viper.GetString("name")
		format := util.GetConfigValue("output", "output", cmd, "").(string)
		if format == "" {
			format = app.JSON
		}
		if format != app.JSON && format != app.YAML && format != "" {
			fmt.Fprintf(os.Stderr, "Error: invalid format: %s\n", format)
		}

		// check for required options
		if id == "" && name == "" {
			fmt.Fprintf(os.Stderr, "Error: must specify one of id or name\n")
			os.Exit(1)
		}

		if id != "" && name != "" {
			fmt.Fprintf(os.Stderr, "Error: must specify either one of id or name\n")
			os.Exit(1)
		}

		// get by id
		if id != "" {
			project, err := client.GetProjectById(ctx, &pb.GetProjectByIdRequest{
				ProjectId: id,
			})
			util.ExitNicelyOnError(err, "Error getting project")
			printProject(project, format)
		} else if name != "" {
			// get by name
			project, err := client.GetProjectByName(ctx, &pb.GetProjectByNameRequest{
				Name: name,
			})
			util.ExitNicelyOnError(err, "Error getting project")
			printProject(project, format)
		}
	},
}

func init() {
	ProjectCmd.AddCommand(project_getCmd)
	project_getCmd.Flags().StringP("id", "i", "", "ID for the project to query")
	project_getCmd.Flags().StringP("name", "n", "", "Name for the project to query")
	project_getCmd.Flags().StringP("output", "o", "", "Output format (json or yaml)")
}
