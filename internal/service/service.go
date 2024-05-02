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

// Package service contains the business logic for the minder services.
package service

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/authz"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/eea"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/flags"
	"github.com/stacklok/minder/internal/marketplaces"
	"github.com/stacklok/minder/internal/profiles"
	"github.com/stacklok/minder/internal/projects"
	"github.com/stacklok/minder/internal/providers"
	ghprov "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/providers/github/clients"
	"github.com/stacklok/minder/internal/providers/github/installations"
	ghmanager "github.com/stacklok/minder/internal/providers/github/manager"
	"github.com/stacklok/minder/internal/providers/github/service"
	"github.com/stacklok/minder/internal/providers/manager"
	"github.com/stacklok/minder/internal/providers/ratecache"
	provtelemetry "github.com/stacklok/minder/internal/providers/telemetry"
	"github.com/stacklok/minder/internal/reconcilers"
	"github.com/stacklok/minder/internal/repositories/github"
	"github.com/stacklok/minder/internal/repositories/github/webhooks"
	"github.com/stacklok/minder/internal/ruletypes"
)

// AllInOneServerService is a helper function that starts the gRPC and HTTP servers,
// the eventer, aggregator, the executor, and the reconciler.
func AllInOneServerService(
	ctx context.Context,
	cfg *serverconfig.Config,
	store db.Store,
	jwt auth.JwtValidator,
	restClientCache ratecache.RestClientCache,
	authzClient authz.Client,
	idClient auth.Resolver,
	serverMetrics metrics.Metrics,
	providerMetrics provtelemetry.ProviderMetrics,
	executorOpts []engine.ExecutorOption,
) error {
	errg, ctx := errgroup.WithContext(ctx)

	evt, err := events.Setup(ctx, &cfg.Events)
	if err != nil {
		return fmt.Errorf("unable to setup eventer: %w", err)
	}

	flags.OpenFeatureProviderFromFlags(ctx, cfg.Flags)
	cryptoEngine, err := crypto.EngineFromAuthConfig(&cfg.Auth)
	if err != nil {
		return fmt.Errorf("failed to create crypto engine: %w", err)
	}

	profileSvc := profiles.NewProfileService(evt)
	ruleSvc := ruletypes.NewRuleTypeService()
	marketplace, err := marketplaces.NewMarketplaceFromServiceConfig(cfg.Marketplace, profileSvc, ruleSvc)
	if err != nil {
		return fmt.Errorf("failed to create marketplace: %w", err)
	}

	fallbackTokenClient := ghprov.NewFallbackTokenClient(cfg.Provider)
	ghClientFactory := clients.NewGitHubClientFactory(providerMetrics)
	providerStore := providers.NewProviderStore(store)
	whManager := webhooks.NewWebhookManager(cfg.WebhookConfig)
	projectCreator := projects.NewProjectCreator(authzClient, marketplace, &cfg.DefaultProfiles)

	// TODO: isolate GitHub-specific wiring. We'll need to isolate GitHub
	// webhook handling to make this viable.
	ghProviders := service.NewGithubProviderService(
		store,
		cryptoEngine,
		serverMetrics,
		providerMetrics,
		&cfg.Provider,
		makeProjectFactory(projectCreator, cfg.Identity),
		restClientCache,
		fallbackTokenClient,
	)
	githubProviderManager := ghmanager.NewGitHubProviderClassManager(
		restClientCache,
		ghClientFactory,
		&cfg.Provider,
		fallbackTokenClient,
		cryptoEngine,
		whManager,
		store,
		ghProviders,
	)
	providerManager, err := manager.NewProviderManager(providerStore, githubProviderManager)
	if err != nil {
		return fmt.Errorf("failed to create provider manager: %w", err)
	}
	repos := github.NewRepositoryService(whManager, store, evt, providerManager)
	projectDeleter := projects.NewProjectDeleter(authzClient, providerManager)

	s := controlplane.NewServer(
		store,
		evt,
		cfg,
		serverMetrics,
		jwt,
		cryptoEngine,
		authzClient,
		idClient,
		repos,
		profileSvc,
		ruleSvc,
		ghProviders,
		providerManager,
		providerStore,
		projectDeleter,
		projectCreator,
	)

	// Subscribe to events from the identity server
	err = controlplane.SubscribeToIdentityEvents(ctx, store, authzClient, cfg, projectDeleter)
	if err != nil {
		return fmt.Errorf("unable to subscribe to identity server events: %w", err)
	}

	aggr := eea.NewEEA(store, evt, &cfg.Events.Aggregator)

	// consume flush-all events
	evt.ConsumeEvents(aggr)

	// prepend the aggregator to the executor options
	executorOpts = append([]engine.ExecutorOption{engine.WithMiddleware(aggr.AggregateMiddleware)},
		executorOpts...)

	exec, err := engine.NewExecutor(ctx, store, &cfg.Auth, &cfg.Provider, evt, providerStore, restClientCache, executorOpts...)
	if err != nil {
		return fmt.Errorf("unable to create executor: %w", err)
	}

	evt.ConsumeEvents(exec)

	rec, err := reconcilers.NewReconciler(store, evt, cryptoEngine, providerManager)
	if err != nil {
		return fmt.Errorf("unable to create reconciler: %w", err)
	}

	evt.ConsumeEvents(rec)

	im := installations.NewInstallationManager(ghProviders)
	evt.ConsumeEvents(im)

	// Start the gRPC and HTTP server in separate goroutines
	errg.Go(func() error {
		return s.StartGRPCServer(ctx)
	})

	errg.Go(func() error {
		return s.StartHTTPServer(ctx)
	})

	errg.Go(func() error {
		defer evt.Close()
		return evt.Run(ctx)
	})

	// Wait for event handlers to start running
	<-evt.Running()

	// Flush all cache
	if err := aggr.FlushAll(ctx); err != nil {
		return fmt.Errorf("error flushing cache: %w", err)
	}

	// Wait for all entity events to be executed
	exec.Wait()

	return errg.Wait()
}

// makeProjectFactory creates a callback used for GitHub project creation.
// The callback is used to construct a project for a GitHub App installation which was
// created by a user through the app installation flow on GitHub.  The flow on GitHub
// cannot be tied back to a specific project, so we create a new project for the provider.
//
// This is a callback because we want to encapsulate components like identity,
// projectCreator and the like from the providers implementation.
func makeProjectFactory(
	projectCreator projects.ProjectCreator,
	identity serverconfig.IdentityConfigWrapper,
) service.ProjectFactory {
	return func(
		ctx context.Context,
		qtx db.Querier,
		name string,
		ghUser int64,
	) (*db.Project, error) {
		user, err := auth.GetUserForGitHubId(ctx, identity, ghUser)
		if err != nil {
			return nil, fmt.Errorf("error getting user for GitHub ID: %w", err)
		}
		// Ensure the user already exists in the database
		if _, err := qtx.GetUserBySubject(ctx, user); err != nil {
			return nil, fmt.Errorf("error getting user %s from database: %w", user, err)
		}

		topLevelProject, err := projectCreator.ProvisionSelfEnrolledProject(
			ctx,
			qtx,
			name,
			user,
		)
		if err != nil {
			return nil, fmt.Errorf("error creating project: %w", err)
		}
		return topLevelProject, nil
	}
}
