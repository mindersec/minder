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

package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/xanzy/go-gitlab"

	"github.com/stacklok/minder/internal/entities/properties"
	"github.com/stacklok/minder/internal/util/ptr"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// RegisterEntity implements the Provider interface
func (c *gitlabClient) RegisterEntity(
	ctx context.Context, entType minderv1.Entity, props *properties.Properties,
) (*properties.Properties, error) {
	if !c.SupportsEntity(entType) {
		return nil, errors.New("unsupported entity type")
	}

	upstreamID := props.GetProperty(properties.PropertyUpstreamID).GetString()
	if upstreamID == "" {
		return nil, errors.New("missing upstream ID")
	}

	if err := c.cleanUpStaleWebhooks(ctx, upstreamID); err != nil {
		// This is a non-fatal error and may be transient. We log it and
		// continue with the registration.
		zerolog.Ctx(ctx).Error().
			Str("upstreamID", upstreamID).
			Str("provider-class", Class).
			Err(err).Msg("failed to clean up stale webhooks")
	}

	whprops, err := c.createWebhook(ctx, upstreamID)
	if err != nil {
		// There is already enough context in the error message
		return nil, err
	}

	return props.Merge(whprops), nil
}

// DeregisterEntity implements the Provider interface
func (c *gitlabClient) DeregisterEntity(
	ctx context.Context, entType minderv1.Entity, props *properties.Properties,
) error {
	if !c.SupportsEntity(entType) {
		return errors.New("unsupported entity type")
	}

	upstreamID := props.GetProperty(properties.PropertyUpstreamID).GetString()
	if upstreamID == "" {
		return errors.New("missing upstream ID")
	}

	hookID := props.GetProperty(RepoPropertyHookID).GetString()
	if hookID == "" {
		return errors.New("missing hook ID")
	}

	if err := c.deleteWebhook(ctx, upstreamID, hookID); err != nil {
		// There is already enough context in the error message
		return err
	}

	return nil
}

func (c *gitlabClient) createWebhook(ctx context.Context, upstreamID string) (*properties.Properties, error) {
	createHookPath, err := url.JoinPath("projects", upstreamID, "hooks")
	if err != nil {
		return nil, fmt.Errorf("failed to join URL path for hooks: %w", err)
	}

	hookUUID := uuid.New()
	webhookUniqueURL, err := url.JoinPath(c.webhookURL, hookUUID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to join URL path for webhook: %w", err)
	}

	trve := ptr.Ptr(true)
	hreq := &gitlab.AddProjectHookOptions{
		URL: &webhookUniqueURL,
		// TODO: Add secret
		PushEvents:          trve,
		TagPushEvents:       trve,
		MergeRequestsEvents: trve,
		ReleasesEvents:      trve,
		// TODO: Enable SSL verification
	}

	hook, err := c.doCreateWebhook(ctx, createHookPath, hreq)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}

	outProps, err := properties.NewProperties(map[string]interface{}{
		// we store as string to avoid any type issues. Note that we
		// need to retrieve it as a string as well.
		RepoPropertyHookID:  fmt.Sprintf("%d", hook.ID),
		RepoPropertyHookURL: hook.URL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create properties: %w", err)
	}

	return outProps, nil
}

func (c *gitlabClient) doCreateWebhook(
	ctx context.Context, createHookPath string, hreq *gitlab.AddProjectHookOptions,
) (*gitlab.ProjectHook, error) {
	req, err := c.NewRequest(http.MethodPost, createHookPath, hreq)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	hook := &gitlab.ProjectHook{}
	if err := json.NewDecoder(resp.Body).Decode(hook); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return hook, nil
}

func (c *gitlabClient) deleteWebhook(ctx context.Context, upstreamID, hookID string) error {
	deleteHookPath, err := url.JoinPath("projects", upstreamID, "hooks", hookID)
	if err != nil {
		return fmt.Errorf("failed to join URL path for hook: %w", err)
	}

	if err := c.doDeleteWebhook(ctx, deleteHookPath); err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	return nil
}

func (c *gitlabClient) doDeleteWebhook(ctx context.Context, deleteHookPath string) error {
	req, err := c.NewRequest(http.MethodDelete, deleteHookPath, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *gitlabClient) cleanUpStaleWebhooks(ctx context.Context, upstreamID string) error {
	getHooksPath, err := url.JoinPath("projects", upstreamID, "hooks")
	if err != nil {
		return fmt.Errorf("failed to join URL path for hooks: %w", err)
	}

	hooks := []*gitlab.ProjectHook{}
	if err := glREST(ctx, c, http.MethodGet, getHooksPath, nil, &hooks); err != nil {
		return fmt.Errorf("failed to get webhooks: %w", err)
	}

	for _, hook := range hooks {
		if strings.HasPrefix(hook.URL, c.webhookURL) {
			if err := c.deleteWebhook(ctx, upstreamID, fmt.Sprintf("%d", hook.ID)); err != nil {
				return fmt.Errorf("failed to delete webhook: %w", err)
			}
		}
	}

	return nil
}
