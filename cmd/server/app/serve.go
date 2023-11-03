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
	"net/url"
	"os"
	"os/signal"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/config"
	"github.com/stacklok/minder/internal/controlplane"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/logger"
	provtelemetry "github.com/stacklok/minder/internal/providers/telemetry"
	"github.com/stacklok/minder/internal/reconcilers"
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
		if cmd.Flag("dump_config").Value.String() == "true" {
			log.Printf("%+v\n", cfg)
			os.Exit(0)
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

		jwksUrl := parsedURL.JoinPath("realms", cfg.Identity.Server.Realm, "protocol/openid-connect/certs")
		vldtr, err := auth.NewJwtValidator(ctx, jwksUrl.String())
		if err != nil {
			return fmt.Errorf("failed to fetch and cache identity provider JWKS: %w\n", err)
		}

		err = controlplane.SubscribeToIdentityEvents(ctx, store, cfg)
		if err != nil {
			return fmt.Errorf("unable to subscribe to identity server events: %w", err)
		}

		serverMetrics := controlplane.NewMetrics()
		providerMetrics := provtelemetry.NewProviderMetrics()

		s, err := controlplane.NewServer(store, evt, serverMetrics, cfg, vldtr, controlplane.WithProviderMetrics(providerMetrics))
		if err != nil {
			return fmt.Errorf("unable to create server: %w", err)
		}

		exec, err := engine.NewExecutor(store, &cfg.Auth, engine.WithProviderMetrics(providerMetrics))
		if err != nil {
			return fmt.Errorf("unable to create executor: %w", err)
		}

		s.ConsumeEvents(exec)

		rec, err := reconcilers.NewReconciler(store, evt, &cfg.Auth, reconcilers.WithProviderMetrics(providerMetrics))
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

		return errg.Wait()
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)

	v := viper.GetViper()

	// Register flags for the server - http, grpc, metrics
	if err := config.RegisterServerFlags(v, serveCmd.Flags()); err != nil {
		log.Fatal(err)
	}

	serveCmd.Flags().String("logging", "", "Log Level")

	serveCmd.Flags().Bool("dump_config", false, "Dump Config and exit")
}
