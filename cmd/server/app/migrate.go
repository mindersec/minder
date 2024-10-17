// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration tool",
	Long:  `Use tool with a combination of up to down to migrate the database.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	RootCmd.AddCommand(migrateCmd)
	migrateCmd.PersistentFlags().BoolP("yes", "y", false, "Answer yes to all questions")
	migrateCmd.PersistentFlags().UintP("num-steps", "n", 0, "Number of steps to migrate")
}
