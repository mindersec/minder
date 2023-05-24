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
	"os"
	"strconv"

	_ "github.com/lib/pq" // nolint
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		}
	}
	if value != nil {
		return value
	}
	return defaultValue
}

func GetDbConnection(cmd *cobra.Command) (*sql.DB, error) {
	// Database configuration
	dbhost := GetConfigValue("database.dbhost", "db-host", cmd, "").(string)
	dbport := GetConfigValue("database.dbport", "db-port", cmd, 0).(int)
	dbuser := GetConfigValue("database.dbuser", "db-user", cmd, "").(string)
	dbpass := GetConfigValue("database.dbpass", "db-pass", cmd, "").(string)
	dbname := GetConfigValue("database.dbname", "db-name", cmd, "").(string)
	dbsslmode := GetConfigValue("database.sslmode", "db-sslmode", cmd, "").(string)

	dbConn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbuser, dbpass, dbhost, strconv.Itoa(dbport), dbname, dbsslmode)
	conn, err := sql.Open("postgres", dbConn)
	if err != nil {
		log.Fatal("Cannot connect to DB: ", err)
	} else {
		log.Println("Connected to DB")
	}
	return conn, err
}

type TestWriter struct {
	Output string
}

func (tw *TestWriter) Write(p []byte) (n int, err error) {
	tw.Output += string(p)
	return len(p), nil
}

func SetupConfigFile() string {
	configFile := "config.yaml"
	config := []byte(`logging: "info"`)
	err := os.WriteFile(configFile, config, 0o600)
	if err != nil {
		panic(err)
	}
	return configFile
}

func RemoveConfigFile(filename string) {
	os.Remove(filename)
}
