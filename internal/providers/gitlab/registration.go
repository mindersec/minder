// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/mindersec/minder/internal/providers/gitlab/webhooksecret"
	"github.com/mindersec/minder/internal/util/ptr"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// RegisterEntity implements the Provider interface
func (c *gitlabClient) RegisterEntity(
	ctx context.Context, entType minderv1.Entity, props *properties.Properties,
) (*properties.Properties, error) {
	if !c.SupportsEntity(entType) {
		return nil, provifv1.ErrUnsupportedEntity
	}

	if entType != minderv1.Entity_ENTITY_REPOSITORIES {
		// We only explicitly register repositories
		// Pull requests are handled via origination
		return props, nil
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
		zerolog.Ctx(ctx).Error().
			Str("upstreamID", upstreamID).
			Str("provider-class", Class).
			Err(err).Msg("failed to create webhook")
		return nil, errors.New("failed to create webhook")
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

	sec, err := webhooksecret.New(c.currentWebhookSecret, hookUUID.String())
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook secret: %w", err)
	}

	trve := ptr.Ptr(true)
	hreq := &gitlab.AddProjectHookOptions{
		URL:                   &webhookUniqueURL,
		Token:                 &sec,
		PushEvents:            trve,
		TagPushEvents:         trve,
		MergeRequestsEvents:   trve,
		ReleasesEvents:        trve,
		EnableSSLVerification: trve,
	}

	hook, err := c.doCreateWebhook(ctx, createHookPath, hreq)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}

	outProps := properties.NewProperties(map[string]interface{}{
		// we store as string to avoid any type issues. Note that we
		// need to retrieve it as a string as well.
		RepoPropertyHookID:  fmt.Sprintf("%d", hook.ID),
		RepoPropertyHookURL: hook.URL,
	})

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
	if err := glRESTGet(ctx, c, getHooksPath, &hooks); err != nil {
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

func (c *gitlabClient) updateWebhook(ctx context.Context, upstreamID, hookID string, hookURL string) error {
	// We don't need to update the webhook URL, as it's unique for each
	// registration. We only need to update the secret.
	updateHookPath, err := url.JoinPath("projects", upstreamID, "hooks", hookID)
	if err != nil {
		return fmt.Errorf("failed to join URL path for hook: %w", err)
	}

	hookURLParsed, err := url.Parse(hookURL)
	if err != nil {
		return fmt.Errorf("failed to parse hook URL: %w", err)
	}

	// We need to extract the UUID from the webhook URL. The UUID is
	// the last part of the path.
	hookMinderUUID := hookURLParsed.Path[strings.LastIndex(hookURLParsed.Path, "/")+1:]

	sec, err := webhooksecret.New(c.currentWebhookSecret, hookMinderUUID)
	if err != nil {
		return fmt.Errorf("failed to create webhook secret: %w", err)
	}

	trve := ptr.Ptr(true)
	hreq := &gitlab.EditProjectHookOptions{
		URL:                   &hookURL,
		Token:                 &sec,
		PushEvents:            trve,
		TagPushEvents:         trve,
		MergeRequestsEvents:   trve,
		ReleasesEvents:        trve,
		EnableSSLVerification: trve,
	}

	if err := c.doUpdateWebhook(ctx, updateHookPath, hreq); err != nil {
		return fmt.Errorf("failed to update webhook: %w", err)
	}

	return nil
}

func (c *gitlabClient) doUpdateWebhook(
	ctx context.Context, updateHookPath string, hreq *gitlab.EditProjectHookOptions,
) error {
	req, err := c.NewRequest(http.MethodPut, updateHookPath, hreq)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get projects: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
