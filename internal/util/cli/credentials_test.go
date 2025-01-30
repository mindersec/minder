// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/util/cli"
)

var (
	// envLockXdgConfigHome is a mutex to ensure that all the tests that run os.SetEnv("XDG_CONFIG_HOME") need to be prevented from running at the same time as each other.
	envLock = &sync.Mutex{}

	XdgConfigHomeEnvVar = "XDG_CONFIG_HOME"
)

// Enforce that only one test setting environment variables runs at a time
func setEnvVar(t *testing.T, env string, value string) {
	t.Helper() // Keep golangci-lint happy
	envLock.Lock()
	t.Cleanup(envLock.Unlock)

	originalEnvVal := os.Getenv(env)
	err := os.Setenv(env, value)
	if err != nil {
		t.Errorf("error setting %v: %v", env, err)
	}

	t.Cleanup(func() { _ = os.Setenv(env, originalEnvVal) })
}

// TestGetGrpcConnection tests the GetGrpcConnection function
func TestGetGrpcConnection(t *testing.T) {
	t.Parallel()
	// authTokenMutex := &sync.Mutex{}
	tests := []struct {
		name          string
		grpcHost      string
		grpcPort      int
		allowInsecure bool
		issuerUrl     string
		clientId      string
		envToken      string
		expectedError bool
	}{
		{
			name:          "Valid GRPC connection to localhost with secure connection",
			grpcHost:      "127.0.0.1",
			grpcPort:      8090,
			allowInsecure: false,
			issuerUrl:     "http://localhost:8081",
			clientId:      "minder-cli",
			envToken:      "MINDER_AUTH_TOKEN",
			expectedError: false,
		},
		{
			name:          "Valid GRPC connection to localhost with insecure connection",
			grpcHost:      "localhost",
			grpcPort:      8090,
			allowInsecure: true,
			issuerUrl:     "http://localhost:8081",
			clientId:      "minder-cli",
			envToken:      "MINDER_AUTH_TOKEN",
			expectedError: false,
		},
		{
			name:          "Valid GRPC connection to localhost without passing MINDER_AUTH_TOKEN as an argument",
			grpcHost:      "127.0.0.1",
			grpcPort:      8090,
			allowInsecure: false,
			issuerUrl:     "http://localhost:8081",
			clientId:      "minder-cli",
			envToken:      "",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			setEnvVar(t, cli.MinderAuthTokenEnvVar, tt.envToken)
			conn, err := cli.GetGrpcConnection(tt.grpcHost, tt.grpcPort, tt.allowInsecure, tt.issuerUrl, tt.clientId)
			if (err != nil) != tt.expectedError {
				t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
			}
			if conn != nil {
				err = conn.Close()
				if err != nil {
					t.Errorf("error closing connection: %v", err)
				}
			}
		})
	}
}

// TestSaveCredentials tests the SaveCredentials function
func TestSaveCredentials(t *testing.T) {
	t.Parallel()

	tokens := cli.OpenIdCredentials{
		AccessToken:          "test_access_token",
		RefreshToken:         "test_refresh_token",
		AccessTokenExpiresAt: time.Time{}.Add(7 * 24 * time.Hour),
	}

	// Create a temporary directory
	testDir := t.TempDir()

	setEnvVar(t, XdgConfigHomeEnvVar, testDir)

	cfgPath := filepath.Join(testDir, "minder")

	expectedFilePath := filepath.Join(cfgPath, "credentials.json")

	filePath, err := cli.SaveCredentials(tokens)
	require.NoError(t, err)

	if filePath != expectedFilePath {
		t.Errorf("expected file path %v, got %v", expectedFilePath, filePath)
	}

	// Verify the file content
	credsJSON, err := json.Marshal(tokens)
	require.NoError(t, err)

	cleanPath := filepath.Clean(filePath)
	content, err := os.ReadFile(cleanPath)
	require.NoError(t, err)

	if string(content) != string(credsJSON) {
		t.Errorf("expected file content %v, got %v", string(credsJSON), string(content))
	}

}

// TestRemoveCredentials tests the RemoveCredentials function
func TestRemoveCredentials(t *testing.T) {
	t.Parallel()

	// Create a temporary directory
	testDir := t.TempDir()
	setEnvVar(t, XdgConfigHomeEnvVar, testDir)
	xdgConfigHome := os.Getenv(XdgConfigHomeEnvVar)

	filePath := filepath.Join(xdgConfigHome, "minder", "credentials.json")

	// Create a dummy credentials file
	err := os.MkdirAll(filepath.Dir(filePath), 0750)

	if err != nil {
		t.Fatalf("error creating directory: %v", err)
	}

	err = os.WriteFile(filePath, []byte(`{"access_token":"test_access_token","refresh_token":"test_refresh_token","access_token_expires_at":1234567890}`), 0600)
	if err != nil {
		t.Fatalf("error writing credentials to file: %v", err)
	}

	err = cli.RemoveCredentials()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the file is removed
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed, but it still exists")
	}
}

// TestRefreshCredentials tests the RefreshCredentials function
func TestRefreshCredentials(t *testing.T) {
	t.Parallel()
	// Create a temporary directory
	testDir := t.TempDir()

	setEnvVar(t, XdgConfigHomeEnvVar, testDir)
	tests := []struct {
		name           string
		refreshToken   string
		issuerUrl      string
		clientId       string
		responseBody   string
		expectedError  string
		expectedResult cli.OpenIdCredentials
	}{
		{
			name:         "Successful refresh with local server",
			refreshToken: "valid_refresh_token",
			clientId:     "minder-cli",
			responseBody: `{"access_token":"new_access_token","refresh_token":"new_refresh_token","expires_in":3600}`,
			expectedResult: cli.OpenIdCredentials{
				AccessToken:          "new_access_token",
				RefreshToken:         "new_refresh_token",
				AccessTokenExpiresAt: time.Now().Add(3600 * time.Second),
			},
		},
		{
			name:          "Error fetching new credentials (responseBody is missing) rwith local server",
			refreshToken:  "valid_refresh_token",
			clientId:      "minder-cli",
			expectedError: "error unmarshaling credentials: EOF",
		},
		{
			name:          "Error unmarshaling credentials with local server",
			refreshToken:  "valid_refresh_token",
			clientId:      "minder-cli",
			responseBody:  `invalid_json`,
			expectedError: "error unmarshaling credentials: invalid character 'i' looking for beginning of value",
		},
		{
			name:          "Error refreshing credentials with local server",
			refreshToken:  "valid_refresh_token",
			clientId:      "minder-cli",
			responseBody:  `{"error":"invalid_grant","error_description":"Invalid refresh token"}`,
			expectedError: "error refreshing credentials: invalid_grant: Invalid refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, tt.responseBody)
			}))
			defer server.Close()

			tt.issuerUrl = server.URL

			result, err := cli.RefreshCredentials(tt.refreshToken, tt.issuerUrl, tt.clientId)
			if tt.expectedError != "" {
				if err == nil || err.Error() != tt.expectedError {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if result.AccessToken != tt.expectedResult.AccessToken || result.RefreshToken != tt.expectedResult.RefreshToken {
					t.Errorf("expected result %v, got %v", tt.expectedResult, result)
				}
			}
		})
	}
}

// TestLoadCredentials tests the LoadCredentials function
func TestLoadCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		fileContent    string
		expectedError  string
		expectedResult cli.OpenIdCredentials
	}{
		{
			name:        "Successful load",
			fileContent: `{"access_token":"access_token","refresh_token":"refresh_token","expiry":"2024-10-05T17:46:27+10:00"}`,
			expectedResult: cli.OpenIdCredentials{
				AccessToken:          "access_token",
				RefreshToken:         "refresh_token",
				AccessTokenExpiresAt: time.Date(2024, 10, 5, 17, 46, 27, 0, time.FixedZone("AEST", 10*60*60)),
			},
		},
		{
			name:          "Error unmarshaling credentials",
			fileContent:   `invalid_json`,
			expectedError: "error unmarshaling credentials: invalid character 'i'",
		},
		{
			name:          "Error reading credentials file",
			fileContent:   "",
			expectedError: "error reading credentials file",
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testDir := t.TempDir()
			setEnvVar(t, XdgConfigHomeEnvVar, testDir)
			// Create the minder directory inside the temp directory
			minderDir := filepath.Join(testDir, "minder")
			err := os.MkdirAll(minderDir, 0750)
			if err != nil {
				t.Fatalf("failed to create minder directory: %v", err)
			}

			filePath := filepath.Join(minderDir, "credentials.json")

			if tt.fileContent != "" {
				// Create a temporary file with the specified content
				require.NoError(t, os.WriteFile(filePath, []byte(tt.fileContent), 0600))
				// Print the file path and content for debugging
				t.Logf("Test %s: written file path %s with content: %s", tt.name, filePath, tt.fileContent)
			} else {
				// Print the file path for debugging
				t.Logf("Test %s: file path %s not created as file content is empty", tt.name, filePath)
			}

			result, err := cli.LoadCredentials()
			if tt.expectedError != "" {
				if err == nil || !strings.HasPrefix(err.Error(), tt.expectedError) {
					t.Errorf("expected error matching %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				if result.AccessToken != tt.expectedResult.AccessToken || result.RefreshToken != tt.expectedResult.RefreshToken || !result.AccessTokenExpiresAt.Equal(tt.expectedResult.AccessTokenExpiresAt) {
					t.Errorf("expected result %v, got %v", tt.expectedResult, result)
				}
			}

		})

	}
}

// TestCase struct for holding test case data
type TestCase struct {
	name         string
	token        string
	issuerUrl    string
	clientId     string
	tokenHint    string
	expectedPath string
	expectError  bool
	createServer func(t *testing.T, tt TestCase) *httptest.Server
}

// TestRevokeToken tests the RevokeToken function
func TestRevokeToken(t *testing.T) {
	t.Parallel()
	tests := []TestCase{
		{
			name:         "Valid token revocation",
			token:        "test-token",
			issuerUrl:    "http://localhost:8081",
			clientId:     "minder-cli",
			tokenHint:    "refresh_token",
			expectedPath: "/realms/stacklok/protocol/openid-connect/revoke",
			expectError:  false,
			createServer: func(t *testing.T, tt TestCase) *httptest.Server {
				t.Helper()
				return httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
					err := r.ParseForm()
					require.NoError(t, err, "error parsing form")
					require.Equal(t, tt.clientId, r.Form.Get("client_id"))
					require.Equal(t, tt.token, r.Form.Get("token"))
					require.Equal(t, tt.tokenHint, r.Form.Get("token_type_hint"))
				}))
			},
		},
		{
			name:         "Invalid issuer URL",
			token:        "test-token",
			issuerUrl:    "://invalid-url",
			clientId:     "minder-cli",
			tokenHint:    "refresh_token",
			expectedPath: "",
			expectError:  true,
			createServer: nil,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var server *httptest.Server
			if tt.createServer != nil {
				server = tt.createServer(t, tt)
				defer server.Close()
				tt.issuerUrl = server.URL
			}

			err := cli.RevokeToken(tt.token, tt.issuerUrl, tt.clientId, tt.tokenHint)
			if (err != nil) != tt.expectError {
				t.Errorf("RevokeToken() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
