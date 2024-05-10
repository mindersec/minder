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

// Package app provides the root command for the mindev CLI
package app

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/stacklok/minder/cmd/dev/app/bundles"
	"github.com/stacklok/minder/cmd/dev/app/image"
	"github.com/stacklok/minder/cmd/dev/app/rule_type"
	"github.com/stacklok/minder/cmd/dev/app/testserver"
	"github.com/stacklok/minder/internal/util/cli"
)

// CmdRoot represents the base command when called without any subcommands
func CmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mindev",
		Short: "mindev provides developer tooling for minder",
		Long: `For more information about minder, please visit:
https://docs.stacklok.com/minder`,
	}

	cmd.AddCommand(rule_type.CmdRuleType())
	cmd.AddCommand(image.CmdImage())
	cmd.AddCommand(testserver.CmdTestServer())
	cmd.AddCommand(bundles.CmdBundle())

	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	cmd := CmdRoot()
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	err := cmd.Execute()
	cli.ExitNicelyOnError(err, "Error on execute")
}
