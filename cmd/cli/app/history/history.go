// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package history provides the CLI subcommand for managing profile statuses
package history

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/cli/app"
)

// historyCmd is the root command for the profile_status subcommands
var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "View evaluation history",
	Long:  `The history subcommands allows evaluation history to be viewed.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(historyCmd)
	historyCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
	historyCmd.PersistentFlags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
