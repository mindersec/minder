// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package offline_token_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it modifies global env and viper state
func TestOfflineGetCommandFlags(t *testing.T) {
	t.Run("verifies default flag values for get subcommand", func(t *testing.T) {
		rootCmd := app.RootCmd
		cli.ResetEntireTree(rootCmd)

		cmd, _, err := rootCmd.Find([]string{"auth", "offline-token", "get"})
		require.NoError(t, err)
		require.NotNil(t, cmd)

		fileFlag := cmd.Flag("file")
		require.NotNil(t, fileFlag)
		assert.Equal(t, "offline.token", fileFlag.DefValue)
		assert.Equal(t, "f", fileFlag.Shorthand)

		skipBrowserFlag := cmd.Flag("skip-browser")
		require.NotNil(t, skipBrowserFlag)
		assert.Equal(t, "false", skipBrowserFlag.DefValue)
	})

	t.Run("fails when config is invalid or missing required issuer", func(t *testing.T) {
		setupOfflineTokenEnv(t, "http://[::1]:namedport")

		_, _, err := runOfflineTokenCmd(t, "auth", "offline-token", "get", "--skip-browser")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error parsing issuer URL")
	})
}

//nolint:paralleltest // Cannot run in parallel because it modifies global env and viper state
func TestOfflineTokenParentCommand(t *testing.T) {
	t.Run("displays usage when offline-token parent command is invoked without subcommands", func(t *testing.T) {
		stdout, _, err := runOfflineTokenCmd(t, "auth", "offline-token")
		require.NoError(t, err)
		assert.Contains(t, stdout, "Usage:")
		assert.Contains(t, stdout, "minder auth offline-token [command]")
	})
}
