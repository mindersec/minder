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

// Package history provides the CLI subcommand for managing profile statuses
package history

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stacklok/minder/cmd/cli/app"
)

// historyCmd is the root command for the profile_status subcommands
var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "View evaluation history",
	Long:  `The history subcommands allows evaluation history to be viewed.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(historyCmd)
	historyCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
	historyCmd.PersistentFlags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
