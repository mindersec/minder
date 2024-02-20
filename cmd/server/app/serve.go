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
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"os/signal"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/authz"
	"github.com/stacklok/minder/internal/config"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/eea"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/providers/ratecache"
	provtelemetry "github.com/stacklok/minder/internal/providers/telemetry"
	"github.com/stacklok/minder/internal/reconcilers"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the minder platform",
	Long:  `Starts the minder platform, which includes the gRPC server and the HTTP gateway.`,
	RunE: func(cmd *cobra.Command, _ []string) error {

		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer cancel()

		cfg, err := config.ReadConfigFromViper[serverconfig.Config](viper.GetViper())
		if err != nil {
			return fmt.Errorf("unable to read config: %w", err)
		}
		if cmd.Flag("dump_config").Value.String() == "true" {
			log.Printf("%+v\n", cfg)
			os.Exit(0)
		}

		ctx = logger.FromFlags(cfg.LoggingConfig).WithContext(ctx)
		l := zerolog.Ctx(ctx)
		l.Info().Msgf("Initializing logger in level: %s", cfg.LoggingConfig.Level)

		// Database configuration
		dbConn, _, err := cfg.Database.GetDBConnection(ctx)
		if err != nil {
			return fmt.Errorf("unable to connect to database: %w", err)
		}
		defer func(dbConn *sql.DB) {
			err := dbConn.Close()
			if err != nil {
				log.Printf("error closing database connection: %v", err)
			}
		}(dbConn)

		store := db.NewStore(dbConn)

		errg, ctx := errgroup.WithContext(ctx)

		evt, err := events.Setup(ctx, &cfg.Events)
		if err != nil {
			log.Printf("Failed to set up eventer: %v", err)
			return err
		}

		// webhook config validation
		webhookURL := cfg.WebhookConfig.ExternalWebhookURL
		webhookping := cfg.WebhookConfig.ExternalPingURL
		webhooksecret := cfg.WebhookConfig.WebhookSecret
		if webhookURL == "" || webhookping == "" || webhooksecret == "" {
			return fmt.Errorf("webhook configuration is not set")
		}

		// Identity
		parsedURL, err := url.Parse(cfg.Identity.Server.IssuerUrl)
		if err != nil {
			return fmt.Errorf("failed to parse issuer URL: %w\n", err)
		}

		jwksUrl := parsedURL.JoinPath("realms/stacklok/protocol/openid-connect/certs")
		vldtr, err := auth.NewJwtValidator(ctx, jwksUrl.String())
		if err != nil {
			return fmt.Errorf("failed to fetch and cache identity provider JWKS: %w\n", err)
		}

		authzc, err := authz.NewAuthzClient(&cfg.Authz, l)
		if err != nil {
			return fmt.Errorf("unable to create authz client: %w", err)
		}

		if err := authzc.PrepareForRun(ctx); err != nil {
			return fmt.Errorf("unable to prepare authz client for run: %w", err)
		}

		err = controlplane.SubscribeToIdentityEvents(ctx, store, authzc, cfg)
		if err != nil {
			return fmt.Errorf("unable to subscribe to identity server events: %w", err)
		}

		serverMetrics := controlplane.NewMetrics()
		providerMetrics := provtelemetry.NewProviderMetrics()
		restClientCache := ratecache.NewRestClientCache(ctx)
		defer restClientCache.Close()

		s, err := controlplane.NewServer(
			store, evt, serverMetrics, cfg, vldtr,
			controlplane.WithProviderMetrics(providerMetrics),
			controlplane.WithAuthzClient(authzc),
			controlplane.WithRestClientCache(restClientCache),
		)
		if err != nil {
			return fmt.Errorf("unable to create server: %w", err)
		}

		aggr := eea.NewEEA(store, evt, &cfg.Events.Aggregator)

		s.ConsumeEvents(aggr)

		tsmdw := logger.NewTelemetryStoreWMMiddleware(l)

		exec, err := engine.NewExecutor(ctx, store, &cfg.Auth, evt,
			engine.WithProviderMetrics(providerMetrics),
			engine.WithMiddleware(aggr.AggregateMiddleware),
			engine.WithMiddleware(tsmdw.TelemetryStoreMiddleware),
			engine.WithRestClientCache(restClientCache),
		)
		if err != nil {
			return fmt.Errorf("unable to create executor: %w", err)
		}

		s.ConsumeEvents(exec)

		rec, err := reconcilers.NewReconciler(store, evt, &cfg.Auth,
			reconcilers.WithProviderMetrics(providerMetrics),
			reconcilers.WithRestClientCache(restClientCache))
		if err != nil {
			return fmt.Errorf("unable to create reconciler: %w", err)
		}

		s.ConsumeEvents(rec)

		// Start the gRPC and HTTP server in separate goroutines
		errg.Go(func() error {
			return s.StartGRPCServer(ctx)
		})

		errg.Go(func() error {
			return s.StartHTTPServer(ctx)
		})

		errg.Go(s.HandleEvents(ctx))

		// Wait for event handlers to start running
		<-evt.Running()

		if err := aggr.FlushAll(ctx); err != nil {
			return fmt.Errorf("error flushing cache: %w", err)
		}

		// Wait for all entity events to be executed
		exec.Wait()

		return errg.Wait()
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)

	v := viper.GetViper()

	// Register flags for the server - http, grpc, metrics
	if err := serverconfig.RegisterServerFlags(v, serveCmd.Flags()); err != nil {
		log.Fatal().Err(err).Msg("Error registering server flags")
	}

	serveCmd.Flags().String("logging", "", "Log Level")

	serveCmd.Flags().Bool("dump_config", false, "Dump Config and exit")
}
