// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package offline_token provides the auth offline_token command for the minder CLI.
package offline_token

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/pkg/config"
	clientconfig "github.com/mindersec/minder/pkg/config/client"
)

// offlineTokenGetCmd represents the offline-token get command
var offlineTokenGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Retrieve an offline token",
	Long: `The minder auth offline-token get command project lets you retrieve an offline token
for the minder control plane.

Offline tokens are used to authenticate to the minder control plane without
requiring the user's presence. This is useful for long-running processes
that need to authenticate to the control plane.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return err
		}
		return viper.BindPFlag("offline.get.skip-browser", cmd.Flags().Lookup("skip-browser"))
	},
	RunE: offlineGetCommand,
}

// offlineGetCommand is the offline-token get subcommand
func offlineGetCommand(cmd *cobra.Command, _ []string) error {
	clientConfig, err := config.ReadConfigFromViper[clientconfig.Config](viper.GetViper())
	if err != nil {
		return fmt.Errorf("error reading config: %w", err)
	}

	f := viper.GetString("file")
	skipBrowser := viper.GetBool("offline.get.skip-browser")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// wait for the token to be received
	token, err := cli.Login(cmd.Context(), cmd, clientConfig, []string{"offline_access"}, skipBrowser)
	if err != nil {
		return err
	}

	// write the token to the file
	if err := os.WriteFile(f, []byte(token.RefreshToken), 0600); err != nil {
		return fmt.Errorf("error writing offline token to file: %w", err)
	}

	cmd.PrintErrf("Offline token written to %s\n", f)

	return nil
}

func init() {
	offlineTokenCmd.AddCommand(offlineTokenGetCmd)

	offlineTokenGetCmd.Flags().StringP("file", "f", "offline.token", "The file to write the offline token to")
	offlineTokenGetCmd.Flags().BoolP("skip-browser", "", false, "Skip opening the browser for OAuth flow")
}
