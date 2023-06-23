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
	"path/filepath"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stacklok/mediator/pkg/util"
)

// auth_logoutCmd represents the logout command
var auth_logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from mediator control plane.",
	Long:  `Logout from mediator control plane. Credentials will be removed from $XDG_CONFIG_HOME/mediator/credentials.json`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewAuthServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		_, err = client.LogOut(ctx, &pb.LogOutRequest{})
		util.ExitNicelyOnError(err, "Error logging out")

		// remove credentials file for extra security
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")

		// just delete token from credentials file
		if xdgConfigHome == "" {
			homeDir, err := os.UserHomeDir()
			util.ExitNicelyOnError(err, "Error getting home directory")
			xdgConfigHome = filepath.Join(homeDir, ".config")
		}

		filePath := filepath.Join(xdgConfigHome, "mediator", "credentials.json")
		err = os.Remove(filePath)
		util.ExitNicelyOnError(err, "Error removing credentials file")

		fmt.Println("User logged out.")

	},
}

func init() {
	AuthCmd.AddCommand(auth_logoutCmd)

}
