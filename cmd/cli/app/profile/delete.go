// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a profile",
	Long:  `The profile delete subcommand lets you delete profiles within Minder.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %s", err)
		}
		return nil
	},
	RunE: deleteCommand,
}

// deleteCommand is the profile delete subcommand
func deleteCommand(cmd *cobra.Command, _ []string) error {
	project := viper.GetString("project")
	id := viper.GetString("id")
	name := viper.GetString("name")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, closeConn, err := cli.GetCLIClient(cmd, minderv1.NewProfileServiceClient)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer closeConn()

	// If name is provided, look up the profile ID first
	if name != "" {
		resp, err := client.GetProfileByName(cmd.Context(), &minderv1.GetProfileByNameRequest{
			Context: &minderv1.Context{Project: &project},
			Name:    name,
		})
		if err != nil {
			return cli.MessageAndError("Error looking up profile by name", err)
		}
		if resp.Profile == nil {
			return cli.MessageAndError("Error looking up profile by name", fmt.Errorf("profile not found"))
		}
		id = resp.Profile.GetId()
	}

	// Delete profile
	_, err = client.DeleteProfile(cmd.Context(), &minderv1.DeleteProfileRequest{
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
	deleteCmd.Flags().StringP("name", "n", "", "Name of profile to delete")
	// Require at least one of --id or --name
	deleteCmd.MarkFlagsOneRequired("id", "name")
	// Prevent providing both at the same time
	deleteCmd.MarkFlagsMutuallyExclusive("id", "name")
}
