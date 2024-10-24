// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"database/sql"
	"fmt"
	"os"
	"os/signal"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/auth/jwt"
	"github.com/mindersec/minder/internal/auth/keycloak"
	"github.com/mindersec/minder/internal/authz"
	cpmetrics "github.com/mindersec/minder/internal/controlplane/metrics"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/logger"
	"github.com/mindersec/minder/internal/metrics/meters"
	"github.com/mindersec/minder/internal/providers/ratecache"
	provtelemetry "github.com/mindersec/minder/internal/providers/telemetry"
	"github.com/mindersec/minder/internal/service"
	"github.com/mindersec/minder/pkg/config"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
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

		ctx = serverconfig.LoggerFromConfigFlags(cfg.LoggingConfig).WithContext(ctx)
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

		// webhook config validation
		webhookURL := cfg.WebhookConfig.ExternalWebhookURL
		webhookping := cfg.WebhookConfig.ExternalPingURL
		webhooksecret, err := cfg.WebhookConfig.GetWebhookSecret()
		if err != nil {
			return fmt.Errorf("failed to get webhook secret: %w", err)
		}
		if webhookURL == "" || webhookping == "" || webhooksecret == "" {
			return fmt.Errorf("webhook configuration is not set")
		}

		// Identity
		// TODO: cfg.Identity.Server.IssuerUrl _should_ be a URL to an issuer that has an
		// .../.well-known/jwks.json or .../.well-known/openid-configuration endpoint.  Right
		// now it's just a hostname.  When we have this, we can consolidate the jwksUrl and issUrl,
		// and remove the Keycloak-specific paths.
		jwksUrl, err := cfg.Identity.Server.Path("/realms/stacklok/protocol/openid-connect/certs")
		if err != nil {
			return fmt.Errorf("failed to create JWKS URL: %w\n", err)
		}
		issUrl, err := cfg.Identity.Server.JwtUrl()
		if err != nil {
			return fmt.Errorf("failed to create issuer URL: %w\n", err)
		}
		jwt, err := jwt.NewJwtValidator(ctx, jwksUrl.String(), issUrl.String(), cfg.Identity.Server.Audience)
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

		kc, err := keycloak.NewKeyCloak("", cfg.Identity.Server)
		if err != nil {
			return fmt.Errorf("unable to create keycloak identity provider: %w", err)
		}
		idClient, err := auth.NewIdentityClient(kc)
		if err != nil {
			return fmt.Errorf("unable to create identity client: %w", err)
		}

		providerMetrics := provtelemetry.NewProviderMetrics()
		restClientCache := ratecache.NewRestClientCache(ctx)
		defer restClientCache.Close()

		telemetryMiddleware := logger.NewTelemetryStoreWMMiddleware(l)
		return service.AllInOneServerService(
			ctx,
			cfg,
			store,
			jwt,
			restClientCache,
			authzc,
			idClient,
			cpmetrics.NewMetrics(),
			providerMetrics,
			[]message.HandlerMiddleware{telemetryMiddleware.TelemetryStoreMiddleware},
			&meters.ExportingMeterFactory{},
		)
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
