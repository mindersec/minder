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
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/config"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/providers"
	ghprovider "github.com/stacklok/minder/internal/providers/github"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
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

	ctx := logger.FromFlags(cfg.LoggingConfig).WithContext(context.Background())

	providerName := cmd.Flag("provider").Value.String()

	cryptoEng, err := crypto.EngineFromAuthConfig(&cfg.Auth)
	if err != nil {
		return fmt.Errorf("failed to create crypto engine: %w", err)
	}

	dbConn, _, err := cfg.Database.GetDBConnection(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer func(dbConn *sql.DB) {
		err := dbConn.Close()
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("error closing database connection")
		}
	}(dbConn)

	store := db.NewStore(dbConn)
	allProviders, err := store.GlobalListProviders(ctx)
	if err != nil {
		return fmt.Errorf("unable to list providers: %w", err)
	}

	webhookUrl, err := url.Parse(cfg.WebhookConfig.ExternalWebhookURL)
	if err != nil {
		return fmt.Errorf("unable to parse webhook url: %w", err)
	}

	for _, provider := range allProviders {
		zerolog.Ctx(ctx).Info().Str("name", provider.Name).Str("uuid", provider.ID.String()).Msg("provider")
		pb, err := providers.GetProviderBuilder(ctx, provider, store, cryptoEng, &cfg.Provider)
		if err != nil {
			return fmt.Errorf("unable to get provider builder: %w", err)
		}

		if !pb.Implements(db.ProviderType(providerName)) {
			zerolog.Ctx(ctx).Info().
				Str("name", provider.Name).
				Str("uuid", provider.ID.String()).
				Msg("provider does not implement the requested provider interface")
			continue
		}

		var updateErr error

		if db.ProviderType(providerName) == db.ProviderTypeGithub {
			ghCli, err := pb.GetGitHub()
			if err != nil {
				zerolog.Ctx(ctx).Err(err).Msg("cannot get github client")
			}
			updateErr = updateGithubWebhooks(ctx, ghCli, store, provider, webhookUrl.Host, cfg.WebhookConfig.WebhookSecret)
		} else {
			updateErr = fmt.Errorf("provider type %s not supported", providerName)
		}

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
	repos, err := store.ListRegisteredRepositoriesByProjectIDAndProvider(ctx,
		db.ListRegisteredRepositoriesByProjectIDAndProviderParams{
			Provider:  provider.Name,
			ProjectID: provider.ProjectID,
		})
	if err != nil {
		return fmt.Errorf("unable to list registered repositories: %w", err)
	}

	for _, repo := range repos {
		err := updateGithubRepoHooks(ctx, ghCli, repo, webhookHost, secret)
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
	repo db.Repository,
	webhookHost string,
	secret string,
) error {
	repoLogger := zerolog.Ctx(ctx).With().Str("repo", repo.RepoName).Str("uuid", repo.ID.String()).Logger()
	repoLogger.Info().Msg("updating repo hooks")

	hooks, err := ghCli.ListHooks(ctx, repo.RepoOwner, repo.RepoName)
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
		_, err = ghCli.EditHook(ctx, repo.RepoOwner, repo.RepoName, hook.GetID(), hook)
		if err != nil {
			hookLogger.Err(err).Msg("unable to update hook")
			continue
		}
		hookLogger.Info().Msg("hook updated")
	}

	return nil
}
