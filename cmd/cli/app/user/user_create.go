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
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
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
	Long: `The medic user create subcommand lets you create new users for a role
within a mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// create the user via GRPC
		email := util.GetConfigValue("email", "email", cmd, nil)
		username := util.GetConfigValue("username", "username", cmd, nil)
		password := util.GetConfigValue("password", "password", cmd, nil)
		firstName := util.GetConfigValue("firstname", "firstname", cmd, nil)
		lastName := util.GetConfigValue("lastname", "lastname", cmd, nil)
		isProtected := util.GetConfigValue("is-protected", "is-protected", cmd, false).(bool)
		force := util.GetConfigValue("force", "force", cmd, false).(bool)
		needsPasswordChange := util.GetConfigValue("needs-password-change", "needs-password-change", cmd, true).(bool)
		org := util.GetConfigValue("org-id", "org-id", cmd, nil).(int)

		// convert string of roles to array
		var roles []int32
		roleField := util.GetConfigValue("roles", "roles", cmd, "").(string)
		if roleField != "" {
			roleStr := strings.Split(roleField, ",")
			for _, str := range roleStr {
				if str != "" {
					number, _ := strconv.ParseInt(str, 10, 32)
					roles = append(roles, int32(number))
				}
			}
		}

		// convert string of groups to array
		groupField := util.GetConfigValue("groups", "groups", cmd, "").(string)
		var groups []int32
		if groupField != "" {
			groupStr := strings.Split(groupField, ",")
			for _, str := range groupStr {
				if str != "" {
					number, _ := strconv.ParseInt(str, 10, 32)
					groups = append(groups, int32(number))
				}
			}
		}

		if username == nil {
			fmt.Fprintf(os.Stderr, "Error: username is required\n")
			os.Exit(1)
		}

		// if no roles are provided, no groups need to be provided as well
		if (len(roles) == 0 && len(groups) > 0) || (len(roles) > 0 && len(groups) == 0) {
			fmt.Fprintf(os.Stderr, "Error: if you specify roles, you need to specify groups as well\n")
			os.Exit(1)
		}

		// if no roles or no groups we need to ask if they want to create a single user
		if org == 0 && !force {
			confirmed := askForConfirmation("You didn't specify any org id. Do you want to create an self enrolled user?")
			if !confirmed {
				fmt.Println("User creation cancelled")
				os.Exit(0)
			}
		}

		// if org is set and groups or roles are empty, verify that this is what we want to do
		if (org != 0 && len(groups) == 0 && len(roles) == 0) && !force {
			confirmed := askForConfirmation("You didn't specify either groups or roles for the user. Are you sure to continue?")
			if !confirmed {
				fmt.Println("User creation cancelled")
				os.Exit(0)
			}
		}

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		ctx, cancel := util.GetAppContext()
		defer cancel()

		// now create the default fields for the user if needed
		if org == 0 {
			client := pb.NewOrganizationServiceClient(conn)
			resp, err := client.CreateOrganization(ctx, &pb.CreateOrganizationRequest{
				Name:    username.(string) + "-org",
				Company: username.(string) + " - Self enrolled",
			})
			util.ExitNicelyOnError(err, "Error creating organization")

			clientg := pb.NewGroupServiceClient(conn)
			respg, err := clientg.CreateGroup(ctx, &pb.CreateGroupRequest{OrganizationId: resp.Id,
				Name: username.(string) + "-group"})
			util.ExitNicelyOnError(err, "Error creating group")

			// create role for org and group
			clientr := pb.NewRoleServiceClient(conn)
			isAdmin := true
			groupId := respg.GroupId

			respo, err := clientr.CreateRoleByOrganization(ctx, &pb.CreateRoleByOrganizationRequest{OrganizationId: resp.Id,
				Name: username.(string) + "-admin-org", IsAdmin: &isAdmin})
			util.ExitNicelyOnError(err, "Error creating role")
			roles = append(roles, respo.Id)

			respr, err := clientr.CreateRoleByGroup(ctx, &pb.CreateRoleByGroupRequest{OrganizationId: resp.Id,
				GroupId: groupId, Name: username.(string) + "-admin-role", IsAdmin: &isAdmin})
			util.ExitNicelyOnError(err, "Error creating role")

			// now assign the role
			roles = append(roles, respr.Id)
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

		client := pb.NewUserServiceClient(conn)
		resp, err := client.CreateUser(ctx, &pb.CreateUserRequest{
			OrganizationId:      int32(org),
			Email:               emailPtr,
			Username:            username.(string),
			Password:            passwordPtr,
			FirstName:           firstNamePtr,
			LastName:            lastNamePtr,
			IsProtected:         protectedPtr,
			NeedsPasswordChange: needsPasswordChangePtr,
			RoleIds:             roles,
			GroupIds:            groups,
		})
		util.ExitNicelyOnError(err, "Error creating user")

		m := protojson.MarshalOptions{
			Indent: "  ",
		}
		out, err := m.Marshal(resp)
		util.ExitNicelyOnError(err, "Error marshalling json")
		fmt.Println(string(out))

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
	User_createCmd.Flags().Int32P("org-id", "o", 0, "Organization ID for the user")
	User_createCmd.Flags().StringP("roles", "r", "", "Comma separated list of roles")
	User_createCmd.Flags().StringP("groups", "g", "", "Comma separated list of groups")
	User_createCmd.Flags().BoolP("force", "s", false, "Skip confirmation")
	User_createCmd.Flags().BoolP("needs-password-change", "c", true, "Does the user need to change their password")
	err := User_createCmd.MarkFlagRequired("username")
	util.ExitNicelyOnError(err, "Error marking flag as required")
}
