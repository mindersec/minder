// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
)

func setupTestCredentials(t *testing.T, token string) {
	t.Helper()
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	minderDir := filepath.Join(tempDir, "minder")
	err := os.MkdirAll(minderDir, 0o700)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(minderDir, "config.yaml"), []byte{}, 0o600)
	require.NoError(t, err)

	creds := cli.OpenIdCredentials{
		AccessToken:          token,
		RefreshToken:         "dummy-refresh-token",
		AccessTokenExpiresAt: time.Now().Add(24 * time.Hour),
	}
	data, err := json.Marshal(creds)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(minderDir, "credentials.json"), data, 0o600)
	require.NoError(t, err)
}

func setupCorruptCredentials(t *testing.T) {
	t.Helper()
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	minderDir := filepath.Join(tempDir, "minder")
	err := os.MkdirAll(minderDir, 0o700)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(minderDir, "config.yaml"), []byte{}, 0o600)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(minderDir, "credentials.json"), []byte("invalid-json{"), 0o600)
	require.NoError(t, err)
}

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout/Env state
func TestTokenCommand(t *testing.T) {
	t.Run("valid token printed to stdout via RunCmdTests", func(t *testing.T) {
		setupTestCredentials(t, "test-token-value-12345")

		tests := []cli.CmdTestCase{
			{
				Name:           "prints access token to stdout",
				Args:           []string{"auth", "token"},
				GoldenFileName: "token",
			},
		}

		cli.RunCmdTests(t, tests, AuthCmd)
	})

	t.Run("error output to stderr on corrupt credentials via RunCmdTests", func(t *testing.T) {
		setupCorruptCredentials(t)

		tests := []cli.CmdTestCase{
			{
				Name:           "routes errors to stderr",
				Args:           []string{"auth", "token"},
				ExpectedStderr: "Error getting token:",
				ExpectedError:  "error loading credentials",
			},
		}

		cli.RunCmdTests(t, tests, AuthCmd)
	})
}

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout/Env state
func TestTokenCommand_StreamsSeparation(t *testing.T) {
	t.Run("clean stdout and empty stderr on success", func(t *testing.T) {
		expectedToken := "clean-auth-token-xyz"
		setupTestCredentials(t, expectedToken)

		rootCmd := app.RootCmd
		cli.ResetEntireTree(rootCmd)
		viper.Reset()

		stdoutBuf := new(bytes.Buffer)
		stderrBuf := new(bytes.Buffer)
		rootCmd.SetOut(stdoutBuf)
		rootCmd.SetErr(stderrBuf)
		rootCmd.SetArgs([]string{"auth", "token"})

		err := rootCmd.Execute()
		require.NoError(t, err)

		assert.Equal(t, expectedToken, stdoutBuf.String(), "token should be printed to stdout cleanly")
		assert.Empty(t, stderrBuf.String(), "stderr must be empty when getting token succeeds")
	})

	t.Run("error routed to stderr and empty stdout on failure", func(t *testing.T) {
		setupCorruptCredentials(t)

		rootCmd := app.RootCmd
		cli.ResetEntireTree(rootCmd)
		viper.Reset()

		stdoutBuf := new(bytes.Buffer)
		stderrBuf := new(bytes.Buffer)
		rootCmd.SetOut(stdoutBuf)
		rootCmd.SetErr(stderrBuf)
		rootCmd.SetArgs([]string{"auth", "token"})

		err := rootCmd.Execute()
		require.Error(t, err)

		assert.Contains(t, stderrBuf.String(), "Error getting token:", "error message must be routed to stderr")
		assert.Empty(t, stdoutBuf.String(), "stdout must remain clean and empty when an error occurs")
	})
}
