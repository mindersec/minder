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

// Package artifact provides the artifact subcommands
package artifact

import (
	"github.com/spf13/cobra"

	"github.com/stacklok/minder/cmd/cli/app"
	ghclient "github.com/stacklok/minder/internal/providers/github"
)

// ArtifactCmd is the artifact subcommand
var ArtifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Manage artifacts within a minder control plane",
	Long:  `The minder artifact commands allow the management of artifacts within a minder control plane`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(ArtifactCmd)
	// Flags for all subcommands
	ArtifactCmd.PersistentFlags().StringP("provider", "p", ghclient.Github, "Name of the provider, i.e. github")
	ArtifactCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}
