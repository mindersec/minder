//
// Copyright 2024 Stacklok, Inc.
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

// Package role is the root command for the role subcommands
package role

import (
	"github.com/spf13/cobra"

	"github.com/stacklok/minder/cmd/cli/app/project"
)

// RoleCmd is the root command for the project subcommands
var RoleCmd = &cobra.Command{
	Use:   "role",
	Short: "Manage roles within a minder control plane",
	Long:  `The minder role commands manage permissions within a minder control plane.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	project.ProjectCmd.AddCommand(RoleCmd)
	RoleCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}
