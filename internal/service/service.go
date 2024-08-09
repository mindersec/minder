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

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/open-feature/go-sdk/openfeature"
	"golang.org/x/sync/errgroup"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/auth/jwt"
	"github.com/stacklok/minder/internal/authz"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/eea"
	"github.com/stacklok/minder/internal/email/awsses"
	"github.com/stacklok/minder/internal/email/noop"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/engine/selectors"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/flags"
	"github.com/stacklok/minder/internal/history"
	"github.com/stacklok/minder/internal/invites"
	"github.com/stacklok/minder/internal/marketplaces"
	"github.com/stacklok/minder/internal/metrics/meters"
	"github.com/stacklok/minder/internal/profiles"
	"github.com/stacklok/minder/internal/projects"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/dockerhub"
	ghprov "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/providers/github/clients"
	"github.com/stacklok/minder/internal/providers/github/installations"
	ghmanager "github.com/stacklok/minder/internal/providers/github/manager"
	"github.com/stacklok/minder/internal/providers/github/service"
	gitlabmanager "github.com/stacklok/minder/internal/providers/gitlab/manager"
	"github.com/stacklok/minder/internal/providers/manager"
	"github.com/stacklok/minder/internal/providers/ratecache"
	"github.com/stacklok/minder/internal/providers/session"
	provtelemetry "github.com/stacklok/minder/internal/providers/telemetry"
	"github.com/stacklok/minder/internal/reconcilers"
	"github.com/stacklok/minder/internal/repositories/github"
	"github.com/stacklok/minder/internal/repositories/github/webhooks"
	"github.com/stacklok/minder/internal/roles"
	"github.com/stacklok/minder/internal/ruletypes"
)

// AllInOneServerService is a helper function that starts the gRPC and HTTP servers,
// the eventer, aggregator, the executor, and the reconciler.
func AllInOneServerService(
	ctx context.Context,
	cfg *serverconfig.Config,
	store db.Store,
	jwtValidator jwt.Validator,
	restClientCache ratecache.RestClientCache,
	authzClient authz.Client,
	idClient auth.Resolver,
	serverMetrics metrics.Metrics,
	providerMetrics provtelemetry.ProviderMetrics,
	executorMiddleware []message.HandlerMiddleware,
	meterFactory meters.MeterFactory,
) error {
	errg, ctx := errgroup.WithContext(ctx)

	evt, err := events.Setup(ctx, &cfg.Events)
	if err != nil {
		return fmt.Errorf("unable to setup eventer: %w", err)
	}

	flags.OpenFeatureProviderFromFlags(ctx, cfg.Flags)
	cryptoEngine, err := crypto.NewEngineFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create crypto engine: %w", err)
	}

	serverconfig.FallbackOAuthClientConfigValues("github", &cfg.Provider.GitHub.OAuthClientConfig)
	serverconfig.FallbackOAuthClientConfigValues("github-app", &cfg.Provider.GitHubApp.OAuthClientConfig)

	historySvc := history.NewEvaluationHistoryService()
	inviteSvc := invites.NewInviteService()
	selChecker := selectors.NewEnv()
	profileSvc := profiles.NewProfileService(evt, selChecker)
	ruleSvc := ruletypes.NewRuleTypeService()
	roleScv := roles.NewRoleService()
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
		&cfg.Provider,
		makeProjectFactory(projectCreator, cfg.Identity),
		ghClientFactory,
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
	dockerhubProviderManager := dockerhub.NewDockerHubProviderClassManager(
		cryptoEngine,
		store,
	)
	gitlabProviderManager := gitlabmanager.NewGitLabProviderClassManager(
		cryptoEngine,
		store,
		cfg.Provider.GitLab,
	)
	providerManager, err := manager.NewProviderManager(providerStore,
		githubProviderManager, dockerhubProviderManager, gitlabProviderManager)
	if err != nil {
		return fmt.Errorf("failed to create provider manager: %w", err)
	}
	providerAuthManager, err := manager.NewAuthManager(githubProviderManager, dockerhubProviderManager, gitlabProviderManager)
	if err != nil {
		return fmt.Errorf("failed to create provider auth manager: %w", err)
	}
	repos := github.NewRepositoryService(whManager, store, evt, providerManager)
	projectDeleter := projects.NewProjectDeleter(authzClient, providerManager)
	sessionsService := session.NewProviderSessionService(providerManager, providerStore, store)
	featureFlagClient := openfeature.NewClient(cfg.Flags.AppName)

	s := controlplane.NewServer(
		store,
		evt,
		cfg,
		serverMetrics,
		jwtValidator,
		cryptoEngine,
		authzClient,
		idClient,
		inviteSvc,
		repos,
		roleScv,
		profileSvc,
		historySvc,
		ruleSvc,
		ghProviders,
		providerManager,
		providerAuthManager,
		providerStore,
		sessionsService,
		projectDeleter,
		projectCreator,
		featureFlagClient,
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
	executorMiddleware = append([]message.HandlerMiddleware{aggr.AggregateMiddleware}, executorMiddleware...)
	executorMetrics, err := engine.NewExecutorMetrics(meterFactory)
	if err != nil {
		return fmt.Errorf("unable to create metrics for executor: %w", err)
	}

	profileStore := profiles.NewProfileStore(store)
	selEnv := selectors.NewEnv()

	// Register the executor to handle entity evaluations
	exec := engine.NewExecutor(
		store,
		providerManager,
		executorMetrics,
		history.NewEvaluationHistoryService(),
		featureFlagClient,
		profileStore,
		selEnv,
	)

	handler := engine.NewExecutorEventHandler(
		ctx,
		evt,
		executorMiddleware,
		exec,
	)

	evt.ConsumeEvents(handler)

	// Register the reconciler to handle entity events
	rec, err := reconcilers.NewReconciler(store, evt, cryptoEngine, providerManager, repos)
	if err != nil {
		return fmt.Errorf("unable to create reconciler: %w", err)
	}
	evt.ConsumeEvents(rec)

	// Register the installation manager to handle provider installation events
	im := installations.NewInstallationManager(ghProviders)
	evt.ConsumeEvents(im)

	// Register the email manager to handle email invitations
	var mailClient events.Consumer
	if cfg.Email.AWSSES.Region != "" && cfg.Email.AWSSES.Sender != "" {
		// If AWS SES is configured, use it to send emails
		mailClient, err = awsses.New(ctx, cfg.Email.AWSSES.Sender, cfg.Email.AWSSES.Region)
		if err != nil {
			return fmt.Errorf("unable to create aws ses email client: %w", err)
		}
	} else {
		// Otherwise, use a no-op email client
		mailClient = noop.New()
	}
	evt.ConsumeEvents(mailClient)

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
	handler.Wait()

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
