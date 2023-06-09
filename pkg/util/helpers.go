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
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	_ "github.com/lib/pq" // nolint
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

// GetDbConnectionFromConfig is a helper to get a database connection from a viper config
func GetDbConnectionFromConfig(settings map[string]interface{}) (*sql.DB, error) {
	// Database configuration
	dbhost := settings["dbhost"].(string)
	dbport := settings["dbport"].(int)
	dbuser := settings["dbuser"].(string)
	dbpass := settings["dbpass"].(string)
	dbname := settings["dbname"].(string)
	dbsslmode := settings["sslmode"].(string)

	dbConn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbuser, dbpass, dbhost, strconv.Itoa(dbport), dbname, dbsslmode)
	conn, err := sql.Open("postgres", dbConn)
	if err != nil {
		log.Fatal("Cannot connect to DB: ", err)
	} else {
		log.Println("Connected to DB")
	}
	return conn, err
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
type JWTTokenCredentials string

// GetRequestMetadata implements the PerRPCCredentials interface.
func (jwt JWTTokenCredentials) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + string(jwt),
	}, nil
}

// RequireTransportSecurity implements the PerRPCCredentials interface.
func (_ JWTTokenCredentials) RequireTransportSecurity() bool {
	return false
}

// GetGrpcConnection is a helper for getting a testing connection for grpc
func GetGrpcConnection(cmd *cobra.Command) (*grpc.ClientConn, error) {
	// Database configuration
	grpc_host := GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
	grpc_port := GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)
	address := fmt.Sprintf("%s:%d", grpc_host, grpc_port)

	// read the credentials
	creds, err := LoadCredentials()
	if err != nil {
		return nil, fmt.Errorf("error loading credentials: %v", err)
	}

	// generate credentials

	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(JWTTokenCredentials(creds.AccessToken)))
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
