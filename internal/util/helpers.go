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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

// Package util provides helper functions for the mediator CLI.
package util

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq" // nolint
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// GetConfigValue is a helper function that retrieves a configuration value
// and updates it if the corresponding flag is set.
//
// Parameters:
// - key: The key used to retrieve the configuration value from Viper.
// - flagName: The flag name used to check if the flag has been set and to retrieve its value.
// - cmd: The cobra.Command object to access the flags.
// - defaultValue: A default value used to determine the type of the flag (string, int, etc.).
//
// Returns:
// - The updated configuration value based on the flag, if it is set, or the original value otherwise.
func GetConfigValue(key string, flagName string, cmd *cobra.Command, defaultValue interface{}) interface{} {
	value := viper.Get(key)
	if cmd.Flags().Changed(flagName) {
		switch defaultValue.(type) {
		case string:
			value, _ = cmd.Flags().GetString(flagName)
		case int:
			value, _ = cmd.Flags().GetInt(flagName)
		case int32:
			value, _ = cmd.Flags().GetInt32(flagName)
		case bool:
			value, _ = cmd.Flags().GetBool(flagName)
			// add additional cases here for other types you need to handle
		}
	}
	if value != nil {
		return value
	}
	return defaultValue
}

// Credentials is a struct to hold the access and refresh tokens
type Credentials struct {
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	AccessTokenExpiresIn  int    `json:"access_token_expires_in"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
}

func getCredentialsPath() (string, error) {
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

	filePath := filepath.Join(xdgConfigHome, "mediator", "credentials.json")
	return filePath, nil
}

// JWTTokenCredentials is a helper struct for grpc
type JWTTokenCredentials struct {
	accessToken  string
	refreshToken string
}

// GetRequestMetadata implements the PerRPCCredentials interface.
func (jwt JWTTokenCredentials) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + string(jwt.accessToken),
		"refresh-token": jwt.refreshToken,
	}, nil
}

// RequireTransportSecurity implements the PerRPCCredentials interface.
func (JWTTokenCredentials) RequireTransportSecurity() bool {
	return false
}

// GrpcForCommand is a helper for getting a testing connection from cobra flags
func GrpcForCommand(cmd *cobra.Command) (*grpc.ClientConn, error) {
	grpc_host := GetConfigValue("grpc_server.host", "grpc-host", cmd, "staging.stacklok.dev").(string)
	grpc_port := GetConfigValue("grpc_server.port", "grpc-port", cmd, 443).(int)
	insecureDefault := grpc_host == "localhost" || grpc_host == "127.0.0.1" || grpc_host == "::1"
	allowInsecure := GetConfigValue("grpc_server.insecure", "grpc-insecure", cmd, insecureDefault).(bool)

	return GetGrpcConnection(grpc_host, grpc_port, allowInsecure)
}

// GetGrpcConnection is a helper for getting a testing connection for grpc
func GetGrpcConnection(grpc_host string, grpc_port int, allowInsecure bool) (*grpc.ClientConn, error) {
	address := fmt.Sprintf("%s:%d", grpc_host, grpc_port)

	// read credentials
	token := ""
	refreshToken := ""
	expirationTime := 0
	creds, err := LoadCredentials()
	if err == nil {
		token = creds.AccessToken
		refreshToken = creds.RefreshToken
		expirationTime = creds.RefreshTokenExpiresIn
	}

	credentialOpts := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS13})
	if allowInsecure {
		credentialOpts = insecure.NewCredentials()
	}

	// generate credentials
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(credentialOpts),
		grpc.WithPerRPCCredentials(JWTTokenCredentials{accessToken: token, refreshToken: refreshToken}))
	if err != nil {
		return nil, fmt.Errorf("error connecting to gRPC server: %v", err)
	}

	// NOTE: refresh is best effort. We will not error out if it fails
	// in the case of failure, the credentials won't be refreshed
	// and user will need to log in again

	// call to verify endpoint
	client := pb.NewAuthServiceClient(conn)
	ctx := context.Background()
	needsRefresh := false
	if token != "" {
		result, err := client.Verify(ctx, &pb.VerifyRequest{})
		if err != nil || result.Status == "KO" {
			needsRefresh = true
		}
	}

	if needsRefresh && refreshToken != "" {
		// call refresh endpoint
		result, err := client.RefreshToken(ctx, &pb.RefreshTokenRequest{})
		if err == nil {
			// combine the credentials and save them
			creds := Credentials{
				AccessToken:           result.AccessToken,
				RefreshToken:          refreshToken,
				AccessTokenExpiresIn:  int(result.AccessTokenExpiresIn),
				RefreshTokenExpiresIn: expirationTime,
			}

			// save credentials
			_, _ = SaveCredentials(creds)
		}
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
func SaveCredentials(creds Credentials) (string, error) {
	// marshal the credentials to json
	credsJSON, err := json.Marshal(creds)
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

// LoadCredentials loads the credentials from a file
func LoadCredentials() (Credentials, error) {
	filePath, err := getCredentialsPath()
	if err != nil {
		return Credentials{}, fmt.Errorf("error getting credentials path: %v", err)
	}

	// Read the file
	credsJSON, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return Credentials{}, fmt.Errorf("error reading credentials file: %v", err)
	}

	var creds Credentials
	err = json.Unmarshal(credsJSON, &creds)
	if err != nil {
		return Credentials{}, fmt.Errorf("error unmarshaling credentials: %v", err)
	}
	return creds, nil
}

// GetAppContext is a helper for getting the cmd app context
func GetAppContext() (context.Context, context.CancelFunc) {
	viper.SetDefault("cli.context_timeout", 5)
	timeout := viper.GetInt("cli.context_timeout")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	return ctx, cancel
}

// GetRandomPort returns a random port number.
// The binding address should not need to be configurable
// as this is a short lived operation just to disover a random available port.
// Note that there is a possible race condition here if another process binds
// to the same port between the time we discover it and the time we use it.
// This is unlikely to happen in practice, but if it does, the user will
// need to retry the command.
// Marking a nosec here because we want this to listen on all addresses to
// ensure a reliable connection chance for the client. This is based on lessons
// learned from the sigstore CLI.
func GetRandomPort() (int, error) {
	listener, err := net.Listen("tcp", ":0") // #nosec
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	return port, nil
}

// WriteToFile writes the content to a file if the out parameter is not empty.
func WriteToFile(out string, content []byte, perms fs.FileMode) error {
	if out != "" {
		err := os.WriteFile(out, content, perms)
		if err != nil {
			return fmt.Errorf("error writing to file: %s", err)
		}
	}

	return nil
}

// GetPassFromTerm gets a password from the terminal
func GetPassFromTerm(confirm bool) ([]byte, error) {
	fmt.Print("Enter password for private key: ")

	pw1, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, err
	}
	fmt.Println()

	if !confirm {
		return pw1, nil
	}

	fmt.Print("Enter password for private key again: ")
	confirmpw, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()

	if err != nil {
		return nil, err
	}

	if !bytesEqual(pw1, confirmpw) {
		return nil, errors.New("passwords do not match")
	}

	return pw1, nil
}

func bytesEqual(a, b []byte) bool {
	return strings.EqualFold(strings.TrimSpace(string(a)), strings.TrimSpace(string(b)))
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
	yamlResult, err := ConvertJsonToYaml(rawMsg)
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
