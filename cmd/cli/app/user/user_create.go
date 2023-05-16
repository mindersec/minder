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
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var user_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an user within the mediator controlplane",
	Long: `The medctl user create subcommand lets you create new users
within the mediator controlplane.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("user create called")
	},
}

func init() {
	UserCmd.AddCommand(user_createCmd)
	user_createCmd.PersistentFlags().StringP("name", "n", "", "Name of user")
	user_createCmd.PersistentFlags().StringP("email", "e", "", "Email of user")
	user_createCmd.PersistentFlags().StringP("username", "u", "", "Username of user")
	user_createCmd.PersistentFlags().StringP("password", "p", "", "Password of user")
	user_createCmd.PersistentFlags().StringP("first-name", "f", "", "First name of user")
	user_createCmd.PersistentFlags().StringP("last-name", "l", "", "Last name of user")
	user_createCmd.PersistentFlags().StringP("group-id", "g", "", "Group ID of user")
	user_createCmd.PersistentFlags().BoolP("active", "a", false, "Active status of user")
	if err := viper.BindPFlags(user_createCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
