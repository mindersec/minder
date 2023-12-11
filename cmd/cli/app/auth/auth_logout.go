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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package auth

import (
	"net/url"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/constants"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
)

// auth_logoutCmd represents the logout command
var auth_logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from minder control plane.",
	Long:  `Logout from minder control plane. Credentials will be removed from $XDG_CONFIG_HOME/minder/credentials.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := util.RemoveCredentials(); err != nil {
			return cli.MessageAndError(cmd, "Error removing credentials", err)
		}

		issuerUrlStr := util.GetConfigValue(viper.GetViper(), "identity.cli.issuer_url", "identity-url", cmd,
			constants.IdentitySeverURL).(string)

		parsedURL, err := url.Parse(issuerUrlStr)
		if err != nil {
			return cli.MessageAndError(cmd, "Error parsing issuer URL", err)
		}

		logoutUrl := parsedURL.JoinPath("realms/stacklok/protocol/openid-connect/logout")
		cli.PrintCmd(cmd, cli.SuccessBanner.Render("You have successfully logged out of the CLI."))
		cli.PrintCmd(cmd, "If you would like to log out of the browser, you can visit %s", logoutUrl.String())
		return nil
	},
}

func init() {
	AuthCmd.AddCommand(auth_logoutCmd)

}
