// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package offline_token provides the auth offline_token command for the minder CLI.
package offline_token

import (
	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/cli/app/auth"
)

// offlineTokenCmd represents the offline-token set of sub-commands
var offlineTokenCmd = &cobra.Command{
	Use:   "offline-token",
	Short: "Manage offline tokens",
	Long: `The minder auth offline-token command project lets you manage offline tokens
for the minder control plane.

Offline tokens are used to authenticate to the minder control plane without
requiring the user's presence. This is useful for long-running processes
that need to authenticate to the control plane.`,

	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	auth.AuthCmd.AddCommand(offlineTokenCmd)
}
