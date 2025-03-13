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

	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/pkg/config"
	clientconfig "github.com/mindersec/minder/pkg/config/client"
)

// offlineTokenUseCmd represents the offline-token use command
var offlineTokenUseCmd = &cobra.Command{
	Use:   "use",
	Short: "Use an offline token",
	Long: `The minder auth offline-token use command project lets you install and use an offline token
for the minder control plane.

Offline tokens are used to authenticate to the minder control plane without
requiring the user's presence. This is useful for long-running processes
that need to authenticate to the control plane.`,
	RunE: cli.GRPCClientWrapRunE(offlineUseCommand),
}

// offlineUseCommand is the offline-token use subcommand
func offlineUseCommand(_ context.Context, cmd *cobra.Command, _ []string, _ *grpc.ClientConn) error {
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

	grpcCfg := clientConfig.GRPCClientConfig
	opts := []grpc.DialOption{grpcCfg.TransportCredentialsOption()}
	issuerUrlStr := clientConfig.Identity.CLI.IssuerUrl
	clientID := clientConfig.Identity.CLI.ClientId
	realm := clientConfig.Identity.CLI.Realm

	realmUrl, err := cli.GetRealmUrl(grpcCfg.GetGRPCAddress(), opts, issuerUrlStr, realm)
	if err != nil {
		return fmt.Errorf("couldn't get realm URL: %v", err)
	}

	creds, err := cli.RefreshCredentials(tok, realmUrl, clientID)
	if err != nil {
		return fmt.Errorf("couldn't fetch credentials: %v", err)
	}

	// save credentials
	filePath, err := cli.SaveCredentials(cli.OpenIdCredentials{
		AccessToken:          creds.AccessToken,
		RefreshToken:         creds.RefreshToken,
		AccessTokenExpiresAt: creds.AccessTokenExpiresAt,
	})
	if err != nil {
		cmd.PrintErrf("couldn't save credentials: %s\n", err)
	}

	cmd.Printf("Your access credentials have been saved to %s\n", filePath)

	return nil
}

func init() {
	offlineTokenCmd.AddCommand(offlineTokenUseCmd)

	offlineTokenUseCmd.Flags().StringP("file", "f", "offline.token", "The file that contains the offline token")
	offlineTokenUseCmd.Flags().StringP("token", "t", "",
		"The offline token to use. Also settable through the MINDER_OFFLINE_TOKEN environment variable.")

	offlineTokenUseCmd.MarkFlagsMutuallyExclusive("file", "token")

	if err := viper.BindPFlag("file", offlineTokenUseCmd.Flag("file")); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag("token", offlineTokenUseCmd.Flag("token")); err != nil {
		panic(err)
	}

	if err := viper.BindEnv("token", "MINDER_OFFLINE_TOKEN"); err != nil {
		panic(err)
	}
}
