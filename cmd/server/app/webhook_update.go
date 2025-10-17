// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/controlplane/metrics"
	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/db"
	propssvc "github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/providers"
	ghprovider "github.com/mindersec/minder/internal/providers/github"
	"github.com/mindersec/minder/internal/providers/github/clients"
	ghmanager "github.com/mindersec/minder/internal/providers/github/manager"
	"github.com/mindersec/minder/internal/providers/manager"
	"github.com/mindersec/minder/internal/providers/ratecache"
	"github.com/mindersec/minder/internal/providers/telemetry"
	"github.com/mindersec/minder/pkg/config"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

func cmdWebhookUpdate() *cobra.Command {
	var updateCmd = &cobra.Command{
		Use:   "update",
		Short: "update the webhook configuration",
		Long:  `Command to upgrade webhook configuration`,
		RunE:  runCmdWebhookUpdate,
	}

	updateCmd.Flags().StringP("provider",
		"p", "github",
		"what provider interface must the provider implement to be updated")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if err := updateCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

	return updateCmd
}

func runCmdWebhookUpdate(cmd *cobra.Command, _ []string) error {
	cfg, err := config.ReadConfigFromViper[serverconfig.Config](viper.GetViper())
	if err != nil {
		return fmt.Errorf("unable to read config: %w", err)
	}

	ctx := serverconfig.LoggerFromConfigFlags(cfg.LoggingConfig).WithContext(context.Background())

	providerName := cmd.Flag("provider").Value.String()

	store, closer, err := wireUpDB(ctx, cfg)
	if err != nil {
		return err
	}
	defer closer()

	allProviders, err := store.GlobalListProviders(ctx)
	if err != nil {
		return fmt.Errorf("unable to list providers: %w", err)
	}

	whSecret, err := getWebhookSecret(cfg)
	if err != nil {
		return err
	}

	providerManager, pmcloser, err := wireUpProviderManager(cmd.Context(), cfg, store)
	if err != nil {
		return fmt.Errorf("failed to instantiate provider manager: %w", err)
	}
	defer pmcloser()

	for _, provider := range allProviders {
		if providerName != "" && providerName != provider.Name {
			continue
		}

		if !provider.CanImplement(db.ProviderTypeGithub) {
			// currently we can only operate on GitHub
			// revisit this once we add more providers with webhooks
			zerolog.Ctx(ctx).Info().
				Str("name", provider.Name).
				Str("uuid", provider.ID.String()).
				Msg("provider does not implement the requested provider interface")
			continue
		}

		zerolog.Ctx(ctx).Info().
			Str("name", provider.Name).
			Str("uuid", provider.ID.String()).
			Msg("provider")

		// We end up querying each provider db record twice - once to build
		// the slice which this loop iterates over, and a second time to
		// instantiate the provider. Taking this approach since we plan on
		// changing webhook handling in minder, so I do not want to create any
		// throwaway code.
		providerInstance, err := providerManager.InstantiateFromID(ctx, provider.ID)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("cannot instantiate provider")
			continue
		}

		ghCli, err := provifv1.As[provifv1.GitHub](providerInstance)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("cannot convert to github provider")
			continue
		}

		webhookURL, err := url.JoinPath(
			cfg.WebhookConfig.ExternalWebhookURL,
			url.PathEscape(string(db.ProviderTypeGithub)),
		)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("error joining webhook URL")
			continue
		}
		updateErr := updateGithubWebhooks(ctx, ghCli, store, provider, webhookURL, whSecret)
		if updateErr != nil {
			zerolog.Ctx(ctx).Err(updateErr).Msg("unable to update webhooks")
		}
	}

	return nil
}

func updateGithubWebhooks(
	ctx context.Context,
	ghCli provifv1.GitHub,
	store db.Store,
	provider db.Provider,
	webhookHost string,
	secret string,
) error {
	// Get all repository entities for this provider and project
	repoEnts, err := store.GetEntitiesByType(ctx, db.GetEntitiesByTypeParams{
		EntityType: db.EntitiesRepository,
		ProviderID: provider.ID,
		Projects:   []uuid.UUID{provider.ProjectID},
	})
	if err != nil {
		return fmt.Errorf("unable to list repositories: %w", err)
	}

	for _, repoEnt := range repoEnts {
		err := updateGithubRepoHooks(ctx, ghCli, repoEnt, webhookHost, secret)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("unable to update repo hooks")
			continue
		}
	}

	return nil
}

func updateGithubRepoHooks(
	ctx context.Context,
	ghCli provifv1.GitHub,
	repoEnt db.EntityInstance,
	webhookHost string,
	secret string,
) error {
	// Parse the entity name to extract owner and repo name
	// Entity name format is "owner/repo"
	parts := strings.Split(repoEnt.Name, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid entity name format: %s", repoEnt.Name)
	}
	repoOwner := parts[0]
	repoName := parts[1]

	repoLogger := zerolog.Ctx(ctx).With().
		Str("repo", repoName).
		Str("owner", repoOwner).
		Str("uuid", repoEnt.ID.String()).
		Logger()
	repoLogger.Info().Msg("updating repo hooks")

	hooks, err := ghCli.ListHooks(ctx, repoOwner, repoName)
	if errors.Is(err, ghprovider.ErrNotFound) {
		repoLogger.Debug().Msg("no hooks found")
		return nil
	} else if err != nil {
		return fmt.Errorf("unable to list hooks: %w", err)
	}

	for _, hook := range hooks {
		hookLogger := repoLogger.With().Int64("hook_id", hook.GetID()).Str("url", hook.GetURL()).Logger()
		isMinder, err := ghprovider.IsMinderHook(hook, webhookHost)
		if err != nil {
			hookLogger.Err(err).Msg("unable to determine if hook is a minder hook")
			continue
		}
		if !isMinder {
			hookLogger.Info().Msg("hook is not a minder hook")
			continue
		}

		hook.Config.Secret = &secret
		_, err = ghCli.EditHook(ctx, repoOwner, repoName, hook.GetID(), hook)
		if err != nil {
			hookLogger.Err(err).Msg("unable to update hook")
			continue
		}
		hookLogger.Info().Msg("hook updated")
	}

	return nil
}

func wireUpProviderManager(
	ctx context.Context, cfg *serverconfig.Config, store db.Store,
) (manager.ProviderManager, func(), error) {
	noop := func() {}
	cryptoEng, err := crypto.NewEngineFromConfig(cfg)
	if err != nil {
		return nil, noop, fmt.Errorf("failed to create crypto engine: %w", err)
	}
	fallbackTokenClient := ghprovider.NewFallbackTokenClient(cfg.Provider)
	providerStore := providers.NewProviderStore(store)
	propSvc := propssvc.NewPropertiesService(store)
	githubProviderManager := ghmanager.NewGitHubProviderClassManager(
		&ratecache.NoopRestClientCache{},
		clients.NewGitHubClientFactory(telemetry.NewNoopMetrics()),
		&cfg.Provider,
		&cfg.WebhookConfig,
		fallbackTokenClient,
		cryptoEng,
		store,
		nil, // ghProviderService not needed here
		propSvc,
		metrics.NewNoopMetrics(),
		nil, // we won't be publishing any messages
	)

	return manager.NewProviderManager(ctx, providerStore, githubProviderManager)
}

func getWebhookSecret(cfg *serverconfig.Config) (string, error) {
	secret, err := cfg.WebhookConfig.GetWebhookSecret()
	if err != nil {
		return "", fmt.Errorf("cannot read secret from config: %w", err)
	}

	if secret == "" {
		return "", fmt.Errorf("webhook secret is empty in config: %w", err)
	}

	return secret, nil
}
