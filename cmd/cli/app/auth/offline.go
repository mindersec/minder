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

package auth

import (
	"github.com/spf13/cobra"
)

// OfflineTokenCmd represents the offline-token set of sub-commands
var OfflineTokenCmd = &cobra.Command{
	Use:   "offline-token",
	Short: "Manage offline tokens",
	Long: `The minder auth offline-token command project lets you manage offline tokens
for the minder control plane.

Offline tokens are used to authenticate to the minder control plane without
requiring the user's presence. This is useful for long-running processes
that need to authenticate to the control plane.`,

	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	AuthCmd.AddCommand(OfflineTokenCmd)
}
