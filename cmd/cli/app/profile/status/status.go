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

// Package status provides the CLI subcommand for managing profile statuses
package status

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/cmd/cli/app/profile"
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
	profileStatusCmd.PersistentFlags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	// Required
	if err := profileStatusCmd.MarkPersistentFlagRequired("name"); err != nil {
		profileStatusCmd.Printf("Error marking flag required: %s", err)
		os.Exit(1)
	}
}
