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
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// deleteCmd represents the account deletion command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Permanently delete account",
	Long:  `Permanently delete account. All associated user data will be permanently removed.`,
	RunE:  cli.GRPCClientWrapRunE(deleteCommand),
}

// deleteCommand is the account deletion subcommand
func deleteCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewUserServiceClient(conn)
	yesFlag := viper.GetBool("yes-delete-my-account")

	// Ensure the user already exists in the local database
	_, _, err := userRegistered(ctx, client)
	if err != nil {
		return cli.MessageAndError("Error checking if user exists", err)
	}

	// We read name and email from the JWT.  We don't need to validate it here.
	creds, err := util.LoadCredentials()
	if err != nil {
		return cli.MessageAndError("Error loading credentials from file", err)
	}
	accessToken, err := jwt.ParseString(creds.AccessToken, jwt.WithVerify(false), jwt.WithToken(openid.New()))
	if err != nil {
		return cli.MessageAndError("Error parsing token", err)
	}
	token, ok := accessToken.(openid.Token)
	if !ok {
		return cli.MessageAndError("Error parsing token", fmt.Errorf("provided token was not an OpenID token"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// Confirm user wants to delete their account
	if !yesFlag {
		yes := cli.PrintYesNoPrompt(cmd,
			fmt.Sprintf(
				"You are about to permanently delete your account. \n\nName: %s\nEmail: %s",
				fmt.Sprintf("%s %s", token.GivenName(), token.FamilyName()),
				token.Email(),
			),
			"Are you sure?",
			"Delete account operation cancelled.",
			false)
		if !yes {
			return nil
		}
	}

	_, err = client.DeleteUser(ctx, &minderv1.DeleteUserRequest{})
	if err != nil {
		return cli.MessageAndError("Error deleting user", err)
	}

	// This step is added to avoid confusing the users by seeing their credentials locally, however it is not
	// directly related to user deletion because the token will expire after 5 minutes and cannot be refreshed
	err = util.RemoveCredentials()
	if err != nil {
		cmd.Println(cli.WarningBanner.Render("Failed to remove locally stored credentials."))
	}
	cmd.Println(cli.SuccessBanner.Render("Successfully deleted account. It may take up to 48 hours for " +
		"all data to be removed."))
	return nil
}

func init() {
	AuthCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().Bool("yes-delete-my-account", false, "Bypass yes/no prompt when deleting the account")
}
