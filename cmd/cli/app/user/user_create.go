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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
)

// askForConfirmation asks the user for confirmation and returns true if confirmed, false otherwise.
func askForConfirmation(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s (y/n): ", prompt)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "y" || response == "yes" {
		return true
	}

	return false
}

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
		role := viper.GetInt("role-id")
		email := util.GetConfigValue("email", "email", cmd, nil)
		username := util.GetConfigValue("username", "username", cmd, nil)
		password := util.GetConfigValue("password", "password", cmd, nil)
		firstName := util.GetConfigValue("firstname", "firstname", cmd, nil)
		lastName := util.GetConfigValue("lastname", "lastname", cmd, nil)
		isProtected := util.GetConfigValue("is-protected", "is-protected", cmd, false).(bool)
		force := util.GetConfigValue("force", "force", cmd, false).(bool)
		needsPasswordChange := util.GetConfigValue("needs-password-change", "needs-password-change", cmd, true).(bool)

		if username == nil {
			fmt.Fprintf(os.Stderr, "Error: username is required\n")
			os.Exit(1)
		}

		// if role is null, we need to ask if they want to create a single user
		if role == 0 && !force {
			confirmed := askForConfirmation("You didn't specify a role id. Do you want to create an user without any organization?")
			if !confirmed {
				fmt.Println("User creation cancelled")
				os.Exit(0)
			}
		}

		conn, err := util.GetGrpcConnection(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting grpc connection: %s\n", err)
			os.Exit(1)
		}
		defer conn.Close()

		client := pb.NewUserServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		// now create the default fields for the user if needed
		if role == 0 {
			client := pb.NewOrganizationServiceClient(conn)
			resp, err := client.CreateOrganization(ctx, &pb.CreateOrganizationRequest{
				Name:    username.(string) + "-org",
				Company: username.(string) + " - Self enrolled",
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating organization: %s\n", err)
				os.Exit(1)
			}

			clientg := pb.NewGroupServiceClient(conn)
			respg, err := clientg.CreateGroup(ctx, &pb.CreateGroupRequest{OrganizationId: resp.Id,
				Name: username.(string) + "-group"})

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating group: %s\n", err)
				os.Exit(1)
			}

			clientr := pb.NewRoleServiceClient(conn)
			isAdmin := true
			respr, err := clientr.CreateRole(ctx, &pb.CreateRoleRequest{GroupId: respg.GroupId,
				Name: username.(string) + "-role", IsAdmin: &isAdmin})

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating role: %s\n", err)
				os.Exit(1)
			}

			// now assign the role
			role = int(respr.Id)
		}

		protectedPtr := &isProtected
		var emailPtr *string
		if email == nil || email == "" {
			emailPtr = nil
		} else {
			temp := email.(string)
			emailPtr = &temp
		}
		var passwordPtr *string
		if password == nil || password == "" {
			passwordPtr = nil
		} else {
			temp := password.(string)
			passwordPtr = &temp
		}
		var firstNamePtr *string
		if firstName == nil || firstName == "" {
			firstNamePtr = nil
		} else {
			temp := firstName.(string)
			firstNamePtr = &temp
		}
		var lastNamePtr *string
		if lastName == nil || lastName == "" {
			lastNamePtr = nil
		} else {
			temp := lastName.(string)
			lastNamePtr = &temp
		}
		needsPasswordChangePtr := &needsPasswordChange

		resp, err := client.CreateUser(ctx, &pb.CreateUserRequest{
			RoleId:              int32(role),
			Email:               emailPtr,
			Username:            username.(string),
			Password:            passwordPtr,
			FirstName:           firstNamePtr,
			LastName:            lastNamePtr,
			IsProtected:         protectedPtr,
			NeedsPasswordChange: needsPasswordChangePtr,
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
	User_createCmd.Flags().Int32P("role-id", "r", 0, "Role ID. If empty, will create a single user")
	User_createCmd.Flags().BoolP("force", "s", false, "Skip confirmation")
	User_createCmd.Flags().BoolP("needs-password-change", "c", true, "Does the user need to change their password")
	if err := User_createCmd.MarkFlagRequired("username"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
}
