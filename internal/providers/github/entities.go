//
// Copyright 2024 Stacklok, Inc.
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

package github

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/google/go-github/v63/github"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/entities/properties"
	ghprop "github.com/stacklok/minder/internal/providers/github/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var (
	targetedEvents = []string{"*"}
)

// GetEntityName implements the Provider interface
func (c *GitHub) GetEntityName(entType minderv1.Entity, props *properties.Properties) (string, error) {
	if props == nil {
		return "", errors.New("properties are nil")
	}
	if c.propertyFetchers == nil {
		return "", errors.New("property fetchers not initialized")
	}
	fetcher := c.propertyFetchers.EntityPropertyFetcher(entType)
	if fetcher == nil {
		return "", fmt.Errorf("no fetcher found for entity type %s", entType)
	}
	return fetcher.GetName(props)
}

// SupportsEntity implements the Provider interface
func (c *GitHub) SupportsEntity(entType minderv1.Entity) bool {
	return c.propertyFetchers.EntityPropertyFetcher(entType) != nil
}

// RegisterEntity implements the Provider interface
func (c *GitHub) RegisterEntity(
	ctx context.Context, entityType minderv1.Entity, props *properties.Properties,
) (*properties.Properties, error) {
	// We only need explicit registration steps for repositories
	// The rest of the entities are originated from them.
	if entityType != minderv1.Entity_ENTITY_REPOSITORIES {
		return props, nil
	}

	// generate unique URL for this webhook
	// TODO: we should change this to use a per-provider configuration.
	baseURL := c.webhookConfig.ExternalWebhookURL
	hookUUID := uuid.New().String()
	webhookURL, err := url.JoinPath(baseURL, hookUUID)
	if err != nil {
		return nil, fmt.Errorf("error joining webhook URL: %w", err)
	}

	parsedWebhookURL, err := url.Parse(webhookURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing webhook URL: %w", err)
	}

	repoNameP := props.GetProperty(ghprop.RepoPropertyName)
	if repoNameP == nil {
		return nil, errors.New("repo name property not found")
	}

	repoOwnerP := props.GetProperty(ghprop.RepoPropertyOwner)
	if repoOwnerP == nil {
		return nil, errors.New("repo owner property not found")
	}

	repoName := repoNameP.GetString()
	repoOwner := repoOwnerP.GetString()

	// If we have an existing hook for same repo, delete it
	if err := c.cleanupStaleHooks(ctx, repoOwner, repoName, parsedWebhookURL.Host); err != nil {
		return nil, fmt.Errorf("error cleaning up stale hooks: %w", err)
	}

	// Attempt to register new webhook
	secret, err := c.webhookConfig.GetWebhookSecret()
	if err != nil {
		return nil, fmt.Errorf("error getting webhook secret: %w", err)
	}
	ping := c.webhookConfig.ExternalPingURL

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

	webhook, err := c.CreateHook(ctx, repoOwner, repoName, newHook)
	if err != nil {
		return nil, fmt.Errorf("error creating hook: %w", err)
	}

	whprops, err := properties.NewProperties(map[string]any{
		ghprop.RepoPropertyHookUiid: hookUUID,
		ghprop.RepoPropertyHookId:   webhook.GetID(),
		ghprop.RepoPropertyHookUrl:  webhook.GetURL(),
		ghprop.RepoPropertyHookName: webhook.GetName(),
		ghprop.RepoPropertyHookType: webhook.GetType(),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating webhook properties: %w", err)
	}

	return props.Merge(whprops), nil
}

// DeregisterEntity implements the Provider interface
func (c *GitHub) DeregisterEntity(ctx context.Context, entityType minderv1.Entity, props *properties.Properties) error {
	// We only need explicit registration steps for repositories
	// The rest of the entities are originated from them.
	if entityType != minderv1.Entity_ENTITY_REPOSITORIES {
		return nil
	}

	repoNameP := props.GetProperty(ghprop.RepoPropertyName)
	if repoNameP == nil {
		return errors.New("repo name property not found")
	}

	repoOwnerP := props.GetProperty(ghprop.RepoPropertyOwner)
	if repoOwnerP == nil {
		return errors.New("repo owner property not found")
	}

	hookIDP := props.GetProperty(ghprop.RepoPropertyHookId)
	if hookIDP == nil {
		return errors.New("hook ID property not found")
	}

	repoName := repoNameP.GetString()
	repoOwner := repoOwnerP.GetString()
	hookID := hookIDP.GetInt64()

	err := c.DeleteHook(ctx, repoOwner, repoName, hookID)
	if err != nil {
		return fmt.Errorf("error deleting hook: %w", err)
	}

	return nil
}

func (c *GitHub) cleanupStaleHooks(
	ctx context.Context,
	repoOwner string,
	repoName string,
	webhookHost string,
) error {
	logger := zerolog.Ctx(ctx)
	hooks, err := c.ListHooks(ctx, repoOwner, repoName)
	if errors.Is(err, ErrNotFound) {
		logger.Debug().Msg("no hooks found")
		return nil
	} else if err != nil {
		return fmt.Errorf("error listing hooks: %w", err)
	}

	for _, hook := range hooks {
		// it is our hook, we can remove it
		shouldDelete, err := IsMinderHook(hook, webhookHost)
		// If err != nil, shouldDelete == false - use one error check for both calls
		if shouldDelete {
			err = c.DeleteHook(ctx, repoOwner, repoName, hook.GetID())
		}
		if err != nil {
			return fmt.Errorf("error deleting hook: %w", err)
		}
	}

	return nil
}
