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

// Package invite provides the auth invite command for the minder CLI.
package invite

import (
	"github.com/spf13/cobra"

	"github.com/stacklok/minder/cmd/cli/app/auth"
)

// inviteCmd represents the offline-token set of sub-commands
var inviteCmd = &cobra.Command{
	Use:   "invite",
	Short: "Manage user invitations",
	Long:  `The minder auth invite command lets you manage (accept/decline/list) your invitations.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	auth.AuthCmd.AddCommand(inviteCmd)
}
