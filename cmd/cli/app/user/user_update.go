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

package user

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

// User_updateCmd is the command for creating an user
var User_updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a user within a mediator control plane",
	Long:  `The medic user update subcommand allows to modify the details of an user.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// create the user via GRPC
		password := util.GetConfigValue("password", "password", cmd, "").(string)
		password_confirmation := util.GetConfigValue("password_confirmation", "password_confirmation", cmd, "").(string)

		email := util.GetConfigValue("email", "email", cmd, "").(string)
		first_name := util.GetConfigValue("firstname", "firstname", cmd, "").(string)
		last_name := util.GetConfigValue("lastname", "lastname", cmd, "").(string)

		// user needs to update at least one of those fields
		if password == "" && email == "" && first_name == "" && last_name == "" {
			fmt.Fprint(os.Stderr, "Error: Must provide at least one of the following: password, email, first_name, last_name")
			os.Exit(1)
		}
		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewUserServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// if password is set, password_confirmation must be set
		if (password != "" && password_confirmation == "") || (password == "" && password_confirmation != "") {
			fmt.Fprint(os.Stderr, "Error: Must provide both password and password_confirmation")
			os.Exit(1)
		}

		// need to choose between password or profile fields
		if password != "" && (email != "" || first_name != "" || last_name != "") {
			fmt.Fprint(os.Stderr, "Error: Cannot update both password and profile fields")
			os.Exit(1)
		}

		// check if we need to update password
		if password != "" && password_confirmation != "" {
			_, err = client.UpdatePassword(ctx, &pb.UpdatePasswordRequest{
				Password:             password,
				PasswordConfirmation: password_confirmation,
			})
			util.ExitNicelyOnError(err, "Error updating user password")
			cmd.Println("Password updated successfully, please authenticate again with your new credentials.")
		} else {
			// need to update profile
			req := pb.UpdateProfileRequest{}
			if email != "" {
				req.Email = &email
			}
			if first_name != "" {
				req.FirstName = &first_name
			}
			if last_name != "" {
				req.LastName = &last_name
			}
			_, err = client.UpdateProfile(ctx, &req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error updating user profile: %s\n", err)
				os.Exit(1)
			}
			cmd.Println("Profile updated successfully.")
		}

	},
}

func init() {
	UserCmd.AddCommand(User_updateCmd)
	User_updateCmd.Flags().StringP("password", "p", "", "Password")
	User_updateCmd.Flags().StringP("password_confirmation", "c", "", "Password confirmation")
	User_updateCmd.Flags().StringP("email", "e", "", "Email for your profile")
	User_updateCmd.Flags().StringP("firstname", "f", "", "First name for your profile")
	User_updateCmd.Flags().StringP("lastname", "l", "", "Last name for your profile")
}
