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
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/stacklok/mediator/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"

	"github.com/spf13/viper"
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
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting grpc connection: %s\n", err)
			os.Exit(1)
		}
		defer conn.Close()

		client := pb.NewLogInServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// call login endpoint
		resp, err := client.LogIn(ctx, &pb.LogInRequest{Username: username, Password: password})
		if err != nil {
			ret := status.Convert(err)
			ns := util.GetNiceStatus(ret.Code())
			fmt.Fprintf(os.Stderr, "Error logging in: %s\n", ns)
			os.Exit(1)
		}
		if resp.Status.Code != int32(codes.OK) {
			util.GetNiceStatus(codes.Code(resp.Status.Code))
			fmt.Fprintf(os.Stderr, "Error logging in: %s\n", resp.Status)
			os.Exit(1)
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

		fmt.Printf("Credentials saved to %s\n", filePath)

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
