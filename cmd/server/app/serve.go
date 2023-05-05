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
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stacklok/mediator/pkg/controlplane"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the mediator platform",
	Long:  `Starts the mediator platform, which includes the gRPC server and the HTTP gateway.`,
	Run: func(cmd *cobra.Command, args []string) {
		http_host := viper.GetString("http_server.host")
		http_port := viper.GetInt("http_server.port")
		grpc_host := viper.GetString("grpc_server.host")
		grpc_port := viper.GetInt("grpc_server.port")

		// If the user has specified a flag, use that value
		// instead of the one set within the config file
		if cmd.Flags().Changed("http-host") {
			http_host, _ = cmd.Flags().GetString("http-host")
		}
		if cmd.Flags().Changed("http-port") {
			http_port, _ = cmd.Flags().GetInt("http-port")
		}
		if cmd.Flags().Changed("grpc-host") {
			grpc_host, _ = cmd.Flags().GetString("grpc-host")
		}
		if cmd.Flags().Changed("grpc-port") {
			grpc_port, _ = cmd.Flags().GetInt("grpc-port")
		}

		httpAddress := fmt.Sprintf("%s:%d", http_host, http_port)
		grpcAddress := fmt.Sprintf("%s:%d", grpc_host, grpc_port)

		var wg sync.WaitGroup
		wg.Add(2)

		s := controlplane.Server{}

		// Start the gRPC and HTTP server in separate goroutines
		go func() {
			s.StartGRPCServer(grpcAddress)
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
	serveCmd.PersistentFlags().String("http-host", "", "Server host")
	serveCmd.PersistentFlags().Int("http-port", 0, "Server port")
	serveCmd.PersistentFlags().String("grpc-host", "", "Server host")
	serveCmd.PersistentFlags().Int("grpc-port", 0, "Server port")
	serveCmd.PersistentFlags().String("logging", "", "Log Level")
	if err := viper.BindPFlags(serveCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
