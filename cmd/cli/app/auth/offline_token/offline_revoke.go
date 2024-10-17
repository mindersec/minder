// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package offline_token provides the auth offline_token command for the minder CLI.
package offline_token

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/mindersec/minder/internal/config"
	clientconfig "github.com/mindersec/minder/internal/config/client"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
)

// offlineTokenRevokeCmd represents the offline-token use command
var offlineTokenRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke an offline token",
	Long: `The minder auth offline-token use command project lets you revoke an offline token
for the minder control plane.

Offline tokens are used to authenticate to the minder control plane without
requiring the user's presence. This is useful for long-running processes
that need to authenticate to the control plane.`,

	RunE: cli.GRPCClientWrapRunE(offlineRevokeCommand),
}

// offlineRevokeCommand is the offline-token revoke subcommand
func offlineRevokeCommand(_ context.Context, cmd *cobra.Command, _ []string, _ *grpc.ClientConn) error {
	clientConfig, err := config.ReadConfigFromViper[clientconfig.Config](viper.GetViper())
	if err != nil {
		return fmt.Errorf("error reading config: %w", err)
	}

	f := viper.GetString("file")
	tok := viper.GetString("token")
	if tok == "" {
		fpath := filepath.Clean(f)
		tokbytes, err := os.ReadFile(fpath)
		if err != nil {
			return fmt.Errorf("error reading file: %w", err)
		}

		tok = string(tokbytes)
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	issuerUrlStr := clientConfig.Identity.CLI.IssuerUrl
	clientID := clientConfig.Identity.CLI.ClientId

	if err := util.RevokeOfflineToken(tok, issuerUrlStr, clientID); err != nil {
		return fmt.Errorf("couldn't revoke token: %v", err)
	}

	cmd.Printf("Token revoked\n")

	return nil
}

func init() {
	offlineTokenCmd.AddCommand(offlineTokenRevokeCmd)

	offlineTokenRevokeCmd.Flags().StringP("file", "f", "offline.token", "The file that contains the offline token")
	offlineTokenRevokeCmd.Flags().StringP("token", "t", "",
		"The offline token to revoke. Also settable through the MINDER_OFFLINE_TOKEN environment variable.")

	offlineTokenRevokeCmd.MarkFlagsMutuallyExclusive("file", "token")

	if err := viper.BindPFlag("file", offlineTokenRevokeCmd.Flag("file")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("token", offlineTokenRevokeCmd.Flag("token")); err != nil {
		panic(err)
	}

	if err := viper.BindEnv("token", "MINDER_OFFLINE_TOKEN"); err != nil {
		panic(err)
	}
}
