//
// Copyright 2024 Stacklok, Inc.
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

package project

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// projectDeleteCmd is the command for deleting sub-projects
var projectDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a sub-project within a minder control plane",
	Long:  `Delete a sub-project within a minder control plane`,
	RunE:  cli.GRPCClientWrapRunE(deleteCommand),
}

// listCommand is the command for listing projects
func deleteCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewProjectsServiceClient(conn)

	project := viper.GetString("project")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	resp, err := client.DeleteProject(ctx, &minderv1.DeleteProjectRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
	})
	if err != nil {
		return cli.MessageAndError("Error deleting sub-project", err)
	}

	cmd.Println("Successfully deleted project with id:", resp.ProjectId)

	return nil
}

func init() {
	ProjectCmd.AddCommand(projectDeleteCmd)

	projectDeleteCmd.Flags().StringP("project", "j", "", "The sub-project to delete")
	// mark as required
	if err := projectDeleteCmd.MarkFlagRequired("project"); err != nil {
		panic(err)
	}
}
