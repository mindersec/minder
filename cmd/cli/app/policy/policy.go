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

// Package policy provides the CLI subcommand for managing policies
package policy

import (
	"github.com/stacklok/mediator/cmd/cli/app"

	"github.com/spf13/cobra"
)

// PolicyCmd is the root command for the policy subcommands
var PolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage policies within a mediator control plane",
	Long: `The medic policy subcommands allows the management of policies within
a mediator controlplane.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(PolicyCmd)
}
