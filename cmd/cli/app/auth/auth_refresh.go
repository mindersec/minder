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
	"github.com/spf13/viper"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// Auth_refreshCmd represents the auth refresh command
var Auth_refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh credentials",
	Long:  `It refreshes credentials for one user`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// load old credentials
		oldCreds, err := util.LoadCredentials()
		util.ExitNicelyOnError(err, "Error loading credentials")

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		client := pb.NewAuthServiceClient(conn)
		util.ExitNicelyOnError(err, "Error getting grpc connection")

		resp, err := client.RefreshToken(ctx, &pb.RefreshTokenRequest{})
		util.ExitNicelyOnError(err, "Error refreshing token")

		// marshal the credentials to json. Only refresh access token
		creds := util.Credentials{
			AccessToken:           resp.AccessToken,
			RefreshToken:          oldCreds.RefreshToken,
			AccessTokenExpiresIn:  int(resp.AccessTokenExpiresIn),
			RefreshTokenExpiresIn: oldCreds.RefreshTokenExpiresIn,
		}

		// save credentials
		filePath, err := util.SaveCredentials(creds)
		util.ExitNicelyOnError(err, "Error saving credentials")

		fmt.Printf("Credentials saved to %s\n", filePath)
	},
}

func init() {
	AuthCmd.AddCommand(Auth_refreshCmd)
}
