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
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
	"google.golang.org/grpc/codes"
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
		conn, err := util.GetGrpcConnection(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting grpc connection: %s\n", err)
			os.Exit(1)
		}
		defer conn.Close()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting grpc connection: %s\n", err)
			os.Exit(1)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		client := pb.NewAuthServiceClient(conn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error revoking tokens: %s\n", err)
			os.Exit(1)
		}
		resp, err := client.RefreshToken(ctx, &pb.RefreshTokenRequest{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error revoking tokens: %s\n", err)
			os.Exit(1)
		}
		if resp.Status.Code != int32(codes.OK) {
			fmt.Fprintf(os.Stderr, "Error refreshing token: %s\n", resp.Status)
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
	AuthCmd.AddCommand(Auth_refreshCmd)
}
