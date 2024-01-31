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

// Package project is the root command for the project subcommands
package project

import (
	"github.com/spf13/cobra"

	"github.com/stacklok/minder/cmd/cli/app"
)

// ProjectCmd is the root command for the project subcommands
var ProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage project within a minder control plane",
	Long:  `The minder project commands manage projects within a minder control plane.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(ProjectCmd)
}
