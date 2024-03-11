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

// Package ruletype provides the CLI subcommand for managing rules
package ruletype

import (
	"github.com/spf13/cobra"

	"github.com/stacklok/minder/cmd/cli/app"
	ghclient "github.com/stacklok/minder/internal/providers/github"
)

// ruleTypeCmd is the root command for the rule subcommands
var ruleTypeCmd = &cobra.Command{
	Use:   "ruletype",
	Short: "Manage rule types",
	Long:  `The ruletype subcommands allows the management of rule types within Minder.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(ruleTypeCmd)
	// Flags for all subcommands
	ruleTypeCmd.PersistentFlags().StringP("provider", "p", ghclient.Github, "Name of the provider, i.e. github")
	ruleTypeCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}
