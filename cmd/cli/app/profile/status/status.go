// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package status provides the CLI subcommand for managing profile statuses
package status

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/cmd/cli/app/profile"
)

// profileStatusCmd is the root command for the profile_status subcommands
var profileStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Manage profile status",
	Long:  `The profile status subcommand allows management of profile status within Minder.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	profile.ProfileCmd.AddCommand(profileStatusCmd)
	// Flags
	profileStatusCmd.PersistentFlags().StringP("name", "n", "", "Profile name to get profile status for")
	profileStatusCmd.PersistentFlags().StringP("id", "i", "", "ID to get profile status for")
	profileStatusCmd.PersistentFlags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	// Required
	profileStatusCmd.MarkFlagsOneRequired("id", "name")
}
