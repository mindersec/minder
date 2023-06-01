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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// authCmd represents the auth command
var auth_revokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke a token / log out of a mediator session",
	Long: `Revoke a token within a mediator control plane, by expiring the token 
and removing it from the database. This is effectively the same as logging out. 
If no --user-id flag is passed, it will revoke the token for the current logged in
user. If a --user-id flag is passed, it will revoke the token for the specified user, 
but only if the current user has sufficient privileges.
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("auth revoke called")
	},
}

func init() {
	AuthCmd.AddCommand(auth_revokeCmd)
}
