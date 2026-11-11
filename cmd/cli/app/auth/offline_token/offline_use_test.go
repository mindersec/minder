// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package offline_token_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/util/cli"
)

//nolint:paralleltest // Cannot run in parallel because it modifies global env and viper state
func TestOfflineUseCommand(t *testing.T) {
	t.Run("fails when both --file and --token flags are specified", func(t *testing.T) {
		setupOfflineTokenEnv(t, "https://auth.example.com")

		_, _, err := runOfflineTokenCmd(t, "auth", "offline-token", "use", "--file", "dummy.token", "--token", "abc-123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "none of the others can be")
	})

	t.Run("fails when token file does not exist", func(t *testing.T) {
		setupOfflineTokenEnv(t, "https://auth.example.com")

		_, _, err := runOfflineTokenCmd(t, "auth", "offline-token", "use", "--file", "nonexistent-token-file.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error reading file")
	})

	t.Run("successfully exchanges token and saves credentials", func(t *testing.T) {
		var receivedRefreshToken, receivedClientId string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/realms/test-realm/protocol/openid-connect/token" && r.Method == http.MethodPost {
				_ = r.ParseForm()
				receivedRefreshToken = r.FormValue("refresh_token")
				receivedClientId = r.FormValue("client_id")

				resp := map[string]any{
					"access_token":  "new-access-token-xyz",
					"refresh_token": "new-refresh-token-xyz",
					"expires_in":    3600,
					"token_type":    "Bearer",
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		tempDir := setupOfflineTokenEnv(t, server.URL)

		stdout, stderr, err := runOfflineTokenCmd(t, "auth", "offline-token", "use", "--token", "test-offline-refresh-token")
		require.NoError(t, err)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Your access credentials have been saved to")
		assert.Equal(t, "test-offline-refresh-token", receivedRefreshToken)
		assert.Equal(t, "test-client", receivedClientId)

		// Verify saved credentials on disk
		credsPath := filepath.Join(tempDir, "minder", "api.example.com_443.json")
		require.FileExists(t, credsPath)
		data, err := os.ReadFile(credsPath)
		require.NoError(t, err)

		var savedCreds cli.OpenIdCredentials
		err = json.Unmarshal(data, &savedCreds)
		require.NoError(t, err)
		assert.Equal(t, "new-access-token-xyz", savedCreds.AccessToken)
		assert.Equal(t, "new-refresh-token-xyz", savedCreds.RefreshToken)
		assert.True(t, savedCreds.AccessTokenExpiresAt.After(time.Now()))
	})

	t.Run("successfully uses token from file", func(t *testing.T) {
		var receivedRefreshToken string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/realms/test-realm/protocol/openid-connect/token" && r.Method == http.MethodPost {
				_ = r.ParseForm()
				receivedRefreshToken = r.FormValue("refresh_token")

				resp := map[string]any{
					"access_token":  "access-token-file",
					"refresh_token": "refresh-token-file",
					"expires_in":    3600,
					"token_type":    "Bearer",
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		tempDir := setupOfflineTokenEnv(t, server.URL)
		tokenFilePath := filepath.Join(tempDir, "offline.token")
		err := os.WriteFile(tokenFilePath, []byte("file-based-offline-token"), 0o600)
		require.NoError(t, err)

		stdout, stderr, err := runOfflineTokenCmd(t, "auth", "offline-token", "use", "--file", tokenFilePath)
		require.NoError(t, err)
		assert.Empty(t, stdout)
		assert.Contains(t, stderr, "Your access credentials have been saved to")
		assert.Equal(t, "file-based-offline-token", receivedRefreshToken)
	})

	t.Run("fails when token exchange endpoint returns an error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"Token has been revoked"}`))
		}))
		defer server.Close()

		setupOfflineTokenEnv(t, server.URL)

		_, _, err := runOfflineTokenCmd(t, "auth", "offline-token", "use", "--token", "revoked-token")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "couldn't fetch credentials")
	})

	t.Run("returns error when issuer URL is malformed", func(t *testing.T) {
		setupOfflineTokenEnv(t, "http://[::1]:namedport")

		_, _, err := runOfflineTokenCmd(t, "auth", "offline-token", "use", "--token", "any-token")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "couldn't get realm URL")
	})
}
