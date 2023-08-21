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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc/status"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// auth_loginCmd represents the login command
var auth_loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to a mediator control plane.",
	Long: `Login to a mediator control plane. Upon successful login, credentials
will be saved to $XDG_CONFIG_HOME/mediator/credentials.json`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		username := util.GetConfigValue("username", "username", cmd, "").(string)
		password := util.GetConfigValue("password", "password", cmd, "").(string)

		conn, err := util.GetGrpcConnection(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewAuthServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		// call login endpoint
		resp, err := client.LogIn(ctx, &pb.LogInRequest{Username: username, Password: password})
		if err != nil {
			ret := status.Convert(err)
			fmt.Fprintf(os.Stderr, "Error logging in: Code: %d\nName: %s\nDetails: %s\n", ret.Code(), ret.Code().String(), ret.Message())

			os.Exit(int(ret.Code()))
		}

		// marshal the credentials to json
		creds := util.Credentials{
			AccessToken:           resp.AccessToken,
			RefreshToken:          resp.RefreshToken,
			AccessTokenExpiresIn:  int(resp.AccessTokenExpiresIn),
			RefreshTokenExpiresIn: int(resp.RefreshTokenExpiresIn),
		}

		// save credentials
		filePath, err := util.SaveCredentials(creds)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf("You have been successfully logged in. Your access credentials saved to %s\n"+
			"Remember that if that's your first login, you will need to update your password "+
			"using the user update --password command", filePath)

	},
}

func init() {
	AuthCmd.AddCommand(auth_loginCmd)
	auth_loginCmd.Flags().StringP("username", "u", "", "Username to use for authentication")
	auth_loginCmd.Flags().StringP("password", "p", "", "Password to use for authentication")

	if err := auth_loginCmd.MarkFlagRequired("username"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
	if err := auth_loginCmd.MarkFlagRequired("password"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}

}
