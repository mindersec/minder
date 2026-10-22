// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/util/cli"
)

func setupTestCredentials(t *testing.T, credsContent []byte) {
	t.Helper()
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	minderDir := filepath.Join(tempDir, "minder")
	err := os.MkdirAll(minderDir, 0o700)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(minderDir, "config.yaml"), []byte{}, 0o600)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(minderDir, "credentials.json"), credsContent, 0o600)
	require.NoError(t, err)
}

func validCredentialsJSON(t *testing.T, token string) []byte {
	t.Helper()
	creds := cli.OpenIdCredentials{
		AccessToken:          token,
		RefreshToken:         "dummy-refresh-token",
		AccessTokenExpiresAt: time.Now().Add(24 * time.Hour),
	}
	data, err := json.Marshal(creds)
	require.NoError(t, err)
	return data
}

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout/Env state
func TestTokenCommand(t *testing.T) {
	t.Run("valid token printed to stdout with clean stderr", func(t *testing.T) {
		const expectedToken = "test-token-value-12345"
		setupTestCredentials(t, validCredentialsJSON(t, expectedToken))

		tests := []cli.CmdTestCase{
			{
				Name:              "prints access token to stdout and leaves stderr empty",
				Args:              []string{"auth", "token"},
				ExpectedStdout:    expectedToken,
				ExpectEmptyStderr: true,
			},
		}

		cli.RunCmdTests(t, tests, AuthCmd)
	})

	t.Run("error output to stderr with clean stdout on corrupt credentials", func(t *testing.T) {
		setupTestCredentials(t, []byte("invalid-json{"))

		tests := []cli.CmdTestCase{
			{
				Name:              "routes error to stderr and leaves stdout empty",
				Args:              []string{"auth", "token"},
				ExpectedError:     "error loading credentials",
				ExpectEmptyStdout: true,
			},
		}

		cli.RunCmdTests(t, tests, AuthCmd)
	})
}
