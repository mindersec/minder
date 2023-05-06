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

package app

import (
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stacklok/mediator/pkg/controlplane"
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
func getConfigValue(key string, flagName string, cmd *cobra.Command, defaultValue interface{}) interface{} {
	value := viper.Get(key)
	if cmd.Flags().Changed(flagName) {
		switch defaultValue.(type) {
		case string:
			value, _ = cmd.Flags().GetString(flagName)
		case int:
			value, _ = cmd.Flags().GetInt(flagName)
		}
	}
	return value
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the mediator platform",
	Long:  `Starts the mediator platform, which includes the gRPC server and the HTTP gateway.`,
	Run: func(cmd *cobra.Command, args []string) {
		// populate config and cmd line flags
		http_host := getConfigValue("http_server.host", "http-host", cmd, "").(string)
		http_port := getConfigValue("http_server.port", "http-port", cmd, 0).(int)
		grpc_host := getConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := getConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		// Database configuration
		dbhost := getConfigValue("database.dbhost", "db-host", cmd, "").(string)
		dbport := getConfigValue("database.dbport", "db-port", cmd, 0).(int)
		dbuser := getConfigValue("database.dbuser", "db-user", cmd, "").(string)
		dbpass := getConfigValue("database.dbpass", "db-pass", cmd, "").(string)
		dbname := getConfigValue("database.dbname", "db-name", cmd, "").(string)
		dbsslmode := getConfigValue("database.sslmode", "db-sslmode", cmd, "").(string)

		dBconn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbuser, dbpass, dbhost, strconv.Itoa(dbport), dbname, dbsslmode)

		// Set up the addresse strings
		httpAddress := fmt.Sprintf("%s:%d", http_host, http_port)
		grpcAddress := fmt.Sprintf("%s:%d", grpc_host, grpc_port)

		var wg sync.WaitGroup
		wg.Add(2)

		s := controlplane.Server{}

		// Start the gRPC and HTTP server in separate goroutines
		go func() {
			s.StartGRPCServer(grpcAddress, dBconn)
			wg.Done()
		}()

		go func() {
			controlplane.StartHTTPServer(httpAddress, grpcAddress)
			wg.Done()
		}()

		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().String("http-host", "", "Server host")
	serveCmd.Flags().Int("http-port", 0, "Server port")
	serveCmd.Flags().String("grpc-host", "", "Server host")
	serveCmd.Flags().Int("grpc-port", 0, "Server port")
	serveCmd.Flags().String("dbhost", "", "Database host")
	serveCmd.Flags().Int("dbport", 5432, "Database port")
	serveCmd.Flags().String("dbname", "", "Database name")
	serveCmd.Flags().String("dbuser", "", "Database user")
	serveCmd.Flags().String("dbpass", "", "Database password")
	serveCmd.Flags().String("dbsslmode", "", "Database sslmode")
	serveCmd.Flags().String("logging", "", "Log Level")
	if err := viper.BindPFlags(serveCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
