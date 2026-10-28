// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package docs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Modifies working directory and global rootCmd state
func TestDocsCommand(t *testing.T) {
	t.Run("verifies command properties", func(t *testing.T) {
		rootCmd := app.RootCmd
		cli.ResetEntireTree(rootCmd)

		cmd, _, err := rootCmd.Find([]string{"docs"})
		require.NoError(t, err)
		require.NotNil(t, cmd)

		assert.Equal(t, "docs", cmd.Use)
		assert.Equal(t, "Generates documentation for the client", cmd.Short)
		assert.True(t, cmd.Hidden)
	})

	t.Run("generates markdown documentation tree", func(t *testing.T) {
		tempDir := t.TempDir()

		// Target directory required by minder docs command: ./docs/docs/ref/cli
		targetDir := filepath.Join(tempDir, "docs", "docs", "ref", "cli")
		err := os.MkdirAll(targetDir, 0o750)
		require.NoError(t, err)

		origWd, err := os.Getwd()
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Chdir(origWd)
		})

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		rootCmd := app.RootCmd
		cli.ResetEntireTree(rootCmd)

		rootCmd.SetArgs([]string{"docs"})
		err = rootCmd.Execute()
		require.NoError(t, err)

		// Verify minder_config.md is generated with expected header
		configDocPath := filepath.Join(targetDir, "minder_config.md")
		assert.FileExists(t, configDocPath)
		configDocBytes, err := os.ReadFile(configDocPath)
		require.NoError(t, err)
		assert.Contains(t, string(configDocBytes), "---\ntitle: minder config\n---\n")

		// Verify main CLI docs are generated
		minderDocPath := filepath.Join(targetDir, "minder.md")
		assert.FileExists(t, minderDocPath)
		minderDocBytes, err := os.ReadFile(minderDocPath)
		require.NoError(t, err)
		assert.Contains(t, string(minderDocBytes), "---\ntitle: minder\n---\n")
	})

	t.Run("fails when target output directory does not exist", func(t *testing.T) {
		tempDir := t.TempDir()

		origWd, err := os.Getwd()
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Chdir(origWd)
		})

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		rootCmd := app.RootCmd
		cli.ResetEntireTree(rootCmd)

		rootCmd.SetArgs([]string{"docs"})
		err = rootCmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to open file for config docs")
	})
}
