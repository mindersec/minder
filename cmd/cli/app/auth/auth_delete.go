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
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// authCmd represents the auth command
var auth_deluserCmd = &cobra.Command{
	Use:   "delete-user",
	Short: "Delete a user account within mediator",
	Long: `Delete a user account within mediator, by removing the user from the
database. This will also revoke any tokens associated with the user.

You can delete a user by passing in the user ID, e.g.
medctl auth delete-user --id=1234

To delete a user by username, pass in the --username flag, e.g.
medctl auth delete-user --username=foo

Note: This command will only work if you are logged in as user with a current
access tokenm, with sufficient privileges. If you peform this command with no
--username or --id flag, it will delete the user associated with the current.
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("auth delete called")
	},
}

func init() {
	AuthCmd.AddCommand(auth_deluserCmd)
	auth_deluserCmd.PersistentFlags().String("username", "", "The username to delete")
	auth_deluserCmd.PersistentFlags().Int32("id", 0, "The password for the user to delete")
	auth_deluserCmd.PersistentFlags().Bool("force", false, "Force deletion of user without confirmation")
	if err := viper.BindPFlags(auth_deluserCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
