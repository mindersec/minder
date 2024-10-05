// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/util"
)
	"google.golang.org/protobuf/proto"
)

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
			os.Setenv("XDG_CONFIG_HOME", tt.envVar)
			path, err := GetConfigDirPath()
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
			os.Setenv(MinderAuthTokenEnvVar, tt.envToken)
			conn, err := GetGrpcConnection(tt.grpcHost, tt.grpcPort, tt.allowInsecure, tt.issuerUrl, tt.clientId)
			if (err != nil) != tt.expectedError {
				t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
			}
			if conn != nil {
				conn.Close()
			}
		})
	}
}

// TestSaveCredentials tests the SaveCredentials function
func TestSaveCredentials(t *testing.T) {
	tokens := OpenIdCredentials{
		AccessToken:          "test_access_token",
		RefreshToken:         "test_refresh_token",
		AccessTokenExpiresAt: time.Time{}.Add(7 * 24 * time.Hour),
	}

	cfgPath, err := GetConfigDirPath()

	if err != nil {
		t.Fatalf("error getting config path: %v", err)
	}

	expectedFilePath := filepath.Join(cfgPath, "credentials.json")

	filePath, err := SaveCredentials(tokens)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if filePath != expectedFilePath {
		t.Errorf("expected file path %v, got %v", expectedFilePath, filePath)
	}

	// Verify the file content
	credsJSON, err := json.Marshal(tokens)
	if err != nil {
		t.Fatalf("error marshaling credentials: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("error reading file: %v", err)
	}

	if string(content) != string(credsJSON) {
		t.Errorf("expected file content %v, got %v", string(credsJSON), string(content))
	}

	// Clean up
	os.Remove(filePath)
}

// TestRemoveCredentials tests the RemoveCredentials function
func TestRemoveCredentials(t *testing.T) {
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("error getting home directory: %v", err)
		}
		xdgConfigHome = filepath.Join(homeDir, ".config")
	}

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

	err = RemoveCredentials()
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
	tests := []struct {
		name           string
		refreshToken   string
		issuerUrl      string
		clientId       string
		responseBody   string
		expectedError  string
		expectedResult OpenIdCredentials
	}{
		{
			name:         "Successful refresh with local server",
			refreshToken: "valid_refresh_token",
			issuerUrl:    "http://localhost:8081",
			clientId:     "minder-cli",
			responseBody: `{"access_token":"new_access_token","refresh_token":"new_refresh_token","expires_in":3600}`,
			expectedResult: OpenIdCredentials{
				AccessToken:          "new_access_token",
				RefreshToken:         "new_refresh_token",
				AccessTokenExpiresAt: time.Now().Add(3600 * time.Second),
			},
		},
		{
			name:          "Error fetching new credentials (responseBody is missing) rwith local server",
			refreshToken:  "valid_refresh_token",
			issuerUrl:     "http://localhost:8081",
			clientId:      "minder-cli",
			expectedError: "error unmarshaling credentials: EOF",
		},
		{
			name:          "Error unmarshaling credentials with local server",
			refreshToken:  "valid_refresh_token",
			issuerUrl:     "http://localhost:8081",
			clientId:      "minder-cli",
			responseBody:  `invalid_json`,
			expectedError: "error unmarshaling credentials: invalid character 'i' looking for beginning of value",
		},
		{
			name:          "Error refreshing credentials with local server",
			refreshToken:  "valid_refresh_token",
			issuerUrl:     "http://localhost:8081",
			clientId:      "minder-cli",
			responseBody:  `{"error":"invalid_grant","error_description":"Invalid refresh token"}`,
			expectedError: "error refreshing credentials: invalid_grant: Invalid refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, tt.responseBody)
			}))
			defer server.Close()

			parsedURL, _ := url.Parse(server.URL)
			tt.issuerUrl = parsedURL.String()

			result, err := RefreshCredentials(tt.refreshToken, tt.issuerUrl, tt.clientId)
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
}

// TestRevokeToken tests the RevokeToken function
func TestRevokeToken(t *testing.T) {

}

// TestGetJsonFromProto tests the GetJsonFromProto function
func TestGetJsonFromProto(t *testing.T) {
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
			jsonResult, err := GetJsonFromProto(tt.input)
			if (err != nil) != tt.expectedError {
				t.Errorf("GetJsonFromProto() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if jsonResult != tt.expectedJson {
				t.Errorf("GetJsonFromProto() = %v, expected %v", jsonResult, tt.expectedJson)
			}
		})
	}
}

// TestGetYamlFromProto tests the GetYamlFromProto function
func TestGetYamlFromProto(t *testing.T) {
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
			yamlResult, err := GetYamlFromProto(tt.input)
			if (err != nil) != tt.expectedError {
				t.Errorf("GetYamlFromProto() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if yamlResult != tt.expectedYaml {
				t.Errorf("GetYamlFromProto() = %v, expected %v", yamlResult, tt.expectedYaml)
			}
		})
	}
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
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Int32FromString(tt.args.v)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Zero(t, got, "expected zero")
				return
			}

			assert.Equal(t, tt.want, got, "result didn't match")
		})
	}
}
