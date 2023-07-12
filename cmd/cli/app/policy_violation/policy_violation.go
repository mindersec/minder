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

// Package policy_violation provides the CLI subcommand for managing policy_violations
package policy_violation

import (
	"github.com/stacklok/mediator/cmd/cli/app"

	"github.com/spf13/cobra"
)

// PolicyViolationCmd is the root command for the policy_violation subcommands
var PolicyViolationCmd = &cobra.Command{
	Use:   "policy_violation",
	Short: "Manage policy violations within a mediator control plane",
	Long: `The medic policy_violation subcommands allows the management of policy violations within
a mediator controlplane.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(PolicyViolationCmd)
}
