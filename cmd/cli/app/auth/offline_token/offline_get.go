//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package offline_token provides the auth offline_token command for the minder CLI.\
package offline_token

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/cmd/cli/app/auth"
	"github.com/stacklok/minder/internal/config"
	clientconfig "github.com/stacklok/minder/internal/config/client"
	"github.com/stacklok/minder/internal/util/cli"
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

	RunE: cli.GRPCClientWrapRunE(offlineGetCommand),
}

// offlineGetCommand is the offline-token get subcommand
func offlineGetCommand(ctx context.Context, cmd *cobra.Command, _ []string, _ *grpc.ClientConn) error {
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
	token, err := auth.Login(ctx, cmd, clientConfig, []string{"offline_access"}, skipBrowser)
	if err != nil {
		return err
	}

	// write the token to the file
	if err := os.WriteFile(f, []byte(token.RefreshToken), 0600); err != nil {
		return fmt.Errorf("error writing offline token to file: %w", err)
	}

	cmd.Printf("Offline token written to %s\n", f)

	return nil
}

func init() {
	offlineTokenCmd.AddCommand(offlineTokenGetCmd)

	offlineTokenGetCmd.Flags().StringP("file", "f", "offline.token", "The file to write the offline token to")

	if err := viper.BindPFlag("file", offlineTokenGetCmd.Flag("file")); err != nil {
		panic(err)
	}

	offlineTokenGetCmd.Flags().BoolP("skip-browser", "", false, "Skip opening the browser for OAuth flow")
	if err := viper.BindPFlag("offline.get.skip-browser", offlineTokenGetCmd.Flag("skip-browser")); err != nil {
		panic(err)
	}
}
