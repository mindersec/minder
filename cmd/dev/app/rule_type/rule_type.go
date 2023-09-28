// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package rule_type provides the CLI subcommand for developing rules
// e.g. the 'rule type test' subcommand.
package rule_type

import (
	"github.com/spf13/cobra"

	"github.com/stacklok/mediator/cmd/dev/app"
)

// rtypeCmd is the root command for the rule subcommands
var rtypeCmd = &cobra.Command{
	Use:          "rule_type",
	Short:        "rule type development commands",
	Long:         `The 'rule type' subcommand allows you to develop rules`,
	SilenceUsage: true,
}

func init() {
	app.RootCmd.AddCommand(rtypeCmd)
}
