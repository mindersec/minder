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
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var role_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a role within a mediator controlplane",
	Long: `The medctl role delete subcommand lets you delete roles within a
mediator control plane.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("role delete called")
	},
}

func init() {
	RoleCmd.AddCommand(role_deleteCmd)
	role_deleteCmd.PersistentFlags().StringP("role-id", "r", "", "role-id of role to delete")
	role_deleteCmd.PersistentFlags().BoolP("force", "f", false, "Force deletion of role (WARNING: this will delete all resources associated with the role)")
	if err := viper.BindPFlags(role_deleteCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
