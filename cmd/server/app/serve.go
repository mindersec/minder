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
	"os/signal"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/pkg/controlplane"
	"github.com/stacklok/mediator/pkg/db"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the mediator platform",
	Long:  `Starts the mediator platform, which includes the gRPC server and the HTTP gateway.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer cancel()

		cfg, err := config.ReadConfigFromViper(viper.GetViper())
		if err != nil {
			return fmt.Errorf("unable to read config: %w", err)
		}

		// Database configuration
		dbConn, _, err := cfg.Database.GetDBConnection()
		if err != nil {
			return fmt.Errorf("unable to connect to database: %w", err)
		}
		defer dbConn.Close()

		store := db.NewStore(dbConn)

		// Set up the addresse strings
		httpAddress := fmt.Sprintf("%s:%d", cfg.HTTPServer.Host, cfg.HTTPServer.Port)
		grpcAddress := fmt.Sprintf("%s:%d", cfg.GRPCServer.Host, cfg.GRPCServer.Port)

		errg, ctx := errgroup.WithContext(ctx)

		s := controlplane.Server{}

		// Start the gRPC and HTTP server in separate goroutines
		errg.Go(func() error {
			return s.StartGRPCServer(ctx, grpcAddress, store)
		})

		errg.Go(func() error {
			return controlplane.StartHTTPServer(ctx, httpAddress, grpcAddress, store)
		})

		return errg.Wait()
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)
	v := viper.GetViper()
	if err := controlplane.RegisterHTTPServerFlags(v, serveCmd.Flags()); err != nil {
		log.Fatal(err)
	}

	if err := controlplane.RegisterGRPCServerFlags(v, serveCmd.Flags()); err != nil {
		log.Fatal(err)
	}

	serveCmd.Flags().String("logging", "", "Log Level")
}
