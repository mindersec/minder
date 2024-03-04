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
	"context"
	_ "embed"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/config"
	clientconfig "github.com/stacklok/minder/internal/config/client"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

//go:embed html/login_success.html
var loginSuccessHtml []byte

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Minder",
	Long: `The login command allows for logging in to Minder. Upon successful login, credentials will be saved to
$XDG_CONFIG_HOME/minder/credentials.json`,
	RunE: LoginCommand,
}

// LoginCommand is the login subcommand
func LoginCommand(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	clientConfig, err := config.ReadConfigFromViper[clientconfig.Config](viper.GetViper())
	if err != nil {
		return cli.MessageAndError("Unable to read config", err)
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// wait for the token to be received
	token, err := login(ctx, cmd, clientConfig, nil)
	if err != nil {
		return err
	}

	// save credentials
	filePath, err := util.SaveCredentials(util.OpenIdCredentials{
		AccessToken:          token.AccessToken,
		RefreshToken:         token.RefreshToken,
		AccessTokenExpiresAt: token.Expiry,
	})
	if err != nil {
		cmd.PrintErrf("couldn't save credentials: %s\n", err)
	}

	conn, err := cli.GrpcForCommand(viper.GetViper())
	if err != nil {
		return cli.MessageAndError("Error getting grpc connection", err)
	}
	defer conn.Close()
	client := minderv1.NewUserServiceClient(conn)

	// check if the user already exists in the local database
	registered, userInfo, err := userRegistered(ctx, client)
	if err != nil {
		return cli.MessageAndError("Error checking if user exists", err)
	}

	if !registered {
		cmd.Println("First login, registering user...")
		newUser, err := client.CreateUser(ctx, &minderv1.CreateUserRequest{})
		if err != nil {
			return cli.MessageAndError("Error registering user", err)
		}

		cmd.Println(cli.SuccessBanner.Render(
			"You have been successfully registered. Welcome!"))
		cmd.Println(cli.WarningBanner.Render(
			"Minder is currently under active development and considered experimental, " +
				" we therefore provide no data retention or service stability guarantees.",
		))
		cmd.Println(cli.Header.Render("Here are your details:"))

		renderNewUser(conn.Target(), newUser)
	} else {
		cmd.Println(cli.SuccessBanner.Render(
			"You have successfully logged in."))

		cmd.Println(cli.Header.Render("Here are your details:"))
		renderUserInfo(conn.Target(), userInfo)
	}

	cmd.Printf("Your access credentials have been saved to %s\n", filePath)
	return nil
}

func init() {
	AuthCmd.AddCommand(loginCmd)
}
