// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-github/v63/github"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/controlplane/metrics"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/entities"
	entMsg "github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/events"
	"github.com/mindersec/minder/internal/providers/github/installations"
	"github.com/mindersec/minder/internal/reconcilers/messages"
	"github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

const (
	webhookActionEventDeleted     = "deleted"
	webhookActionEventOpened      = "opened"
	webhookActionEventReopened    = "reopened"
	webhookActionEventSynchronize = "synchronize"
	webhookActionEventClosed      = "closed"
	webhookActionEventPublished   = "published"
	webhookActionEventTransferred = "transferred"
)

// toMessage interface ensures that payloads returned by processor
// routines can be turned into a message.Message
type toMessage interface {
	ToMessage(*message.Message) error
}

var _ toMessage = (*entities.EntityInfoWrapper)(nil)
var _ toMessage = (*installations.InstallationInfoWrapper)(nil)
var _ toMessage = (*messages.MinderEvent)(nil)
var _ toMessage = (*entMsg.HandleEntityAndDoMessage)(nil)

// processingResult struct contains the sole information necessary to
// send a message out from the handler, namely a destination topic and
// an object that knows how to "convert itself" to a watermill message.
//
// It is supposed to just be an easy, uniform way of returning
// results.
type processingResult struct {
	// destination topic
	topic string
	// wrapper object for repository, pull-request, and artifact
	// (package) events.
	wrapper toMessage
}

// HandleWebhookEvent is the main entry point for processing github webhook events
func HandleWebhookEvent(
	mt metrics.Metrics,
	publisher interfaces.Publisher,
	whconfig *server.WebhookConfig,
) http.HandlerFunc {
	// the function handles incoming GitHub webhooks
	// See https://docs.github.com/en/developers/webhooks-and-events/webhooks/about-webhooks
	// for more information.
	// nolint:gocyclo
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := zerolog.Ctx(ctx).With().Logger()

		wes := &metrics.WebhookEventState{
			Typ:      "unknown",
			Accepted: false,
			Error:    true,
		}
		defer func() {
			mt.AddWebhookEventTypeCount(r.Context(), wes)
		}()

		// Validate the payload signature. This is required for security reasons.
		// See https://docs.github.com/en/developers/webhooks-and-events/webhooks/securing-your-webhooks
		// for more information. Note that this is not required for the GitHub App
		// webhook secret, but it is required for OAuth2 App.
		// it returns a uuid for the webhook, but we are not currently using it
		segments := strings.Split(r.URL.Path, "/")
		_ = segments[len(segments)-1]

		rawWBPayload, err := validatePayloadSignature(r, whconfig)
		if err != nil {
			l.Info().Err(err).Msg("Error validating webhook payload")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		wes.Typ = github.WebHookType(r)

		// TODO: extract sender and event time from payload portably
		m := message.NewMessage(uuid.New().String(), nil)
		m.Metadata.Set(events.ProviderDeliveryIdKey, github.DeliveryID(r))
		m.Metadata.Set(events.ProviderTypeKey, string(db.ProviderTypeGithub))
		m.Metadata.Set(events.ProviderSourceKey, "https://api.github.com/") // TODO: handle other sources
		m.Metadata.Set(events.GithubWebhookEventTypeKey, wes.Typ)

		l = l.With().
			Str("webhook-event-type", m.Metadata[events.GithubWebhookEventTypeKey]).
			Str("providertype", m.Metadata[events.ProviderTypeKey]).
			Str("upstream-delivery-id", m.Metadata[events.ProviderDeliveryIdKey]).
			// This is added for consistency with how
			// watermill tracks message UUID when logging.
			Str("message_uuid", m.UUID).
			Logger()
		ctx = l.WithContext(ctx)

		l.Debug().Msg("parsing event")

		var res *processingResult
		var processingErr error

		switch github.WebHookType(r) {
		// All these events are related to a repo and usually
		// contain an action. They all trigger a
		// reconciliation or, in some cases, a deletion.
		case "repository", "meta":
			wes.Accepted = true
			res, processingErr = processRelevantRepositoryEvent(ctx, rawWBPayload)
		case "branch_protection_configuration",
			"branch_protection_rule",
			"code_scanning_alert",
			"create",
			"member",
			"public",
			"push",
			"repository_advisory",
			"repository_import",
			"repository_ruleset",
			"repository_vulnerability_alert",
			"secret_scanning_alert",
			"secret_scanning_alert_location",
			"security_advisory",
			"security_and_analysis",
			"team",
			"team_add":
			wes.Accepted = true
			res, processingErr = processRepositoryEvent(ctx, rawWBPayload)
		case "package":
			// This is an artifact-related event, and can
			// only trigger a reconciliation.
			wes.Accepted = true
			res, processingErr = processPackageEvent(ctx, rawWBPayload)
		case "pull_request":
			wes.Accepted = true
			res, processingErr = processPullRequestEvent(ctx, rawWBPayload)
		case "ping":
			// For ping events, we do not set wes.Accepted
			// to true because they're not relevant
			// business events.
			wes.Error = false
			processPingEvent(ctx, rawWBPayload)
		default:
			l.Info().Msgf("webhook event %s not handled", wes.Typ)
		}

		if processingErr != nil {
			wes = handleParseError(ctx, wes.Typ, processingErr)
			if wes.Error {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			return
		}

		// res is null only when a ping event occurred.
		if res != nil && res.wrapper != nil {
			if err := res.wrapper.ToMessage(m); err != nil {
				wes.Error = true
				l.Error().Err(err).Msg("Error creating event")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// This ensures that loggers on downstream
			// processors have all log attributes
			// available.
			m.SetContext(ctx)

			// Publish the message to the event router
			if err := publisher.Publish(res.topic, m); err != nil {
				wes.Error = true
				l.Error().Err(err).Msg("Error publishing message")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		// We successfully published the message
		wes.Error = false
		w.WriteHeader(http.StatusOK)
	})
}

func validatePayloadSignature(r *http.Request, wc *server.WebhookConfig) (payload []byte, err error) {
	var br *bytes.Reader
	br, err = readerFromRequest(r)
	if err != nil {
		return
	}

	signature := r.Header.Get(github.SHA256SignatureHeader)
	if signature == "" {
		signature = r.Header.Get(github.SHA1SignatureHeader)
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return
	}

	whSecret, err := wc.GetWebhookSecret()
	if err != nil {
		return
	}

	payload, err = github.ValidatePayloadFromBody(contentType, br, signature, []byte(whSecret))
	if err == nil {
		return
	}

	payload, err = validatePreviousSecrets(r.Context(), signature, contentType, br, wc)
	return
}

func readerFromRequest(r *http.Request) (*bytes.Reader, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	err = r.Body.Close()
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

func validatePreviousSecrets(
	ctx context.Context,
	signature, contentType string,
	br *bytes.Reader,
	wc *server.WebhookConfig,
) (payload []byte, err error) {
	previousSecrets := []string{}
	if wc.PreviousWebhookSecretFile != "" {
		previousSecrets, err = wc.GetPreviousWebhookSecrets()
		if err != nil {
			return
		}
	}

	for _, prevSecret := range previousSecrets {
		_, err = br.Seek(0, io.SeekStart)
		if err != nil {
			return
		}
		payload, err = github.ValidatePayloadFromBody(contentType, br, signature, []byte(prevSecret))
		if err == nil {
			zerolog.Ctx(ctx).Warn().Msg("used previous secret to validate payload")
			return
		}
	}

	err = fmt.Errorf("failed to validate payload with any fallback secret")
	return
}

func handleParseError(ctx context.Context, typ string, parseErr error) *metrics.WebhookEventState {
	state := &metrics.WebhookEventState{Typ: typ, Accepted: false, Error: true}
	l := zerolog.Ctx(ctx)

	switch {
	case errors.Is(parseErr, errRepoNotFound):
		state.Error = false
		l.Info().Msg("repository not found")
	case errors.Is(parseErr, errArtifactNotFound):
		state.Error = false
		l.Info().Msg("artifact not found")
	case errors.Is(parseErr, errRepoIsPrivate):
		state.Error = false
		l.Info().Msg("repository is private")
	case errors.Is(parseErr, errNotHandled):
		state.Error = false
		l.Info().Msg("webhook event not handled")
	case errors.Is(parseErr, errArtifactVersionSkipped):
		state.Error = false
		l.Info().Msg("artifact version skipped, has no tags")
	default:
		l.Error().Err(parseErr).Msg("Error parsing github webhook message")
	}

	return state
}
