// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package webhooks contains logic relating to manipulating GitHub webhooks
package webhooks

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/go-github/v61/github"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/config/server"
	ghprovider "github.com/stacklok/minder/internal/providers/github"
	ghclient "github.com/stacklok/minder/internal/repositories/github/clients"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// WebhookManager encapsulates logic for creating and deleting GitHub webhooks
type WebhookManager interface {
	CreateWebhook(
		ctx context.Context,
		client ghclient.GitHubRepoClient,
		repoOwner string,
		repoName string,
	) (string, *github.Hook, error)

	DeleteWebhook(
		ctx context.Context,
		client ghclient.GitHubRepoClient,
		repoOwner string,
		repoName string,
		hookID int64,
	) error
}

type webhookManager struct {
	webhookConfig server.WebhookConfig
}

// NewWebhookManager instantiates an instances of the WebhookManager interface
func NewWebhookManager(webhookConfig server.WebhookConfig) WebhookManager {
	return &webhookManager{webhookConfig: webhookConfig}
}

var (
	targetedEvents = []string{"*"}
)

// CreateWebhook creates a Minder-managed webhook in the specified GitHub repo
// Note that this method will delete any previous Minder-managed webhooks
// before attempting to make a new one.
// Returns the UUID of this webhook, along with the API response from GitHub
// with details of the new webhook
// https://docs.github.com/en/rest/reference/repos#create-a-repository-webhook
func (w *webhookManager) CreateWebhook(
	ctx context.Context,
	client ghclient.GitHubRepoClient,
	repoOwner string,
	repoName string,
) (string, *github.Hook, error) {
	// generate unique URL for this webhook
	baseURL := w.webhookConfig.ExternalWebhookURL
	hookUUID := uuid.New().String()
	webhookURL := fmt.Sprintf("%s/%s", baseURL, hookUUID)
	parsedWebhookURL, err := url.Parse(webhookURL)
	if err != nil {
		return "", nil, err
	}

	// If we have an existing hook for same repo, delete it
	if err := w.cleanupStaleHooks(ctx, client, repoOwner, repoName, parsedWebhookURL.Host); err != nil {
		return "", nil, err
	}

	// Attempt to register new webhook
	secret, err := w.webhookConfig.GetWebhookSecret()
	if err != nil {
		return "", nil, err
	}
	ping := w.webhookConfig.ExternalPingURL

	jsonCT := "json"
	newHook := &github.Hook{
		Config: &github.HookConfig{
			URL:         &webhookURL,
			ContentType: &jsonCT,
			Secret:      &secret,
		},
		PingURL: &ping,
		Events:  targetedEvents,
	}

	webhook, err := client.CreateHook(ctx, repoOwner, repoName, newHook)
	if err != nil {
		return "", nil, fmt.Errorf("error creating hook: %w", err)
	}

	return hookUUID, webhook, nil
}

// DeleteWebhook deletes the specified webhook from the specified GitHub repo
// Note that deletions of non-existent webhooks are treated as no-ops
func (_ *webhookManager) DeleteWebhook(
	ctx context.Context,
	client ghclient.GitHubRepoClient,
	repoOwner string,
	repoName string,
	hookID int64,
) error {
	resp, err := client.DeleteHook(ctx, repoOwner, repoName, hookID)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// if the hook is not found, we can ignore the error, user might have deleted it manually
			return nil
		}
		return fmt.Errorf("error deleting hook: %w", err)
	}

	return nil
}

func (w *webhookManager) cleanupStaleHooks(
	ctx context.Context,
	client ghclient.GitHubRepoClient,
	repoOwner string,
	repoName string,
	webhookHost string,
) error {
	logger := zerolog.Ctx(ctx)
	hooks, err := client.ListHooks(ctx, repoOwner, repoName)
	if errors.Is(err, ghprovider.ErrNotFound) {
		logger.Debug().Msg("no hooks found")
		return nil
	} else if err != nil {
		return fmt.Errorf("error listing hooks: %w", err)
	}

	for _, hook := range hooks {
		// it is our hook, we can remove it
		shouldDelete, err := ghprovider.IsMinderHook(hook, webhookHost)
		// If err != nil, shouldDelete == false - use one error check for both calls
		if shouldDelete {
			err = w.DeleteWebhook(ctx, client, repoOwner, repoName, hook.GetID())
		}
		if err != nil {
			return fmt.Errorf("error deleting hook: %w", err)
		}
	}

	return nil
}
