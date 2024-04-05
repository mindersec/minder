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

package repo

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var reconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "Reconcile (Sync) a repository with Minder.",
	Long: `The reconcile command is used to trigger a reconciliation (sync) of a repository against
profiles and rules in a project.`,
	RunE: cli.GRPCClientWrapRunE(reconcileCommand),
}

// getCommand is the repo get subcommand
func reconcileCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	name := viper.GetString("name")
	id := viper.GetString("id")
	project := viper.GetString("project")
	provider := viper.GetString("provider")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	if id == "" {
		repoClient := minderv1.NewRepositoryServiceClient(conn)

		repo, err := repoClient.GetRepositoryByName(ctx, &minderv1.GetRepositoryByNameRequest{
			Name: name,
			Context: &minderv1.Context{
				Provider: &provider,
				Project:  &project,
			},
		})
		if err != nil {
			return cli.MessageAndError("Failed to get repository", err)
		}

		id = repo.GetRepository().GetId()
	}

	projectsClient := minderv1.NewProjectsServiceClient(conn)
	_, err := projectsClient.CreateEntityReconciliationTask(ctx, &minderv1.CreateEntityReconciliationTaskRequest{
		Entity: &minderv1.EntityTypedId{
			Id:   id,
			Type: minderv1.Entity_ENTITY_REPOSITORIES,
		},
		Context: &minderv1.Context{
			Provider: &provider,
			Project:  &project,
		},
	})
	if err != nil {
		return cli.MessageAndError("Error creating reconciliation task", err)
	}

	fmt.Println("Reconciliation task created")
	return nil
}

func init() {
	RepoCmd.AddCommand(reconcileCmd)
	reconcileCmd.Flags().StringP("name", "n", "", "Name of the repository (owner/repo)")
	reconcileCmd.Flags().StringP("id", "i", "", "ID of the repository")

	reconcileCmd.MarkFlagsOneRequired("name", "id")
	reconcileCmd.MarkFlagsMutuallyExclusive("name", "id")
}
