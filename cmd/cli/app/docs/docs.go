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

// Package apply provides the apply command for the medic CLI
package apply

import (
	"github.com/stacklok/mediator/cmd/cli/app"
	"github.com/stacklok/mediator/internal/util"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
)

// DocsCmd generates documentation
var DocsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generates documentation for the client",
	Long:  `Generates documentation for the client.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		err := viper.BindPFlags(cmd.Flags())
		util.ExitNicelyOnError(err, "Error binding flags")
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := doc.GenMarkdownTree(app.RootCmd, "./docs/docs/cli")
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	app.RootCmd.AddCommand(DocsCmd)
}
