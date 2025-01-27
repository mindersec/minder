// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/util"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var (
	// envLock is a mutex to ensure that all the tests that run os.SetEnv("XDG_CONFIG...") need to be prevented from running at the same time as each other.
	envLock = &sync.Mutex{}
)

func setEnvVar(t *testing.T, env string, value string) {
	t.Helper() // Keep golangci-lint happy
	envLock.Lock()
	t.Cleanup(envLock.Unlock)

	originalEnvToken := os.Getenv(env)
	err := os.Setenv(env, value)
	if err != nil {
		t.Errorf("error setting %v: %v", env, err)
	}
	defer os.Setenv(env, originalEnvToken)
}

// TestGetConfigDirPath tests the GetConfigDirPath function
func TestGetConfigDirPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		envVar         string
		expectedPath   string
		expectingError bool
	}{
		{
			name:           "XDG_CONFIG_HOME set",
			envVar:         "/custom/config",
			expectedPath:   "/custom/config/minder",
			expectingError: false,
		},
		{
			name:           "XDG_CONFIG_HOME is not set",
			envVar:         "",
			expectedPath:   filepath.Join(os.Getenv("HOME"), ".config", "minder"),
			expectingError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			setEnvVar(t, "XDG_CONFIG_HOME", tt.envVar)
			path, err := util.GetConfigDirPath()
			if (err != nil) != tt.expectingError {
				t.Errorf("expected error: %v, got: %v", tt.expectingError, err)
			}
			if path != tt.expectedPath {
				t.Errorf("expected path: %s, got: %s", tt.expectedPath, path)
			}
		})
	}
}

// TestGetGrpcConnection tests the GetGrpcConnection function
func TestGetGrpcConnection(t *testing.T) {
	t.Parallel()
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
			grpcHost:      "127.0.0.1",
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
		{
			name:          "Valid GRPC connection to remote server with secure connection",
			grpcHost:      "api.stacklok.com",
			grpcPort:      443,
			allowInsecure: false,
			issuerUrl:     "https://auth.stacklok.com",
			clientId:      "minder-cli",
			envToken:      "MINDER_AUTH_TOKEN",
			expectedError: false,
		},
		{
			name:          "Valid GRPC connection to remote server with insecure connection",
			grpcHost:      "api.stacklok.com",
			grpcPort:      443,
			allowInsecure: true,
			issuerUrl:     "https://auth.stacklok.com",
			clientId:      "minder-cli",
			envToken:      "MINDER_AUTH_TOKEN",
			expectedError: false,
		},
		{
			name:          "Valid GRPC connection to remote server without passing MINDER_AUTH_TOKEN as an argument",
			grpcHost:      "api.stacklok.com",
			grpcPort:      443,
			allowInsecure: true,
			issuerUrl:     "https://auth.stacklok.com",
			clientId:      "minder-cli",
			envToken:      "",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			setEnvVar(t, util.MinderAuthTokenEnvVar, tt.envToken)

			conn, err := util.GetGrpcConnection(tt.grpcHost, tt.grpcPort, tt.allowInsecure, tt.issuerUrl, tt.clientId)
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

	tokens := util.OpenIdCredentials{
		AccessToken:          "test_access_token",
		RefreshToken:         "test_refresh_token",
		AccessTokenExpiresAt: time.Time{}.Add(7 * 24 * time.Hour),
	}

	// Create a temporary directory
	testDir := t.TempDir()

	setEnvVar(t, "XDG_CONFIG_HOME", testDir)

	cfgPath := filepath.Join(testDir, "minder")

	expectedFilePath := filepath.Join(cfgPath, "credentials.json")

	filePath, err := util.SaveCredentials(tokens)
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

	setEnvVar(t, "XDG_CONFIG_HOME", testDir)
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")

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

	err = util.RemoveCredentials()
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

	setEnvVar(t, "XDG_CONFIG_HOME", testDir)
	tests := []struct {
		name           string
		refreshToken   string
		issuerUrl      string
		clientId       string
		responseBody   string
		expectedError  string
		expectedResult util.OpenIdCredentials
	}{
		{
			name:         "Successful refresh with local server",
			refreshToken: "valid_refresh_token",
			clientId:     "minder-cli",
			responseBody: `{"access_token":"new_access_token","refresh_token":"new_refresh_token","expires_in":3600}`,
			expectedResult: util.OpenIdCredentials{
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

			result, err := util.RefreshCredentials(tt.refreshToken, tt.issuerUrl, tt.clientId)
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
		expectedResult util.OpenIdCredentials
	}{
		{
			name:        "Successful load",
			fileContent: `{"access_token":"access_token","refresh_token":"refresh_token","expiry":"2024-10-05T17:46:27+10:00"}`,
			expectedResult: util.OpenIdCredentials{
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
			setEnvVar(t, "XDG_CONFIG_HOME", testDir)
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

			result, err := util.LoadCredentials()
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

			err := util.RevokeToken(tt.token, tt.issuerUrl, tt.clientId, tt.tokenHint)
			if (err != nil) != tt.expectError {
				t.Errorf("RevokeToken() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// TestGetJsonFromProto tests the GetJsonFromProto function
func TestGetJsonFromProto(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		input         proto.Message
		expectedJson  string
		expectedError bool
	}{
		{
			name: "Valid proto message",
			input: &pb.Repository{
				Owner: "repoOwner",
				Name:  "repoName",
			},
			expectedJson: `{
  "owner": "repoOwner",
  "name": "repoName"
}`,
			expectedError: false,
		},
		{
			name:          "Nil proto message",
			input:         nil,
			expectedJson:  `{}`,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			jsonStr, err := util.GetJsonFromProto(tt.input)

			if (err != nil) != tt.expectedError {
				t.Errorf("GetJsonFromProto() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			// Normalize JSON strings by removing all whitespaces and new lines
			normalizedResult := strings.Join(strings.Fields(jsonStr), "")
			normalizedExpected := strings.Join(strings.Fields(tt.expectedJson), "")

			if normalizedResult != normalizedExpected {
				t.Errorf("GetJsonFromProto() = %v, expected %v", normalizedResult, normalizedExpected)
			}
		})
	}
}

// TestGetYamlFromProto tests the GetYamlFromProto function
func TestGetYamlFromProto(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		input         proto.Message
		expectedYaml  string
		expectedError bool
	}{
		{
			name: "Valid proto message",
			input: &pb.Repository{
				Owner: "repoOwner",
				Name:  "repoName",
			},
			expectedYaml: `name: repoName
owner: repoOwner
`,
			expectedError: false,
		},
		{
			name:  "Nil proto message",
			input: nil,
			expectedYaml: `{}
`,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			yamlResult, err := util.GetYamlFromProto(tt.input)
			if (err != nil) != tt.expectedError {
				t.Errorf("GetYamlFromProto() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			// Normalize JSON strings by removing all whitespaces and new lines
			normalizedResult := strings.Join(strings.Fields(yamlResult), "")
			normalizedExpected := strings.Join(strings.Fields(tt.expectedYaml), "")

			if normalizedResult != normalizedExpected {
				t.Errorf("GetJsonFromProto() = %v, expected %v", normalizedResult, normalizedExpected)
			}
		})
	}
}

// TestOpenFileArg tests the OpenFileArg function
func TestOpenFileArg(t *testing.T) {
	t.Parallel()

	testDir := t.TempDir()
	tempFilePath := filepath.Join(testDir, "testfile.txt")
	err := os.WriteFile(tempFilePath, []byte("test content"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testCases := []struct {
		name         string
		filePath     string
		dashOpen     io.Reader
		expectedDesc string
		expectError  bool
	}{
		{
			name:         "Dash as file path",
			filePath:     "-",
			dashOpen:     strings.NewReader("dash input"),
			expectedDesc: "dash input",
			expectError:  false,
		},
		{
			name:         "Valid file path",
			filePath:     tempFilePath,
			dashOpen:     nil,
			expectedDesc: "test content",
			expectError:  false,
		},
		{
			name:         "Invalid file path",
			filePath:     "nonexistent.txt",
			dashOpen:     nil,
			expectedDesc: "",
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			desc, closer, err := util.OpenFileArg(tc.filePath, tc.dashOpen)
			if closer != nil {
				defer closer()
			}

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				buf := new(strings.Builder)
				_, err := io.Copy(buf, desc)
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedDesc, buf.String())
			}
		})
	}
}

// TestExpandFileArgs tests the ExpandFileArgs function.
func TestExpandFileArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		files    []string
		expected []util.ExpandedFile
		wantErr  bool
	}{
		{
			name:     "Single file",
			files:    []string{"testfile.txt"},
			expected: []util.ExpandedFile{{Path: "testfile.txt", Expanded: false}},
			wantErr:  false,
		},

		{
			name:     "Single directory",
			files:    []string{"testdir"},
			expected: []util.ExpandedFile{{Path: "testdir/file1.txt", Expanded: true}, {Path: "testdir/file2.txt", Expanded: true}},
			wantErr:  false,
		},
		{
			name:     "File and directory",
			files:    []string{"testfile.txt", "testdir"},
			expected: []util.ExpandedFile{{Path: "testfile.txt", Expanded: false}, {Path: "testdir/file1.txt", Expanded: true}, {Path: "testdir/file2.txt", Expanded: true}},
			wantErr:  false,
		},
		{
			name:     "File with '-'",
			files:    []string{"-"},
			expected: []util.ExpandedFile{{Path: "-", Expanded: false}},
			wantErr:  false,
		},
		{
			name:     "Non-existent file",
			files:    []string{"nonexistent.txt"},
			expected: nil,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a temporary directory
			testDir := t.TempDir()
			if err := setupTestFiles(testDir); err != nil {
				t.Fatalf("Failed to set up test files: %v", err)
			}
			// Update file paths to include the unique directory
			for i, file := range tt.files {
				tt.files[i] = fmt.Sprintf("%s/%s", testDir, file)
			}
			for i, file := range tt.expected {
				tt.expected[i].Path = fmt.Sprintf("%s/%s", testDir, file.Path)
			}
			combinedFiles := tt.files
			got, err := util.ExpandFileArgs(combinedFiles...)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExpandFileArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !slices.EqualFunc(got, tt.expected, func(a, b util.ExpandedFile) bool {
				return a.Path == b.Path && a.Expanded == b.Expanded
			}) {
				t.Errorf("ExpandFileArgs() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// setupTestFiles creates test files and directories for the unit tests.
func setupTestFiles(testDir string) error {
	if err := os.MkdirAll(fmt.Sprintf("%s/testdir", testDir), 0750); err != nil {
		return fmt.Errorf("failed to create directory '%s/testdir': %w", testDir, err)
	}

	if err := os.WriteFile(fmt.Sprintf("%s/testfile.txt", testDir), []byte("test file"), 0600); err != nil {
		return fmt.Errorf("failed to create file '%s/testfile.txt': %w", testDir, err)
	}

	if err := os.WriteFile(fmt.Sprintf("%s/testdir/file1.txt", testDir), []byte("file 1"), 0600); err != nil {
		return fmt.Errorf("failed to create file '%s/testdir/file1.txt': %w", testDir, err)
	}

	if err := os.WriteFile(fmt.Sprintf("%s/testdir/file2.txt", testDir), []byte("file 2"), 0600); err != nil {
		return fmt.Errorf("failed to create file '%s/testdir/file2.txt': %w", testDir, err)
	}

	// Create a file named "-"
	if err := os.WriteFile(fmt.Sprintf("%s/-", testDir), []byte("dash file"), 0600); err != nil {
		return fmt.Errorf("failed to create file '%s/-': %w", testDir, err)
	}

	return nil
}

func TestGetConfigValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		key          string
		flagName     string
		defaultValue interface{}
		flagValue    interface{}
		flagSet      bool
		expected     interface{}
	}{
		{
			name:         "string flag set",
			key:          "testString",
			flagName:     "test-string",
			defaultValue: "default",
			flagValue:    "newValue",
			flagSet:      true,
			expected:     "newValue",
		},
		{
			name:         "int flag set",
			key:          "testInt",
			flagName:     "test-int",
			defaultValue: 1,
			flagValue:    42,
			flagSet:      true,
			expected:     42,
		},
		{
			name:         "flag not set",
			key:          "testFlagNotSet",
			flagName:     "test-notset",
			defaultValue: "default",
			flagValue:    "",
			flagSet:      false,
			expected:     "default",
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			v := viper.New()

			v.SetDefault(tc.key, tc.defaultValue)

			cmd := &cobra.Command{}
			switch tc.defaultValue.(type) {
			case string:
				cmd.Flags().String(tc.flagName, tc.defaultValue.(string), "")
			case int:
				cmd.Flags().Int(tc.flagName, tc.defaultValue.(int), "")
			}
			// bind the flag to viper
			err := v.BindPFlag(tc.key, cmd.Flags().Lookup(tc.flagName))
			if err != nil {
				t.Fatalf("Error binding flag %s: %v", tc.flagName, err)
			}
			if tc.flagSet {
				switch tc.flagValue.(type) {
				case string:
					err := cmd.Flags().Set(tc.flagName, tc.flagValue.(string))
					if err != nil {
						t.Fatalf("Error setting flag %s: %v", tc.flagName, err)
					}
				case int:
					err := cmd.Flags().Set(tc.flagName, strconv.Itoa(tc.flagValue.(int)))
					if err != nil {
						t.Fatalf("Error setting flag %s: %v", tc.flagName, err)
					}
				}
			}

			result := v.Get(tc.key)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestInt32FromString(t *testing.T) {
	t.Parallel()

	type args struct {
		v string
	}
	tests := []struct {
		name    string
		args    args
		want    int32
		wantErr bool
	}{
		{
			name:    "valid int32",
			args:    args{v: "42"},
			want:    42,
			wantErr: false,
		},
		{
			name:    "valid int32 negative",
			args:    args{v: "-42"},
			want:    -42,
			wantErr: false,
		},
		{
			name:    "big int32",
			args:    args{v: "2147483647"},
			want:    2147483647,
			wantErr: false,
		},
		{
			name:    "big int32 negative",
			args:    args{v: "-2147483648"},
			want:    -2147483648,
			wantErr: false,
		},
		{
			name:    "too big int32",
			args:    args{v: "12147483648"},
			want:    0,
			wantErr: true,
		},
		{
			name:    "valid zero",
			args:    args{v: "0"},
			want:    0,
			wantErr: false,
		},
		{
			name:    "invalid int32",
			args:    args{v: "invalid"},
			want:    0,
			wantErr: true,
		},
		{
			name:    "empty string",
			args:    args{v: ""},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := util.Int32FromString(tt.args.v)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Zero(t, got, "expected zero")
				return
			}

			assert.Equal(t, tt.want, got, "result didn't match")
		})
	}
}
