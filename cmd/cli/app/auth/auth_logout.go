//
// Copyright 2023 Stacklok, Inc.
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

package auth

import (
	"net/url"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/config"
	clientconfig "github.com/stacklok/minder/internal/config/client"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
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
