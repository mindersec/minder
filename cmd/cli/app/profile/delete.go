// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a profile",
	Long:  `The profile delete subcommand lets you delete profiles within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(deleteCommand),
}

// deleteCommand is the profile delete subcommand
func deleteCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewProfileServiceClient(conn)

	project := viper.GetString("project")
	id := viper.GetString("id")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// Delete profile
	_, err := client.DeleteProfile(ctx, &minderv1.DeleteProfileRequest{
		Context: &minderv1.Context{Project: &project},
		Id:      id,
	})
	if err != nil {
		return cli.MessageAndError("Error deleting profile", err)
	}

	cmd.Println("Successfully deleted profile with id:", id)

	return nil
}

func init() {
	ProfileCmd.AddCommand(deleteCmd)
	// Flags
	deleteCmd.Flags().StringP("id", "i", "", "ID of profile to delete")
	// TODO: add a flag for the profile name
	// Required
	if err := deleteCmd.MarkFlagRequired("id"); err != nil {
		deleteCmd.Printf("Error marking flag required: %s", err)
		os.Exit(1)
	}
}
