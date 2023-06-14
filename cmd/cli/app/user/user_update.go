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
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
)

// User_updateCmd is the command for creating an user
var User_updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a user within a mediator control plane",
	Long:  `The medctl user update subcommand allows to modify the details of an user.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// create the user via GRPC
		password := util.GetConfigValue("password", "password", cmd, nil)
		password_confirmation := util.GetConfigValue("password_confirmation", "password_confirmation", cmd, nil)

		conn, err := util.GetGrpcConnection(cmd)
		if err != nil {
			util.ExitNicelyOnError(err, "Error getting grpc connection")
		}
		defer conn.Close()

		client := pb.NewUserServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err = client.UpdatePassword(ctx, &pb.UpdatePasswordRequest{
			Password:             password.(string),
			PasswordConfirmation: password_confirmation.(string),
		})

		if err != nil {
			util.ExitNicelyOnError(err, "Error updating user password")
		}
		cmd.Println("Password updated successfully, please authenticate again with your new credentials.")
	},
}

func init() {
	UserCmd.AddCommand(User_updateCmd)
	User_updateCmd.Flags().StringP("password", "p", "", "Password for the user")
	User_updateCmd.Flags().StringP("password_confirmation", "c", "", "Password confirmation for the user")
	if err := User_updateCmd.MarkFlagRequired("password"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
	if err := User_updateCmd.MarkFlagRequired("password_confirmation"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
}
