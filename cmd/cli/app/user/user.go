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

// Package user provides the user subcommand for the medic CLI.
package user

import (
	"github.com/stacklok/mediator/cmd/cli/app"

	"github.com/spf13/cobra"
)

// UserCmd is the root command for the user subcommands.
var UserCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users within a mediator control plane",
	Long: `The medic user subcommands allows the management of users within
a mediator controlplane.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(UserCmd)
}
