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

// Package apply provides the apply command for the minder CLI
package apply

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/util"
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
		// We auto-generate the docs daily, so don't include the date at the bottom.
		app.RootCmd.DisableAutoGenTag = true
		// We need to add header material, since GenMarkdownTree always
		// generates an h2 and not an h1.
		// See https://github.com/spf13/cobra/issues/1948
		prefix := func(filename string) string {
			// Undo the transformation in https://github.com/spf13/cobra/blob/v1.7.0/doc/md_docs.go#L141
			filename = filepath.Base(filename)
			cmdString := strings.ReplaceAll(strings.TrimSuffix(filename, ".md"), "_", " ")
			return fmt.Sprintf("---\ntitle: %s\n---\n", cmdString)
		}
		identity := func(s string) string { return s }
		err := doc.GenMarkdownTreeCustom(app.RootCmd, "./docs/docs/ref/cli/commands", prefix, identity)
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	app.RootCmd.AddCommand(DocsCmd)
}
