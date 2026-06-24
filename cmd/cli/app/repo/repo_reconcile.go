// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app/project"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var reconcileCmd = &cobra.Command{
	Use:   "reconcile",
	Short: "Reconcile (Sync) a repository with Minder.",
	Long: `The reconcile command is used to trigger a reconciliation (sync) of a repository against
profiles and rules in a project.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}
		return nil
	},
	RunE: reconcileCommand,
}

// getCommand is the repo get subcommand
func reconcileCommand(cmd *cobra.Command, _ []string) error {
	name := viper.GetString("name")
	id := viper.GetString("id")
	projectName := viper.GetString("project")
	provider := viper.GetString("provider")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	entity := &minderv1.EntityTypedId{
		Type: minderv1.Entity_ENTITY_REPOSITORIES,
	}
	if id != "" {
		entity.Id = id
	}
	if name != "" {
		entity.Name = name
	}

	projectsClient, cleanup, err := project.GetProjectsClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer cleanup()

	_, err = projectsClient.CreateEntityReconciliationTask(cmd.Context(), &minderv1.CreateEntityReconciliationTaskRequest{
		Entity: entity,
		Context: &minderv1.Context{
			Provider: &provider,
			Project:  &projectName,
		},
	})
	if err != nil {
		return cli.MessageAndError("Error creating reconciliation task", err)
	}

	cmd.Println("Reconciliation task created")
	return nil
}

func init() {
	RepoCmd.AddCommand(reconcileCmd)
	reconcileCmd.Flags().StringP("name", "n", "", "Name of the repository (owner/repo)")
	reconcileCmd.Flags().StringP("id", "i", "", "ID of the repository")

	reconcileCmd.MarkFlagsOneRequired("name", "id")
	reconcileCmd.MarkFlagsMutuallyExclusive("name", "id")
}
