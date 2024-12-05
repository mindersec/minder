// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package invite provides the auth invite command for the minder CLI.
package invite

import (
	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/cli/app/auth"
)

// inviteCmd represents the offline-token set of sub-commands
var inviteCmd = &cobra.Command{
	Use:   "invite",
	Short: "Manage user invitations",
	Long:  `The minder auth invite command lets you manage (accept/decline/list) your invitations.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	auth.AuthCmd.AddCommand(inviteCmd)
}
