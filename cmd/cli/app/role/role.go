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

package role

import (
	"fmt"
	"os"

	"github.com/stacklok/mediator/cmd/cli/app"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var RoleCmd = &cobra.Command{
	Use:   "role",
	Short: "Manage roles within a mediator control plane",
	Long: `The medctl role subcommands allows the management of roles within
a mediator controlplane.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("role called")
	},
}

func init() {
	app.RootCmd.AddCommand(RoleCmd)
	if err := viper.BindPFlags(RoleCmd.PersistentFlags()); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags19: %s\n", err)
	}
}
