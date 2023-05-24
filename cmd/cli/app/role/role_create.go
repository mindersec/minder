//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.role/licenses/LICENSE-2.0
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

package role

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var role_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an role within a mediator control plane",
	Long: `The medctl role create subcommand lets you create new roles
within a mediator control plane.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("role create called")
	},
}

func init() {
	RoleCmd.AddCommand(role_createCmd)
	// flag for name
	role_createCmd.Flags().StringP("name", "n", "", "Name of the role")
	role_createCmd.Flags().BoolP("is-admin", "a", false, "Is the role an admin role")
	role_createCmd.Flags().BoolP("active", "e", true, "Whether the role is active or not")

	if err := viper.BindPFlags(role_createCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
