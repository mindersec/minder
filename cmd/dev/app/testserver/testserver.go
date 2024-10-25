// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package testserver spawns a test server useful for integration testing.
package testserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/signal"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-github/v63/github"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/auth"
	noopauth "github.com/mindersec/minder/internal/auth/jwt/noop"
	mockauthz "github.com/mindersec/minder/internal/authz/mock"
	"github.com/mindersec/minder/internal/controlplane/metrics"
	"github.com/mindersec/minder/internal/metrics/meters"
	"github.com/mindersec/minder/internal/providers/ratecache"
	provtelemetry "github.com/mindersec/minder/internal/providers/telemetry"
	"github.com/mindersec/minder/internal/service"
	"github.com/mindersec/minder/pkg/config"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/db/embedded"
)

// CmdTestServer starts a test server for integration testing.
func CmdTestServer() *cobra.Command {
	var rtCmd = &cobra.Command{
		Use:   "testserver",
		Short: "testserver starts a test server for integration testing",
		RunE:  runTestServer,
	}

	return rtCmd
}

func runTestServer(cmd *cobra.Command, _ []string) error {
	v := viper.GetViper()
	serverconfig.SetViperDefaults(v)
	cfg, err := config.ReadConfigFromViper[serverconfig.Config](v)
	if err != nil {
		return fmt.Errorf("unable to read config: %w", err)
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer cancel()

	ctx = serverconfig.LoggerFromConfigFlags(cfg.LoggingConfig).WithContext(ctx)
	l := zerolog.Ctx(ctx)
	l.Info().Msgf("Initializing logger in level: %s", cfg.LoggingConfig.Level)

	store, td, err := embedded.GetFakeStore()
	if err != nil {
		return fmt.Errorf("unable to spawn embedded store: %w", err)
	}
	defer td()

	cfg.Events.Driver = "go-channel"
	cfg.Events.RouterCloseTimeout = 10
	cfg.Events.Aggregator.LockInterval = 30

	jwt := noopauth.NewJwtValidator("mindev")

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()
	packageListingClient := github.NewClient(http.DefaultClient)
	testServerUrl, err := url.Parse(testServer.URL + "/")
	if err != nil {
		return fmt.Errorf("unable to spawn embedded server: %w", err)
	}
	packageListingClient.BaseURL = testServerUrl

	return service.AllInOneServerService(
		ctx,
		cfg,
		store,
		jwt,
		&ratecache.NoopRestClientCache{},
		&mockauthz.SimpleClient{},
		&auth.IdentityClient{},
		metrics.NewNoopMetrics(),
		provtelemetry.NewNoopMetrics(),
		[]message.HandlerMiddleware{},
		&meters.NoopMeterFactory{},
	)
}
