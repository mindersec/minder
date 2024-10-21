// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package apply provides the apply command for the minder CLI
package apply

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
)

// DocsCmd generates documentation
var DocsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generates documentation for the client",
	Long:  `Generates documentation for the client.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return cli.MessageAndError("Error binding flags", err)
		}
		return nil
	},
	RunE: func(_ *cobra.Command, _ []string) error {
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
		// GenMarkdownTreeCustom doesn't include additional help commands, so write it manually.
		configHelpFile, err := os.Create("./docs/docs/ref/cli/minder_config.md")
		if err != nil {
			return fmt.Errorf("Unable to open file for config docs: %w", err)
		}
		if _, err := configHelpFile.WriteString("---\ntitle: minder config\n---\n"); err != nil {
			return fmt.Errorf("Unable to write docs header: %w", err)
		}
		if err := doc.GenMarkdown(app.ConfigHelpCmd, configHelpFile); err != nil {
			return fmt.Errorf("Unable to write markdown for config help: %w", err)
		}
		return doc.GenMarkdownTreeCustom(app.RootCmd, "./docs/docs/ref/cli", prefix, identity)
	},
}

func init() {
	app.RootCmd.AddCommand(DocsCmd)
}
