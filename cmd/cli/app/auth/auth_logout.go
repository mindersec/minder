// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"net/url"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/pkg/config"
	clientconfig "github.com/mindersec/minder/pkg/config/client"
)

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from minder control plane.",
	Long:  `Logout from minder control plane. Credentials will be removed from $XDG_CONFIG_HOME/minder/credentials.json`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if err := util.RemoveCredentials(); err != nil {
			return cli.MessageAndError("Error removing credentials", err)
		}

		clientConfig, err := config.ReadConfigFromViper[clientconfig.Config](viper.GetViper())
		if err != nil {
			return cli.MessageAndError("Unable to read config", err)
		}
		issuerUrlStr := clientConfig.Identity.CLI.IssuerUrl

		parsedURL, err := url.Parse(issuerUrlStr)
		if err != nil {
			return cli.MessageAndError("Error parsing issuer URL", err)
		}

		// No longer print usage on returned error, since we've parsed our inputs
		// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
		cmd.SilenceUsage = true

		logoutUrl := parsedURL.JoinPath("realms/stacklok/protocol/openid-connect/logout")
		cmd.Println(cli.SuccessBanner.Render("You have successfully logged out of the CLI."))
		cmd.Printf("If you would like to log out of the browser, you can visit %s\n", logoutUrl.String())
		return nil
	},
}

func init() {
	AuthCmd.AddCommand(logoutCmd)
}
