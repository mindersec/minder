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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
)

var user_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a user within a mediator control plane",
	Long: `The medctl user create subcommand lets you create new users for a role
within a mediator control plane.`,
	Run: func(cmd *cobra.Command, args []string) {
		// create the user via GRPC
		role := util.GetConfigValue("role-id", "role-id", cmd, int32(0)).(int32)
		email := util.GetConfigValue("email", "email", cmd, nil)
		username := util.GetConfigValue("username", "username", cmd, nil)
		password := util.GetConfigValue("password", "password", cmd, nil)
		firstName := util.GetConfigValue("firstname", "firstname", cmd, nil).(string)
		lastName := util.GetConfigValue("lastname", "lastname", cmd, nil).(string)
		isProtected := util.GetConfigValue("is-protected", "is-protected", cmd, false).(bool)

		conn, err := util.GetGrpcConnection(cmd)
		defer conn.Close()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting grpc connection: %s\n", err)
			os.Exit(1)
		}

		client := pb.NewUserServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		firstNamePtr := &firstName
		lastNamePtr := &lastName
		protectedPtr := &isProtected
		resp, err := client.CreateUser(ctx, &pb.CreateUserRequest{
			RoleId:      role,
			Email:       email.(string),
			Username:    username.(string),
			Password:    password.(string),
			FirstName:   firstNamePtr,
			LastName:    lastNamePtr,
			IsProtected: protectedPtr,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating user: %s\n", err)
			os.Exit(1)
		}

		user, err := json.Marshal(resp)
		if err != nil {
			cmd.Println("Created user: ", resp.Username)
		} else {
			cmd.Println("Created user:", string(user))
		}
	},
}

func init() {
	UserCmd.AddCommand(user_createCmd)
	user_createCmd.PersistentFlags().StringP("username", "u", "", "Username")
	user_createCmd.PersistentFlags().StringP("email", "e", "", "E-mail for the user")
	user_createCmd.PersistentFlags().StringP("password", "p", "", "Password for the user")
	user_createCmd.PersistentFlags().StringP("firstname", "f", "", "User's first name")
	user_createCmd.PersistentFlags().StringP("lastname", "l", "", "User's last name")
	user_createCmd.PersistentFlags().BoolP("is-protected", "i", false, "Is the user protected")
	user_createCmd.PersistentFlags().Int32P("role-id", "r", 0, "Role ID")
	if err := user_createCmd.MarkPersistentFlagRequired("username"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
	if err := user_createCmd.MarkPersistentFlagRequired("email"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
	if err := user_createCmd.MarkPersistentFlagRequired("password"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
	if err := user_createCmd.MarkPersistentFlagRequired("role-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

	if err := viper.BindPFlags(user_createCmd.PersistentFlags()); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		os.Exit(1)
	}

	if err := viper.BindPFlags(user_createCmd.PersistentFlags()); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		os.Exit(1)
	}
}
