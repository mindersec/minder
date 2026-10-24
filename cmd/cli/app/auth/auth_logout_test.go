// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"bytes"
	"encoding/json"
	"net/url"
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

// runLogout executes the auth logout command with custom in-memory buffers.
// RunCmdTests is not used here because the logout test requires custom filesystem setup
// (creating credentials files) and post-execution assertions verifying that the credentials
// file was physically deleted from disk.
func runLogout(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	rootCmd := app.RootCmd
	cli.ResetEntireTree(rootCmd)
	viper.Reset()

	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)
	rootCmd.SetOut(stdoutBuf)
	rootCmd.SetErr(stderrBuf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return stdoutBuf.String(), stderrBuf.String(), err
}

const defaultLogoutConfig = "grpc_server:\n  host: api.example.com\n  port: 443\nidentity:\n  cli:\n    issuer_url: https://auth.example.com\n    realm: minder\n"

func setupLogoutEnv(t *testing.T, serverSpecific bool, createCreds bool, configYAML string) string {
	t.Helper()
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	minderDir := filepath.Join(tempDir, "minder")
	err := os.MkdirAll(minderDir, 0o700)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(minderDir, "config.yaml"), []byte(configYAML), 0o600)
	require.NoError(t, err)

	credsFileName := "credentials.json"
	if serverSpecific {
		credsFileName = "api.example.com_443.json"
	}
	credsPath := filepath.Join(minderDir, credsFileName)

	if createCreds {
		creds := cli.OpenIdCredentials{
			AccessToken:          "dummy-token",
			RefreshToken:         "dummy-refresh",
			AccessTokenExpiresAt: time.Now().Add(24 * time.Hour),
		}
		data, err := json.Marshal(creds)
		require.NoError(t, err)
		err = os.WriteFile(credsPath, data, 0o600)
		require.NoError(t, err)
	}

	return credsPath
}

//nolint:paralleltest // Cannot run in parallel because it modifies global env and viper state
func TestLogoutCommand(t *testing.T) {
	t.Run("successfully logs out and removes credentials file", func(t *testing.T) {
		credsPath := setupLogoutEnv(t, true, true, defaultLogoutConfig)
		require.FileExists(t, credsPath)

		stdout, stderr, err := runLogout(t, "auth", "logout")
		require.NoError(t, err)

		// Success banner and browser logout hint belong on stderr (see #6895)
		assert.Contains(t, stderr, "You have successfully logged out of the CLI.")
		assert.Contains(t, stderr, "If you would like to log out of the browser")
		expectedLogoutURL := "https://auth.example.com/realms/minder/protocol/openid-connect/logout"
		assert.Contains(t, stderr, expectedLogoutURL)

		// Verify credentials file is removed and stdout is completely clean
		assert.NoFileExists(t, credsPath)
		assert.Empty(t, stdout)
	})

	t.Run("successfully removes legacy credentials file when server specific file does not exist", func(t *testing.T) {
		credsPath := setupLogoutEnv(t, false, true, defaultLogoutConfig)
		require.FileExists(t, credsPath)

		stdout, stderr, err := runLogout(t, "auth", "logout")
		require.NoError(t, err)

		assert.Contains(t, stderr, "You have successfully logged out of the CLI.")
		assert.NoFileExists(t, credsPath)
		assert.Empty(t, stdout)
	})

	t.Run("returns error when credentials file does not exist", func(t *testing.T) {
		setupLogoutEnv(t, false, false, defaultLogoutConfig)

		stdout, _, err := runLogout(t, "auth", "logout")
		require.Error(t, err)

		var wrappedErr *cli.ErrWrappedCLIError
		require.ErrorAs(t, err, &wrappedErr)
		assert.Equal(t, "Error removing credentials", wrappedErr.Message)
		assert.Contains(t, wrappedErr.Err.Error(), "error removing credentials file")
		assert.Empty(t, stdout)
	})

	t.Run("returns error when issuer URL is invalid", func(t *testing.T) {
		invalidConfig := "grpc_server:\n  host: api.example.com\n  port: 443\nidentity:\n  cli:\n    issuer_url: \"http://[::1]:namedport\"\n"
		setupLogoutEnv(t, true, true, invalidConfig)

		stdout, _, err := runLogout(t, "auth", "logout")
		require.Error(t, err)

		var wrappedErr *cli.ErrWrappedCLIError
		require.ErrorAs(t, err, &wrappedErr)
		assert.Equal(t, "Error parsing issuer URL", wrappedErr.Message)
		var urlErr *url.Error
		assert.ErrorAs(t, wrappedErr.Err, &urlErr)
		assert.Empty(t, stdout)
	})
}
