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
	"os"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stacklok/mediator/pkg/controlplane"
	"github.com/stacklok/mediator/pkg/db"
	"github.com/stacklok/mediator/pkg/util"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the mediator platform",
	Long:  `Starts the mediator platform, which includes the gRPC server and the HTTP gateway.`,
	Run: func(cmd *cobra.Command, args []string) {
		// populate config and cmd line flags
		http_host := util.GetConfigValue("http_server.host", "http-host", cmd, "").(string)
		http_port := util.GetConfigValue("http_server.port", "http-port", cmd, 8080).(int)
		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 8090).(int)

		// Database configuration
		dbConn, _, err := util.GetDbConnectionFromConfig(cmd)
		if err != nil {
			fmt.Printf("Unable to connect to database: %v\n", err)
			os.Exit(1)
		}
		defer dbConn.Close()

		store := db.NewStore(dbConn)

		// Set up the addresse strings
		httpAddress := fmt.Sprintf("%s:%d", http_host, http_port)
		grpcAddress := fmt.Sprintf("%s:%d", grpc_host, grpc_port)

		var wg sync.WaitGroup
		wg.Add(2)

		s := controlplane.Server{}

		// Start the gRPC and HTTP server in separate goroutines
		go func() {
			s.StartGRPCServer(grpcAddress, store)
			wg.Done()
		}()

		go func() {
			controlplane.StartHTTPServer(httpAddress, grpcAddress, store)
			wg.Done()
		}()

		wg.Wait()
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)
	serveCmd.Flags().String("http-host", "", "Server host")
	serveCmd.Flags().Int("http-port", 0, "Server port")
	serveCmd.Flags().String("grpc-host", "", "Server host")
	serveCmd.Flags().Int("grpc-port", 0, "Server port")
	serveCmd.Flags().String("db-host", "", "Database host")
	serveCmd.Flags().Int("db-port", 5432, "Database port")
	serveCmd.Flags().String("db-name", "", "Database name")
	serveCmd.Flags().String("db-user", "", "Database user")
	serveCmd.Flags().String("db-pass", "", "Database password")
	serveCmd.Flags().String("db-sslmode", "", "Database sslmode")
	serveCmd.Flags().String("logging", "", "Log Level")
	if err := viper.BindPFlags(serveCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
