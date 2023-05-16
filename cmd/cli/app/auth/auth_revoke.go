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
	Short: "Revoke a token",
	Long: `Revoke a token within mediator, by expiring the token and removing it
from the database. If the token is a refresh token, then the associated access
token will also be revoked.

You can revoke a token by passing in the token itself, or by passing in the
token ID.

To revoke a token by ID, pass in the --id flag, e.g.
medctl auth revoke --id=1234

To revoke a token by value, pass in the --token flag, e.g.
medctl auth revoke --token=1234-1234-1234-1234
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("account revoke called")
		// token := viper.GetString("token")
		// id := viper.GetInt32("id")
	},
}

func init() {
	AuthCmd.AddCommand(auth_revokeCmd)
	auth_revokeCmd.PersistentFlags().String("token", "", "The token to revoke")
	auth_revokeCmd.PersistentFlags().Int32("id", 0, "The ID of the token to revoke")
	if err := viper.BindPFlags(auth_revokeCmd.PersistentFlags()); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
	}
}
