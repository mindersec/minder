// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package offline_token_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/cmd/cli/app"
	_ "github.com/mindersec/minder/cmd/cli/app/auth/offline_token"
	"github.com/mindersec/minder/internal/util/cli"
)

func runOfflineTokenCmd(t *testing.T, args ...string) (string, string, error) {
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

func setupOfflineTokenEnv(t *testing.T, issuerURL string) string {
	t.Helper()
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	minderDir := filepath.Join(tempDir, "minder")
	err := os.MkdirAll(minderDir, 0o700)
	require.NoError(t, err)

	configYAML := fmt.Sprintf(`grpc_server:
  host: api.example.com
  port: 443
identity:
  cli:
    issuer_url: %s
    realm: test-realm
    client_id: test-client
`, issuerURL)

	err = os.WriteFile(filepath.Join(minderDir, "config.yaml"), []byte(configYAML), 0o600)
	require.NoError(t, err)

	return tempDir
}

//nolint:paralleltest // Cannot run in parallel because it modifies global env and viper state
func TestOfflineRevokeCommand(t *testing.T) {
	t.Run("fails when both --file and --token flags are specified", func(t *testing.T) {
		setupOfflineTokenEnv(t, "https://auth.example.com")

		_, _, err := runOfflineTokenCmd(t, "auth", "offline-token", "revoke", "--file", "dummy.token", "--token", "abc-123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "none of the others can be")
	})

	t.Run("fails when token file does not exist", func(t *testing.T) {
		setupOfflineTokenEnv(t, "https://auth.example.com")

		_, _, err := runOfflineTokenCmd(t, "auth", "offline-token", "revoke", "--file", "nonexistent-token-file.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error reading file")
	})

	t.Run("successfully revokes token supplied via --token flag", func(t *testing.T) {
		var receivedToken, receivedClientId, receivedHint string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/realms/test-realm/protocol/openid-connect/revoke" && r.Method == http.MethodPost {
				_ = r.ParseForm()
				receivedToken = r.FormValue("token")
				receivedClientId = r.FormValue("client_id")
				receivedHint = r.FormValue("token_type_hint")
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		setupOfflineTokenEnv(t, server.URL)

		stdout, stderr, err := runOfflineTokenCmd(t, "auth", "offline-token", "revoke", "--token", "direct-secret-token")
		require.NoError(t, err)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Token revoked")
		assert.Equal(t, "direct-secret-token", receivedToken)
		assert.Equal(t, "test-client", receivedClientId)
		assert.Equal(t, "refresh_token", receivedHint)
	})

	t.Run("successfully revokes token supplied via --file flag", func(t *testing.T) {
		var receivedToken string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/realms/test-realm/protocol/openid-connect/revoke" {
				_ = r.ParseForm()
				receivedToken = r.FormValue("token")
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		tempDir := setupOfflineTokenEnv(t, server.URL)
		tokenFilePath := filepath.Join(tempDir, "sample.token")
		err := os.WriteFile(tokenFilePath, []byte("file-secret-token"), 0o600)
		require.NoError(t, err)

		stdout, stderr, err := runOfflineTokenCmd(t, "auth", "offline-token", "revoke", "--file", tokenFilePath)
		require.NoError(t, err)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Token revoked")
		assert.Equal(t, "file-secret-token", receivedToken)
	})

	t.Run("successfully revokes token supplied via MINDER_OFFLINE_TOKEN environment variable", func(t *testing.T) {
		var receivedToken string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/realms/test-realm/protocol/openid-connect/revoke" {
				_ = r.ParseForm()
				receivedToken = r.FormValue("token")
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		setupOfflineTokenEnv(t, server.URL)
		t.Setenv("MINDER_OFFLINE_TOKEN", "env-secret-token")

		stdout, stderr, err := runOfflineTokenCmd(t, "auth", "offline-token", "revoke")
		require.NoError(t, err)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Token revoked")
		assert.Equal(t, "env-secret-token", receivedToken)
	})

	t.Run("returns error when issuer URL is malformed", func(t *testing.T) {
		setupOfflineTokenEnv(t, "http://[::1]:namedport")

		_, _, err := runOfflineTokenCmd(t, "auth", "offline-token", "revoke", "--token", "any-token")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "couldn't revoke token")
	})
}
