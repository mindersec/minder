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

// Package version provides the version command for the minder CLI
package version

import (
	"github.com/spf13/cobra"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/constants"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/cli/useragent"
)

// VersionCmd is the version command
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of the minder CLI",
	Long:  `The minder version command prints the version of the minder CLI.`,
	Run: func(cmd *cobra.Command, _ []string) {
		cli.PrintCmd(cmd, constants.VerboseCLIVersion)
		cli.PrintCmd(cmd, "User Agent: %s", useragent.GetUserAgent())
	},
}

func init() {
	app.RootCmd.AddCommand(VersionCmd)
}
