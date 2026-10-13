// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/config"
	clientconfig "github.com/mindersec/minder/pkg/config/client"
)

// whoamiCmd represents the whoami command
var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "whoami for current user",
	Long:  `whoami gets information about the current user from the minder server`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}

		format, err := cmd.Flags().GetString("output")
		if err != nil {
			return cli.MessageAndError("Error parsing output flag", err)
		}

		// Ensure the output format is supported
		if !app.IsOutputFormatSupported(format) {
			return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
		}

		return nil
	},
	RunE: whoamiCommand,
}

// whoamiCommand is the whoami subcommand
func whoamiCommand(cmd *cobra.Command, _ []string) error {
	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, closeConn, err := cli.GetCLIClient(cmd, minderv1.NewUserServiceClient)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer closeConn()

	userInfo, err := client.GetUser(cmd.Context(), &minderv1.GetUserRequest{})
	if err != nil {
		return cli.MessageAndError("Error getting information for user", err)
	}

	cfg, err := config.ReadConfigFromViper[clientconfig.Config](viper.GetViper())
	if err != nil || cfg == nil {
		return cli.MessageAndError("Error reading config", err)
	}

	renderUserInfoWhoami(cfg.GRPCClientConfig.GetGRPCAddress(), cmd.OutOrStdout(), viper.GetString("output"), userInfo)
	return nil
}

func init() {
	AuthCmd.AddCommand(whoamiCmd)

	whoamiCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
