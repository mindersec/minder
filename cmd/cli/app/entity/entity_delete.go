// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an entity",
	Long:  `The entity delete subcommand is used to delete an entity instance within Minder.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}
		return nil
	},
	RunE: deleteCommand,
}

// deleteCommand is the entity delete subcommand
func deleteCommand(cmd *cobra.Command, _ []string) error {
	client, closeConn, err := cli.GetCLIClient(cmd, minderv1.NewEntityInstanceServiceClient)
	if err != nil {
		return cli.MessageAndError("Error creating gRPC client", err)
	}
	defer closeConn()

	project := viper.GetString("project")
	provider := viper.GetString("provider")
	id := viper.GetString("id")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	resp, err := client.DeleteEntityById(cmd.Context(), &minderv1.DeleteEntityByIdRequest{
		Context: &minderv1.ContextV2{
			ProjectId: project,
			Provider:  provider,
		},
		Id: id,
	})
	if err != nil {
		return cli.MessageAndError("Error deleting entity", err)
	}

	cmd.Printf("Successfully deleted entity with ID: %s\n", resp.GetId())
	return nil
}

func init() {
	EntityCmd.AddCommand(deleteCmd)
	// Flags
	deleteCmd.Flags().StringP("id", "i", "", "ID of the entity to delete")
	if err := deleteCmd.MarkFlagRequired("id"); err != nil {
		panic(err)
	}
}
