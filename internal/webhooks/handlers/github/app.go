// Package github implements logic for handling GitHub webhooks
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-github/v61/github"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/webhooks/handlers"
	"net/http"
)

type githubAppHandler struct {
	publisher events.Publisher
	metrics   metrics.Metrics
	ghService providers.ProviderService
}

func NewGitHubAppWebhookHandler(
	publisher events.Publisher,
	mt metrics.Metrics,
	ghService providers.ProviderService,
) handlers.WebhookHandler {
	return &githubAppHandler{
		publisher: publisher,
		metrics:   mt,
		ghService: ghService,
	}
}

// HandleGitHubAppWebhook handles incoming GitHub App webhooks
func (g *githubAppHandler) Handle(ctx context.Context, r *http.Request) error {
	wes := &metrics.WebhookEventState{
		Typ:      "unknown",
		Accepted: false,
		Error:    true,
	}
	defer func() {
		g.metrics.AddWebhookEventTypeCount(ctx, wes)
	}()

	rawWBPayload, err := g.ghService.ValidateGitHubAppWebhookPayload(r)
	if err != nil {
		wes.Error = true // TODO: confirm this
		return fmt.Errorf("error validating webhook payload: %w", err)
	}

	wes.Typ = github.WebHookType(r)
	if wes.Typ == "ping" {
		logPingReceivedEvent(r.Context(), rawWBPayload)
		wes.Error = false
		return nil
	}

	m := message.NewMessage(uuid.New().String(), nil)
	m.Metadata.Set(events.ProviderDeliveryIdKey, github.DeliveryID(r))
	m.Metadata.Set(events.ProviderSourceKey, "https://api.github.com/") // TODO: handle other sources
	m.Metadata.Set(events.GithubWebhookEventTypeKey, wes.Typ)

	l := zerolog.Ctx(ctx).With().
		Str("webhook-event-type", m.Metadata[events.GithubWebhookEventTypeKey]).
		Str("providertype", m.Metadata[events.ProviderTypeKey]).
		Str("upstream-delivery-id", m.Metadata[events.ProviderDeliveryIdKey]).
		Logger()

	if err := parseGithubAppEventForProcessing(rawWBPayload, m); err != nil {
		wes = handleParseError(wes.Typ, err)
		if wes.Error {
			return fmt.Errorf("error during parsing: %w", err)
		}
		return nil
	}

	wes.Accepted = true
	l.Info().Str("message-id", m.UUID).Msg("publishing event for execution")

	if err := g.publisher.Publish(providers.ProviderInstallationTopic, m); err != nil {
		return err
	}

	wes.Accepted = true
	return nil
}

func parseGithubAppEventForProcessing(
	rawWHPayload []byte,
	msg *message.Message,
) error {
	var event github.InstallationEvent

	if msg.Metadata.Get(events.GithubWebhookEventTypeKey) != "installation" {
		return newErrNotHandled("github app event %s not handled", msg.Metadata.Get(events.GithubWebhookEventTypeKey))
	}

	err := json.Unmarshal(rawWHPayload, &event)
	if err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}

	action := event.GetAction()
	if action == "" {
		return fmt.Errorf("action is empty")
	}

	if action != WebhookActionEventDeleted {
		return newErrNotHandled("event %s with action %s not handled",
			msg.Metadata.Get(events.GithubWebhookEventTypeKey), action)
	}

	installationID := event.GetInstallation().GetID()
	if installationID == 0 {
		return fmt.Errorf("installation ID is 0")
	}

	payloadBytes, err := json.Marshal(providers.GitHubAppInstallationDeletedPayload{
		InstallationID: installationID,
	})
	if err != nil {
		return fmt.Errorf("error marshalling payload: %w", err)
	}

	providers.ProviderInstanceRemovedMessage(
		msg,
		db.ProviderClassGithubApp,
		payloadBytes)

	return nil
}
