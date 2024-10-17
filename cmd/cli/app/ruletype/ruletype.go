// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package ruletype provides the CLI subcommand for managing rules
package ruletype

import (
	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/cli/app"
)

// ruleTypeCmd is the root command for the rule subcommands
var ruleTypeCmd = &cobra.Command{
	Use:   "ruletype",
	Short: "Manage rule types",
	Long:  `The ruletype subcommands allows the management of rule types within Minder.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(ruleTypeCmd)
	// Flags for all subcommands
	ruleTypeCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}
