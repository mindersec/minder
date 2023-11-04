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
	"fmt"
	"net/url"
	"os"

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
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := util.RemoveCredentials()
		util.ExitNicelyOnError(err, "Error removing credentials")

		issuerUrlStr := util.GetConfigValue(viper.GetViper(), "identity.cli.issuer_url", "identity-url", cmd,
			constants.IdentitySeverURL).(string)
		realm := util.GetConfigValue(viper.GetViper(), "identity.cli.realm", "identity-realm", cmd, "stacklok").(string)

		parsedURL, err := url.Parse(issuerUrlStr)
		util.ExitNicelyOnError(err, "Error parsing issuer URL")

		logoutUrl := parsedURL.JoinPath("realms", realm, "protocol/openid-connect/logout")
		cli.PrintCmd(cmd, cli.SuccessBanner.Render("You have successfully logged out of the CLI."))
		cli.PrintCmd(cmd, "If you would like to log out of the browser, you can visit %s", logoutUrl.String())
	},
}

func init() {
	AuthCmd.AddCommand(auth_logoutCmd)

}
