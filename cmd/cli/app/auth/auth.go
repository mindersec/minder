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

	"github.com/stacklok/mediator/cmd/cli/app"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// authCmd represents the auth command
var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "medctl auth commands",
	Long: `The medctl auth command group lets you grant and revoke
	authorization to tokens within mediators control plane.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("auth called")
	},
}

func init() {
	app.RootCmd.AddCommand(AuthCmd)
	AuthCmd.PersistentFlags().String("grpc-host", "", "Server host")
	AuthCmd.PersistentFlags().Int("grpc-port", 0, "Server port")
	AuthCmd.PersistentFlags().String("provider", "", "The OAuth2 provider to use for login")
	if err := viper.BindPFlags(AuthCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}

}
