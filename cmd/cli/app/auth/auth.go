// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package auth provides the auth command project for the minder CLI.
package auth

import (
	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/cli/app"
)

// AuthCmd represents the account command
var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authorize and manage accounts within a minder control plane",
	Long: `The minder auth command project lets you create accounts and grant or revoke
authorization to existing accounts within a minder control plane.`,

	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(AuthCmd)
}
