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

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/internal/engine"
	"github.com/stacklok/mediator/internal/logger"
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

		ctx = logger.FromFlags(cfg.LoggingConfig).WithContext(ctx)
		zerolog.Ctx(ctx).Info().Msgf("Initializing logger in level: %s", cfg.LoggingConfig.Level)

		// Database configuration
		dbConn, _, err := cfg.Database.GetDBConnection(ctx)
		if err != nil {
			return fmt.Errorf("unable to connect to database: %w", err)
		}
		defer dbConn.Close()

		store := db.NewStore(dbConn)

		errg, ctx := errgroup.WithContext(ctx)

		s, err := controlplane.NewServer(store, cfg)
		if err != nil {
			return fmt.Errorf("unable to create server: %w", err)
		}

		s.ConsumeEvents(engine.NewExecutor(store))

		// Start the gRPC and HTTP server in separate goroutines
		errg.Go(func() error {
			return s.StartGRPCServer(ctx)
		})

		errg.Go(func() error {
			return s.StartHTTPServer(ctx)
		})

		errg.Go(s.HandleEvents(ctx))

		return errg.Wait()
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)

	v := viper.GetViper()

	if err := config.RegisterHTTPServerFlags(v, serveCmd.Flags()); err != nil {
		log.Fatal(err)
	}

	if err := config.RegisterGRPCServerFlags(v, serveCmd.Flags()); err != nil {
		log.Fatal(err)
	}

	serveCmd.Flags().String("logging", "", "Log Level")
}
