//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package util provides helper functions for the minder CLI.
package util

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	_ "github.com/signalfx/splunk-otel-go/instrumentation/github.com/lib/pq/splunkpq" // nolint
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/minder/internal/util/jsonyaml"
)

var (
	// PyRequestsVersionRegexp is a regexp to match a line in a requirements.txt file, including the package version
	// and the comparison operators
	PyRequestsVersionRegexp = regexp.MustCompile(`\s*(>=|<=|==|>|<|!=)\s*(\d+(\.\d+)*(\*)?)`)
	// PyRequestsNameRegexp is a regexp to match a line in a requirements.txt file, parsing out the package name
	PyRequestsNameRegexp = regexp.MustCompile(`\s*(>=|<=|==|>|<|!=)`)
	// MinderAuthTokenEnvVar is the environment variable for the minder auth token
	//nolint:gosec // This is not a hardcoded credential
	MinderAuthTokenEnvVar = "MINDER_AUTH_TOKEN"
	// ErrGettingRefreshToken is an error for when we can't get a refresh token
	ErrGettingRefreshToken = errors.New("error refreshing credentials")
)

// OpenIdCredentials is a struct to hold the access and refresh tokens
type OpenIdCredentials struct {
	AccessToken          string    `json:"access_token"`
	RefreshToken         string    `json:"refresh_token"`
	AccessTokenExpiresAt time.Time `json:"expiry"`
}

// GetConfigDirPath returns the path to the config directory
func GetConfigDirPath() (string, error) {
	// Get the XDG_CONFIG_HOME environment variable
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")

	// If XDG_CONFIG_HOME is not set or empty, use $HOME/.config as the base directory
	if xdgConfigHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("error getting home directory: %v", err)
		}
		xdgConfigHome = filepath.Join(homeDir, ".config")
	}

	filePath := filepath.Join(xdgConfigHome, "minder")
	return filePath, nil
}

func getCredentialsPath() (string, error) {
	cfgPath, err := GetConfigDirPath()
	if err != nil {
		return "", fmt.Errorf("error getting config path: %v", err)
	}

	filePath := filepath.Join(cfgPath, "credentials.json")
	return filePath, nil
}

// JWTTokenCredentials is a helper struct for grpc
type JWTTokenCredentials struct {
	accessToken string
}

// GetRequestMetadata implements the PerRPCCredentials interface.
func (jwt JWTTokenCredentials) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + string(jwt.accessToken),
	}, nil
}

// RequireTransportSecurity implements the PerRPCCredentials interface.
func (JWTTokenCredentials) RequireTransportSecurity() bool {
	return false
}

// GetGrpcConnection is a helper for getting a testing connection for grpc
func GetGrpcConnection(
	grpc_host string, grpc_port int,
	allowInsecure bool,
	issuerUrl string, clientId string,
	opts ...grpc.DialOption) (
	*grpc.ClientConn, error) {
	address := fmt.Sprintf("%s:%d", grpc_host, grpc_port)

	// read credentials
	token := ""
	if os.Getenv(MinderAuthTokenEnvVar) != "" {
		token = os.Getenv(MinderAuthTokenEnvVar)
	} else {
		t, err := GetToken(issuerUrl, clientId)
		if err == nil {
			token = t
		}
	}

	credentialOpts := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13})
	if allowInsecure {
		credentialOpts = insecure.NewCredentials()
	}

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentialOpts),
		grpc.WithPerRPCCredentials(JWTTokenCredentials{accessToken: token}),
	}
	dialOpts = append(dialOpts, opts...)

	// generate credentials
	conn, err := grpc.NewClient(address, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("error connecting to gRPC server: %v", err)
	}

	return conn, nil
}

// TestWriter is a helper struct for testing
type TestWriter struct {
	Output string
}

func (tw *TestWriter) Write(p []byte) (n int, err error) {
	tw.Output += string(p)
	return len(p), nil
}

// SaveCredentials saves the credentials to a file
func SaveCredentials(tokens OpenIdCredentials) (string, error) {
	// marshal the credentials to json
	credsJSON, err := json.Marshal(tokens)
	if err != nil {
		return "", fmt.Errorf("error marshaling credentials: %v", err)
	}

	filePath, err := getCredentialsPath()
	if err != nil {
		return "", fmt.Errorf("error getting credentials path: %v", err)
	}

	err = os.MkdirAll(filepath.Dir(filePath), 0750)
	if err != nil {
		return "", fmt.Errorf("error creating directory: %v", err)
	}

	// Write the JSON data to the file
	err = os.WriteFile(filePath, credsJSON, 0600)
	if err != nil {
		return "", fmt.Errorf("error writing credentials to file: %v", err)
	}
	return filePath, nil
}

// RemoveCredentials removes the local credentials file
func RemoveCredentials() error {
	// remove credentials file
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")

	// just delete token from credentials file
	if xdgConfigHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error getting home directory: %v", err)
		}
		xdgConfigHome = filepath.Join(homeDir, ".config")
	}

	filePath := filepath.Join(xdgConfigHome, "minder", "credentials.json")
	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("error removing credentials file: %v", err)
	}
	return nil
}

// GetToken retrieves the access token from the credentials file and refreshes it if necessary
func GetToken(issuerUrl string, clientId string) (string, error) {
	refreshLimit := 10
	creds, err := LoadCredentials()
	if err != nil {
		return "", fmt.Errorf("error loading credentials: %v", err)
	}
	needsRefresh := time.Now().Add(time.Duration(refreshLimit) * time.Second).After(creds.AccessTokenExpiresAt)

	if needsRefresh {
		updatedCreds, err := RefreshCredentials(creds.RefreshToken, issuerUrl, clientId)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrGettingRefreshToken, err)
		}
		return updatedCreds.AccessToken, nil
	}

	return creds.AccessToken, nil
}

type refreshTokenResponse struct {
	AccessToken          string `json:"access_token"`
	RefreshToken         string `json:"refresh_token"`
	AccessTokenExpiresIn int    `json:"expires_in"`
	// These will be present if there's an error
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// RefreshCredentials uses a refresh token to get and save a new set of credentials
func RefreshCredentials(refreshToken string, issuerUrl string, clientId string) (OpenIdCredentials, error) {

	parsedURL, err := url.Parse(issuerUrl)
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error parsing issuer URL: %v", err)
	}
	tokenUrl := parsedURL.JoinPath("realms/stacklok/protocol/openid-connect/token")

	data := url.Values{}
	data.Set("client_id", clientId)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequest("POST", tokenUrl.String(), strings.NewReader(data.Encode()))
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error creating: %v", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error fetching new credentials: %v", err)
	}
	defer resp.Body.Close()

	tokens := refreshTokenResponse{}
	err = json.NewDecoder(resp.Body).Decode(&tokens)
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error unmarshaling credentials: %v", err)
	}

	if tokens.Error != "" {
		return OpenIdCredentials{}, fmt.Errorf("error refreshing credentials: %s: %s", tokens.Error, tokens.ErrorDescription)
	}

	updatedCredentials := OpenIdCredentials{
		AccessToken:          tokens.AccessToken,
		RefreshToken:         tokens.RefreshToken,
		AccessTokenExpiresAt: time.Now().Add(time.Duration(tokens.AccessTokenExpiresIn) * time.Second),
	}
	_, err = SaveCredentials(updatedCredentials)
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error saving credentials: %v", err)
	}

	return updatedCredentials, nil
}

// LoadCredentials loads the credentials from a file
func LoadCredentials() (OpenIdCredentials, error) {
	filePath, err := getCredentialsPath()
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error getting credentials path: %v", err)
	}

	// Read the file
	credsJSON, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error reading credentials file: %v", err)
	}

	var creds OpenIdCredentials
	err = json.Unmarshal(credsJSON, &creds)
	if err != nil {
		return OpenIdCredentials{}, fmt.Errorf("error unmarshaling credentials: %v", err)
	}
	return creds, nil
}

// RevokeOfflineToken revokes the given offline token using OAuth2.0's Token Revocation endpoint
// from RFC 7009.
func RevokeOfflineToken(token string, issuerUrl string, clientId string) error {
	return RevokeToken(token, issuerUrl, clientId, "refresh_token")
}

// RevokeToken revokes the given token using OAuth2.0's Token Revocation endpoint
// from RFC 7009. The tokenHint is the type of token being revoked, such as
// "access_token" or "refresh_token". In the case of an offline token, the
// tokenHint should be "refresh_token".
func RevokeToken(token string, issuerUrl string, clientId string, tokenHint string) error {
	parsedURL, err := url.Parse(issuerUrl)
	if err != nil {
		return fmt.Errorf("error parsing issuer URL: %v", err)
	}
	logoutUrl := parsedURL.JoinPath("realms/stacklok/protocol/openid-connect/revoke")

	data := url.Values{}
	data.Set("client_id", clientId)
	data.Set("token", token)
	data.Set("token_type_hint", tokenHint)

	req, err := http.NewRequest("POST", logoutUrl.String(), strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("error creating: %v", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error revoking token: %v", err)
	}
	defer resp.Body.Close()

	return nil
}

func getProtoMarshalOptions() protojson.MarshalOptions {
	return protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}

}

// GetJsonFromProto given a proto message, formats into json
func GetJsonFromProto(msg protoreflect.ProtoMessage) (string, error) {
	m := getProtoMarshalOptions()
	out, err := m.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// GetYamlFromProto given a proto message, formats into yaml
func GetYamlFromProto(msg protoreflect.ProtoMessage) (string, error) {
	// first converts into json using the marshal options
	m := getProtoMarshalOptions()
	out, err := m.Marshal(msg)
	if err != nil {
		return "", err
	}

	// from byte, we get the raw message so we can convert into yaml
	var rawMsg json.RawMessage
	err = json.Unmarshal(out, &rawMsg)
	if err != nil {
		return "", err
	}
	yamlResult, err := jsonyaml.ConvertJsonToYaml(rawMsg)
	if err != nil {
		return "", err
	}
	return yamlResult, nil
}

// GetBytesFromProto given a proto message, formats into bytes
func GetBytesFromProto(message protoreflect.ProtoMessage) ([]byte, error) {
	m := getProtoMarshalOptions()
	return m.Marshal(message)
}

// OpenFileArg opens a file argument and returns a descriptor, closer, and error
// If the file is "-", it will return whatever is passed in as dashOpen and a no-op closer
func OpenFileArg(f string, dashOpen io.Reader) (desc io.Reader, closer func(), err error) {
	if f == "-" {
		desc = dashOpen
		closer = func() {}
		return desc, closer, nil
	}

	f = filepath.Clean(f)
	ftemp, err := os.Open(f)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening file: %w", err)
	}

	closer = func() {
		err := ftemp.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error closing file: %v\n", err)
		}
	}

	desc = ftemp
	return desc, closer, nil
}

// ExpandFileArgs expands a list of file arguments into a list of files.
// If the file list contains "-" or regular files, it will leave them as-is.
// If the file list contains directories, it will expand them into a list of files.
func ExpandFileArgs(files []string) ([]string, error) {
	var expandedFiles []string
	for _, f := range files {
		if f == "-" {
			expandedFiles = append(expandedFiles, f)
			continue
		}
		f = filepath.Clean(f)
		fi, err := os.Stat(f)
		if err != nil {
			return nil, fmt.Errorf("error getting file info: %w", err)
		}

		if fi.IsDir() {
			// expand directory
			err := filepath.Walk(f, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return fmt.Errorf("error walking directory: %w", err)
				}

				if !info.IsDir() {
					expandedFiles = append(expandedFiles, path)
				}

				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("error walking directory: %w", err)
			}
		} else {
			// add file
			expandedFiles = append(expandedFiles, f)
		}
	}

	return expandedFiles, nil
}

// Int32FromString converts a string to an int32
func Int32FromString(v string) (int32, error) {
	if v == "" {
		return 0, fmt.Errorf("cannot convert empty string to int")
	}

	// convert string to int
	asInt32, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("error converting string to int: %w", err)
	}
	if asInt32 > math.MaxInt32 || asInt32 < math.MinInt32 {
		return 0, fmt.Errorf("integer %d cannot fit into int32", asInt32)
	}
	// already validated overflow
	// nolint:gosec
	return int32(asInt32), nil
}

// ViperLogLevelToZerologLevel converts a viper log level to a zerolog log level
func ViperLogLevelToZerologLevel(viperLogLevel string) zerolog.Level {
	switch viperLogLevel {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel // Default to info level if the mapping is not found
	}
}
