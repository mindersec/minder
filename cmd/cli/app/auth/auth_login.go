// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/config"
	clientconfig "github.com/mindersec/minder/pkg/config/client"
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Minder",
	Long: `The login command allows for logging in to Minder. Upon successful login, credentials will be saved to
$XDG_CONFIG_HOME/minder/ based on the hostname and port of the server.`,
	RunE: LoginCommand,
}

// LoginCommand is the login subcommand
func LoginCommand(cmd *cobra.Command, _ []string) error {
	clientConfig, err := config.ReadConfigFromViper[clientconfig.Config](viper.GetViper())
	if err != nil {
		return cli.MessageAndError("Unable to read config", err)
	}
	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	ctx := cmd.Context()

	filePath, err := cli.LoginAndSaveCreds(ctx, cmd, clientConfig)
	if err != nil {
		return cli.MessageAndError("Error ensuring credentials", err)
	}

	// Get a connection to the GRPC server after we have the credentials
	conn, err := cli.GrpcForCommand(cmd, viper.GetViper())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := minderv1.NewUserServiceClient(conn)

	// check if the user already exists on the server
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

	// hidden flags
	loginCmd.Flags().BoolP("skip-browser", "", false, "Skip opening the browser for OAuth flow")
	// Bind flags
	if err := viper.BindPFlag("login.skip-browser", loginCmd.Flags().Lookup("skip-browser")); err != nil {
		panic(err)
	}
}
