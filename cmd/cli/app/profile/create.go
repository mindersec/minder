// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// createCmd represents the profile create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a profile",
	Long:  `The profile create subcommand lets you create new profiles for a project within Minder.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %s", err)
		}
		return nil
	},
	RunE: createCommand,
}

// createCommand is the profile create subcommand
func createCommand(cmd *cobra.Command, _ []string) error {
	f := viper.GetString("file")
	project := viper.GetString("project")
	enableAlerts := viper.GetBool("enable-alerts")
	enableRems := viper.GetBool("enable-remediations")
	onOverride := "on"

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, closeConn, err := cli.GetCLIClient(cmd, minderv1.NewProfileServiceClient)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer closeConn()

	table := NewProfileRulesTable(cmd.OutOrStdout())

	createFunc := func(ctx context.Context, _ string, p *minderv1.Profile) (*minderv1.Profile, error) {
		if enableAlerts {
			p.Alert = &onOverride
		}
		if enableRems {
			p.Remediate = &onOverride
		}

		// create a profile
		resp, err := client.CreateProfile(ctx, &minderv1.CreateProfileRequest{
			Profile: p,
		})
		if err != nil {
			return nil, err
		}

		return resp.GetProfile(), nil
	}

	// cmd.Context() is the root context. We need to create a new context for each file
	// so we can avoid the timeout.
	profile, err := ExecOnOneProfile(cmd.Context(), table, f, cmd.InOrStdin(), project, createFunc)
	if err != nil {
		return cli.MessageAndError(fmt.Sprintf("error creating profile from %s", f), err)
	}

	// display the name above the table
	cmd.Printf("Successfully created new profile named: %s\n", profile.GetName())
	table.Render()
	return nil
}

func init() {
	ProfileCmd.AddCommand(createCmd)
	// Flags
	createCmd.Flags().StringP("file", "f", "", "Path to the YAML defining the profile (or - for stdin)")
	createCmd.Flags().Bool("enable-alerts", false, "Explicitly enable alerts for this profile. Overrides the YAML file.")
	createCmd.Flags().Bool("enable-remediations", false, "Explicitly enable remediations for this profile. Overrides the YAML file.")
	// Required
	if err := createCmd.MarkFlagRequired("file"); err != nil {
		createCmd.Printf("Error marking flag required: %s", err)
		os.Exit(1)
	}
}
