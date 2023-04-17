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
var auth_loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to mediator",
	Long: `This command allows a user to login to mediator
, should you require an oauth2 login, then pass in the --provider flag,
e.g. --provider=github. This will then initiate the OAuth2 flow and allow
mediator to access user account details via the provider / iDP .`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("auth login called")
		// provider := viper.GetString("provider")
	},
}

func init() {
	AuthCmd.AddCommand(auth_loginCmd)
	auth_loginCmd.PersistentFlags().String("provider", "", "The OAuth2 provider to use for login")
	if err := viper.BindPFlags(auth_loginCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
