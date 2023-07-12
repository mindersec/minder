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

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// Auth_revokeCmd represents the auth revoke command
var Auth_revokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke access tokens",
	Long:  `It can revoke access tokens for one user or for all.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// check if we need to revoke all tokens or the user one
		all := util.GetConfigValue("all", "all", cmd, false).(bool)
		user := viper.GetInt32("user-id")

		if all && user != 0 {
			fmt.Fprintf(os.Stderr, "Error: you can't use --all and --user-id together\n")
			os.Exit(1)
		}

		if !all && user == 0 {
			fmt.Fprintf(os.Stderr, "Error: you must use either --all or --user-id\n")
			os.Exit(1)
		}

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		util.ExitNicelyOnError(err, "Error getting grpc connection")

		ctx, cancel := util.GetAppContext()
		defer cancel()
		client := pb.NewAuthServiceClient(conn)
		if all {
			_, err := client.RevokeTokens(ctx, &pb.RevokeTokensRequest{})
			util.ExitNicelyOnError(err, "Error revoking tokens")
			cmd.Println("Revoked all tokens")
		} else {
			_, err := client.RevokeUserToken(ctx, &pb.RevokeUserTokenRequest{UserId: user})
			util.ExitNicelyOnError(err, "Error revoking tokens")
			cmd.Println("Revoked token for user", user)
		}
	},
}

func init() {
	AuthCmd.AddCommand(Auth_revokeCmd)
	Auth_revokeCmd.Flags().BoolP("all", "a", false, "Revoke all tokens")
	Auth_revokeCmd.Flags().Int32P("user-id", "u", 0, "User ID to revoke tokens")

}
