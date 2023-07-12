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

// Package policy_type provides the CLI subcommand for managing policies
package policy_type

import (
	"github.com/spf13/cobra"

	"github.com/stacklok/mediator/cmd/cli/app"
)

// PolicyTypeCmd is the root command for the policy subcommands
var PolicyTypeCmd = &cobra.Command{
	Use:   "policy_type",
	Short: "Manage policy types within a mediator control plane",
	Long: `The medic policy_type subcommands allows the management of policy types within
a mediator controlplane.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(PolicyTypeCmd)
}
