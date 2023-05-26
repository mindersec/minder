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

package util

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	_ "github.com/lib/pq" // nolint
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// getConfigValue is a helper function that retrieves a configuration value
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

func GetGrpcConnection(cmd *cobra.Command) (*grpc.ClientConn, error) {
	// Database configuration
	grpc_host := GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
	grpc_port := GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)
	address := fmt.Sprintf("%s:%d", grpc_host, grpc_port)

	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("error connecting to gRPC server: %v", err)
	}
	return conn, nil
}

type TestWriter struct {
	Output string
}

func (tw *TestWriter) Write(p []byte) (n int, err error) {
	tw.Output += string(p)
	return len(p), nil
}
