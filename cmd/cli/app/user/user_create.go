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

// User_createCmd is the command for creating an user
var User_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a user within a mediator control plane",
	Long: `The medctl user create subcommand lets you create new users for a role
within a mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// create the user via GRPC
		role := util.GetConfigValue("role-id", "role-id", cmd, int32(0)).(int32)
		email := util.GetConfigValue("email", "email", cmd, nil)
		username := util.GetConfigValue("username", "username", cmd, nil)
		password := util.GetConfigValue("password", "password", cmd, nil)
		firstName := util.GetConfigValue("firstname", "firstname", cmd, nil)
		lastName := util.GetConfigValue("lastname", "lastname", cmd, nil)
		isProtected := util.GetConfigValue("is-protected", "is-protected", cmd, false).(bool)

		if username == nil {
			fmt.Fprintf(os.Stderr, "Error: username is required\n")
			os.Exit(1)
		}

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

		client := pb.NewUserServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		protectedPtr := &isProtected

		var emailPtr *string
		if email == nil {
			emailPtr = nil
		} else {
			emailPtr = email.(*string)
		}
		var passwordPtr *string
		if password == nil {
			passwordPtr = nil
		} else {
			passwordPtr = password.(*string)
		}
		var firstNamePtr *string
		if firstName == nil {
			firstNamePtr = nil
		} else {
			firstNamePtr = password.(*string)
		}
		var lastNamePtr *string
		if lastName == nil {
			lastNamePtr = nil
		} else {
			lastNamePtr = lastName.(*string)
		}

		resp, err := client.CreateUser(ctx, &pb.CreateUserRequest{
			RoleId:      role,
			Email:       emailPtr,
			Username:    username.(string),
			Password:    passwordPtr,
			FirstName:   firstNamePtr,
			LastName:    lastNamePtr,
			IsProtected: protectedPtr,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating user: %s\n", err)
			os.Exit(1)
		}

		user, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			cmd.Println("Created user: ", resp.Username)
		} else {
			cmd.Println("Created user:", string(user))
		}
	},
}

func init() {
	UserCmd.AddCommand(User_createCmd)
	User_createCmd.Flags().StringP("username", "u", "", "Username")
	User_createCmd.Flags().StringP("email", "e", "", "E-mail for the user")
	User_createCmd.Flags().StringP("password", "p", "", "Password for the user")
	User_createCmd.Flags().StringP("firstname", "f", "", "User's first name")
	User_createCmd.Flags().StringP("lastname", "l", "", "User's last name")
	User_createCmd.Flags().BoolP("is-protected", "i", false, "Is the user protected")
	User_createCmd.Flags().Int32P("role-id", "r", 0, "Role ID")
	if err := User_createCmd.MarkFlagRequired("username"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
	if err := User_createCmd.MarkFlagRequired("role-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
}
