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

// Package repo contains the repo logic for the control plane
package repo

import (
	"github.com/spf13/cobra"

	"github.com/stacklok/minder/cmd/cli/app"
)

// RepoCmd is the root command for the repo subcommands
var RepoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories",
	Long:  `The repo commands allow the management of repositories within Minder.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(RepoCmd)
	// Flags for all subcommands
	RepoCmd.PersistentFlags().StringP("provider", "p", "", "Name of the provider, i.e. github")
	RepoCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}
