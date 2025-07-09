// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/config/client"
)

var (
	// envLockXdgConfigHome is a mutex to ensure that all the tests that run os.SetEnv("XDG_CONFIG_HOME") need to be prevented from running at the same time as each other.
	envLock = &sync.Mutex{}

	XdgConfigHomeEnvVar = "XDG_CONFIG_HOME"
)

const serverAddress = "localhost:8081"

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
//
//nolint:paralleltest
func TestGetGrpcConnection(t *testing.T) {
	// authTokenMutex := &sync.Mutex{}

	fakeSvc := fakeUserService{}
	grpcServer := grpc.NewServer(grpc.Creds(insecure.NewCredentials()))
	minderv1.RegisterUserServiceServer(grpcServer, &fakeSvc)
	var grpcCalls, authCalls, legacyCalls int

	requestHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("content-type"), "application/grpc") {
			grpcCalls++
			grpcServer.ServeHTTP(w, r)
			return
		}
		switch r.URL.Path {
		case "/realms/stacklok/protocol/openid-connect/token":
			legacyCalls++
			w.WriteHeader(http.StatusBadRequest) // Custom error for hard-coded default
		case "/custom/protocol/openid-connect/token":
			authCalls++
			_, _ = w.Write([]byte(`{"access_token":"JWT"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	h2cServer := http2.Server{}
	h2cHandler := h2c.NewHandler(requestHandler, &h2cServer)

	authServer := httptest.NewUnstartedServer(h2cHandler)
	authServer.EnableHTTP2 = true
	require.NoError(t, http2.ConfigureServer(authServer.Config, &h2cServer))
	authServer.Start()
	fakeSvc.authUrl = authServer.URL + "/custom"
	t.Cleanup(authServer.Close)

	grpcHost, grpcPortStr, err := net.SplitHostPort(authServer.URL[7:]) // strip off "http://"
	require.NoError(t, err)
	grpcPort, err := strconv.Atoi(grpcPortStr)
	require.NoError(t, err)

	tests := []struct {
		name           string
		externalName   bool
		overridePort   int
		allowInsecure  bool
		envToken       string
		expectedGRPC   int
		expectedAuths  int
		expectedLegacy int
		expectedError  string
	}{
		{
			name:     "If the token is provided, create connection without calling server",
			envToken: "IS_A_TOKEN",
		},
		{
			name:          "With token, don't dial or handshake endpoint",
			externalName:  true,
			allowInsecure: false, // Force TLS with non-localhost name
			envToken:      "IS_A_TOKEN",
		},
		{
			name:          "Connect and get token from GRPC handshake",
			allowInsecure: true,
			envToken:      "IS_A_TOKEN",
		},
		{
			name:          "Localhost GRPC auto-discovery",
			allowInsecure: false,
			expectedGRPC:  1,
			expectedAuths: 1,
			envToken:      "",
		},
		{
			// It's not easy to thread a set of trusted certs to the client call, so we only test non-TLS here
			name:          "GRPC auto-discovery with insecure external host",
			externalName:  true,
			allowInsecure: true,
			envToken:      "IS_A_TOKEN",
		},
		{
			name:           "Defaults connect to legacy endpoint",
			overridePort:   9, // discard service, generally not listening
			allowInsecure:  true,
			expectedLegacy: 1,
			expectedError:  "error unmarshaling credentials: EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvVar(t, cli.MinderAuthTokenEnvVar, tt.envToken)
			grpcCalls = 0
			authCalls = 0
			legacyCalls = 0

			host := grpcHost
			// Use https://sslip.io to fake out localhost detection
			if tt.externalName {
				host = strings.ReplaceAll(host, ":", "-") + ".sslip.io"
			}
			port := cmp.Or(tt.overridePort, grpcPort)

			grpcCfg := client.GRPCClientConfig{
				Host:     host,
				Port:     port,
				Insecure: tt.allowInsecure,
			}
			conn, err := cli.GetGrpcConnection(grpcCfg, authServer.URL, "stacklok", "minder-cli")

			if tt.expectedGRPC > 0 && tt.expectedGRPC != grpcCalls {
				t.Errorf("Expected %d grpc calls, got %d", tt.expectedAuths, authCalls)
			}
			if tt.expectedAuths > 0 && tt.expectedAuths != authCalls {
				t.Errorf("Expected %d auth calls, got %d", tt.expectedAuths, authCalls)
			}
			if tt.expectedLegacy > 0 && tt.expectedLegacy != legacyCalls {
				t.Errorf("Expected %d auth calls, got %d", tt.expectedLegacy, legacyCalls)
			}

			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
				return
			}
			if conn != nil {
				require.NoError(t, conn.Close())
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

	expectedFilePath := filepath.Join(cfgPath, "localhost_8081.json")

	filePath, err := cli.SaveCredentials(serverAddress, tokens)
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
//
//nolint:paralleltest
func TestRemoveCredentials(t *testing.T) {
	// Create a temporary directory
	testDir := t.TempDir()
	setEnvVar(t, XdgConfigHomeEnvVar, testDir)
	xdgConfigHome := os.Getenv(XdgConfigHomeEnvVar)

	filePath := filepath.Join(xdgConfigHome, "minder", "localhost_8081.json")

	// Create a dummy credentials file
	err := os.MkdirAll(filepath.Dir(filePath), 0750)

	if err != nil {
		t.Fatalf("error creating directory: %v", err)
	}

	err = os.WriteFile(filePath, []byte(`{"access_token":"test_access_token","refresh_token":"test_refresh_token","access_token_expires_at":1234567890}`), 0600)
	if err != nil {
		t.Fatalf("error writing credentials to file: %v", err)
	}

	err = cli.RemoveCredentials(serverAddress)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify the file is removed
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed, but it still exists")
	}
}

// TestRefreshCredentials tests the RefreshCredentials function
//
//nolint:paralleltest
func TestRefreshCredentials(t *testing.T) {
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, tt.responseBody)
			}))
			defer server.Close()

			tt.issuerUrl = server.URL

			result, err := cli.RefreshCredentials("localhost:8081", tt.refreshToken, tt.issuerUrl, tt.clientId)
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
//
//nolint:paralleltest
func TestLoadCredentials(t *testing.T) {
	tests := []struct {
		name           string
		filePath       string
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
			name:        "Load from default",
			filePath:    "minder/credentials.json",
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
			testDir := t.TempDir()
			setEnvVar(t, XdgConfigHomeEnvVar, testDir)
			// Create the minder directory inside the temp directory
			minderDir := filepath.Join(testDir, "minder")
			err := os.MkdirAll(minderDir, 0750)
			if err != nil {
				t.Fatalf("failed to create minder directory: %v", err)
			}

			filePath := filepath.Join(minderDir, "localhost_8081.json")
			if tt.filePath != "" {
				filePath = filepath.Join(testDir, tt.filePath)
			}

			if tt.fileContent != "" {
				// Create a temporary file with the specified content
				require.NoError(t, os.WriteFile(filePath, []byte(tt.fileContent), 0600))
				// Print the file path and content for debugging
				t.Logf("Test %s: written file path %s with content: %s", tt.name, filePath, tt.fileContent)
			} else {
				// Print the file path for debugging
				t.Logf("Test %s: file path %s not created as file content is empty", tt.name, filePath)
			}

			result, err := cli.LoadCredentials(serverAddress)
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

			err := cli.RevokeToken(tt.token, tt.issuerUrl, "stacklok", tt.clientId, tt.tokenHint)
			if (err != nil) != tt.expectError {
				t.Errorf("RevokeToken() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// fakeUserService implements just enough of UserService to bootstrap OAuth authentication
type fakeUserService struct {
	minderv1.UnimplementedUserServiceServer

	authUrl string
}

var _ minderv1.UserServiceServer = (*fakeUserService)(nil)

func (fus *fakeUserService) GetUser(ctx context.Context, _ *minderv1.GetUserRequest) (*minderv1.GetUserResponse, error) {
	err := grpc.SendHeader(ctx, metadata.New(map[string]string{
		"www-authenticate": fmt.Sprintf(`Bearer realm=%q, scope="minder"`, fus.authUrl),
	}))
	if err != nil {
		return nil, err
	}
	return nil, status.Error(codes.Unauthenticated, "unauthenticated")
}
