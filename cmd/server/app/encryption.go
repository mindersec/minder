// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"github.com/spf13/cobra"
)

// encryptionCmd groups together the encryption-related commands
var encryptionCmd = &cobra.Command{
	Use:   "encryption",
	Short: "Tools for managing encryption keys",
	Long:  `Use with rotate to re-encrypt provider access tokens with new keys/algorithms`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	RootCmd.AddCommand(encryptionCmd)
	encryptionCmd.PersistentFlags().BoolP("yes", "y", false, "Answer yes to all questions")
}
