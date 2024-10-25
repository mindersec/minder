// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package service contains the business logic for the minder services.
package service

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/open-feature/go-sdk/openfeature"
	"golang.org/x/sync/errgroup"

	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/auth/jwt"
	"github.com/mindersec/minder/internal/authz"
	"github.com/mindersec/minder/internal/controlplane"
	"github.com/mindersec/minder/internal/controlplane/metrics"
	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/eea"
	"github.com/mindersec/minder/internal/email/awsses"
	"github.com/mindersec/minder/internal/email/noop"
	"github.com/mindersec/minder/internal/engine"
	"github.com/mindersec/minder/internal/entities/handlers"
	propService "github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/flags"
	"github.com/mindersec/minder/internal/history"
	"github.com/mindersec/minder/internal/invites"
	"github.com/mindersec/minder/internal/marketplaces"
	"github.com/mindersec/minder/internal/metrics/meters"
	"github.com/mindersec/minder/internal/projects"
	"github.com/mindersec/minder/internal/providers"
	"github.com/mindersec/minder/internal/providers/dockerhub"
	ghprov "github.com/mindersec/minder/internal/providers/github"
	"github.com/mindersec/minder/internal/providers/github/clients"
	"github.com/mindersec/minder/internal/providers/github/installations"
	ghmanager "github.com/mindersec/minder/internal/providers/github/manager"
	"github.com/mindersec/minder/internal/providers/github/service"
	gitlabmanager "github.com/mindersec/minder/internal/providers/gitlab/manager"
	"github.com/mindersec/minder/internal/providers/manager"
	"github.com/mindersec/minder/internal/providers/ratecache"
	"github.com/mindersec/minder/internal/providers/session"
	provtelemetry "github.com/mindersec/minder/internal/providers/telemetry"
	"github.com/mindersec/minder/internal/reconcilers"
	"github.com/mindersec/minder/internal/reminderprocessor"
	"github.com/mindersec/minder/internal/repositories"
	"github.com/mindersec/minder/internal/roles"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/engine/selectors"
	"github.com/mindersec/minder/pkg/eventer"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
	"github.com/mindersec/minder/pkg/profiles"
	"github.com/mindersec/minder/pkg/ruletypes"
)

// AllInOneServerService is a helper function that starts the gRPC and HTTP servers,
// the eventer, aggregator, the executor, and the reconciler.
//
//nolint:gocyclo // This function is expected to be large
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

	evt, err := eventer.New(ctx, &cfg.Events)
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
	projectCreator := projects.NewProjectCreator(authzClient, marketplace, &cfg.DefaultProfiles)
	propSvc := propService.NewPropertiesService(store)
	featureFlagClient := openfeature.NewClient(cfg.Flags.AppName)

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
		&cfg.WebhookConfig,
		fallbackTokenClient,
		cryptoEngine,
		store,
		ghProviders,
		propSvc,
		serverMetrics,
		evt,
	)

	provmans := []manager.ProviderClassManager{githubProviderManager}

	if flags.Bool(ctx, featureFlagClient, flags.DockerHubProvider) {
		dockerhubProviderManager := dockerhub.NewDockerHubProviderClassManager(
			cryptoEngine,
			store,
		)
		provmans = append(provmans, dockerhubProviderManager)
	}

	if flags.Bool(ctx, featureFlagClient, flags.GitLabProvider) {
		gitlabProviderManager, err := gitlabmanager.NewGitLabProviderClassManager(
			ctx,
			cryptoEngine,
			store,
			evt,
			cfg.Provider.GitLab,
			cfg.WebhookConfig,
		)
		if err != nil {
			return fmt.Errorf("failed to create gitlab provider manager: %w", err)
		}

		provmans = append(provmans, gitlabProviderManager)
	}

	providerManager, closer, err := manager.NewProviderManager(ctx, providerStore,
		provmans...)
	if err != nil {
		return fmt.Errorf("failed to create provider manager: %w", err)
	}
	defer closer()

	providerAuthManager, err := manager.NewAuthManager(provmans...)
	if err != nil {
		return fmt.Errorf("failed to create provider auth manager: %w", err)
	}
	historySvc := history.NewEvaluationHistoryService(providerManager)
	repos := repositories.NewRepositoryService(store, propSvc, evt, providerManager)
	projectDeleter := projects.NewProjectDeleter(authzClient, providerManager)
	sessionsService := session.NewProviderSessionService(providerManager, providerStore, store)

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
		propSvc,
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
	err = controlplane.SubscribeToAdminEvents(ctx, store, authzClient, cfg, projectDeleter)
	if err != nil {
		return fmt.Errorf("unable to subscribe to account events: %w", err)
	}

	aggr := eea.NewEEA(store, evt, &cfg.Events.Aggregator, propSvc, providerManager)

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
		historySvc,
		featureFlagClient,
		profileStore,
		selEnv,
		propSvc,
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

	// Register the entity refresh manager to handle entity refresh events
	refresh := handlers.NewRefreshEntityAndEvaluateHandler(evt, store, propSvc, providerManager)
	evt.ConsumeEvents(refresh)

	refreshById := handlers.NewRefreshByIDAndEvaluateHandler(evt, store, propSvc, providerManager)
	evt.ConsumeEvents(refreshById)

	addOriginatingEntity := handlers.NewAddOriginatingEntityHandler(evt, store, propSvc, providerManager)
	evt.ConsumeEvents(addOriginatingEntity)

	delOriginatingEntity := handlers.NewRemoveOriginatingEntityHandler(evt, store, propSvc, providerManager)
	evt.ConsumeEvents(delOriginatingEntity)

	getAndDeleteEntity := handlers.NewGetEntityAndDeleteHandler(evt, store, propSvc)
	evt.ConsumeEvents(getAndDeleteEntity)

	// Register the email manager to handle email invitations
	var mailClient interfaces.Consumer
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

	// Processor would only work for sql driver as reminder publisher is sql based
	reminderProcessor := reminderprocessor.NewReminderProcessor(evt)
	evt.ConsumeEvents(reminderProcessor)

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
