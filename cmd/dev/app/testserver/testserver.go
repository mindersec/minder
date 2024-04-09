// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package testserver spawns a test server useful for integration testing.
package testserver

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/signal"

	"github.com/google/go-github/v61/github"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	noopauth "github.com/stacklok/minder/internal/auth/noop"
	mockauthz "github.com/stacklok/minder/internal/authz/mock"
	"github.com/stacklok/minder/internal/config"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane"
	"github.com/stacklok/minder/internal/db/embedded"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/reconcilers"
	"github.com/stacklok/minder/internal/service"
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

	ctx = logger.FromFlags(cfg.LoggingConfig).WithContext(ctx)
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

	vldtr := noopauth.NewJwtValidator("mindev")

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

	return service.AllInOneServerService(ctx, cfg, store, vldtr,
		[]controlplane.ServerOption{
			controlplane.WithAuthzClient(&mockauthz.SimpleClient{}),
		},
		[]engine.ExecutorOption{},
		[]reconcilers.ReconcilerOption{},
	)
}
