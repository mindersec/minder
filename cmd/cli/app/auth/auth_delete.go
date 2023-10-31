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
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/util"
	"github.com/stacklok/mediator/internal/util/cli"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

// auth_deleteCmd represents the account deletion command
var auth_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Permanently delete account",
	Long:  `Permanently delete account. All associated user data will be permanently removed.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Are you sure you want to permanently delete your account? (yes/no): ")
		response, _ := reader.ReadString('\n')

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "yes" && response != "y" {
			cli.PrintCmd(cmd, cli.Header.Render("Delete account operation cancelled."))
			return
		}

		ctx, cancel := util.GetAppContext()
		defer cancel()

		conn, err := util.GrpcForCommand(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()
		client := pb.NewUserServiceClient(conn)

		_, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{})
		util.ExitNicelyOnError(err, "Error registering user")

		// This step is added to avoid confusing the users by seeing their credentials locally, however it is not
		// directly related to user deletion because the token will expire after 5 minutes and cannot be refreshed
		err = util.RemoveCredentials()
		if err != nil {
			cli.PrintCmd(cmd, cli.WarningBanner.Render("Failed to remove locally stored credentials."))
		}
		cli.PrintCmd(cmd, cli.SuccessBanner.Render("Successfully deleted account. It may take up to 48 hours for "+
			"all data to be removed."))
	},
}

func init() {
	AuthCmd.AddCommand(auth_deleteCmd)
}
