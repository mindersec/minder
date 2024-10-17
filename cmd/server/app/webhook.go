// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"github.com/spf13/cobra"
)

// cmdWebhook is the root command for the webhook subcommands
func cmdWebhook() *cobra.Command {
	var whCmd = &cobra.Command{
		Use:   "webhook",
		Short: "Webhook management tool",
	}

	whCmd.AddCommand(cmdWebhookUpdate())
	return whCmd
}

func init() {
	RootCmd.AddCommand(cmdWebhook())
}
