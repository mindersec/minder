// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package webhook implements github webhook handlers for the github provider
package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-github/v63/github"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/controlplane/metrics"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/entities/properties"
	"github.com/mindersec/minder/internal/providers/github/clients"
	"github.com/mindersec/minder/internal/providers/github/installations"
	"github.com/mindersec/minder/internal/providers/github/service"
	"github.com/mindersec/minder/internal/reconcilers/messages"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/eventer/constants"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

// installationEvent are events related the GitHub App. Minder uses
// them for provider enrollement.
type installationEvent struct {
	Action       *string       `json:"action,omitempty"`
	Installation *installation `json:"installation,omitempty"`
}

func (i *installationEvent) GetAction() string {
	if i.Action != nil {
		return *i.Action
	}
	return ""
}

func (i *installationEvent) GetInstallation() *installation {
	return i.Installation
}

// installationRepositoriesEvent are events occurring when there is
// activity relating to which repositories a GitHub App installation
// can access.
type installationRepositoriesEvent struct {
	Action              *string       `json:"action,omitempty"`
	RepositoriesAdded   []*repo       `json:"repositories_added,omitempty"`
	RepositoriesRemoved []*repo       `json:"repositories_removed,omitempty"`
	RepositorySelection *string       `json:"repository_selection,omitempty"`
	Sender              *user         `json:"sender,omitempty"`
	Installation        *installation `json:"installation,omitempty"`
}

func (i *installationRepositoriesEvent) GetAction() string {
	if i.Action != nil {
		return *i.Action
	}
	return ""
}

func (i *installationRepositoriesEvent) GetRepositoriesAdded() []*repo {
	return i.RepositoriesAdded
}

func (i *installationRepositoriesEvent) GetRepositoriesRemoved() []*repo {
	return i.RepositoriesRemoved
}

func (i *installationRepositoriesEvent) GetRepositorySelection() string {
	if i.RepositorySelection != nil {
		return *i.RepositorySelection
	}
	return ""
}

func (i *installationRepositoriesEvent) GetSender() *user {
	return i.Sender
}

func (i *installationRepositoriesEvent) GetInstallation() *installation {
	return i.Installation
}

type installation struct {
	ID *int64 `json:"id,omitempty"`
}

func (i *installation) GetID() int64 {
	if i.ID != nil {
		return *i.ID
	}
	return 0
}

// HandleGitHubAppWebhook handles incoming GitHub App webhooks
func HandleGitHubAppWebhook(
	store db.Store,
	ghService service.GitHubProviderService,
	mt metrics.Metrics,
	publisher interfaces.Publisher,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		rawWBPayload, err := ghService.ValidateGitHubAppWebhookPayload(r)
		if err != nil {
			l.Info().Err(err).Msg("Error validating webhook payload")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		wes.Typ = github.WebHookType(r)

		m := message.NewMessage(uuid.New().String(), nil)
		m.Metadata.Set(constants.ProviderDeliveryIdKey, github.DeliveryID(r))
		// TODO: handle other sources
		m.Metadata.Set(constants.ProviderSourceKey, "https://api.github.com/")
		m.Metadata.Set(constants.GithubWebhookEventTypeKey, wes.Typ)

		l = l.With().
			Str("webhook-event-type", m.Metadata[constants.GithubWebhookEventTypeKey]).
			Str("providertype", m.Metadata[constants.ProviderTypeKey]).
			Str("upstream-delivery-id", m.Metadata[constants.ProviderDeliveryIdKey]).
			// This is added for consistency with how
			// watermill tracks message UUID when logging.
			Str("message_uuid", m.UUID).
			Logger()
		ctx = l.WithContext(ctx)

		var results []*processingResult
		var processingErr error

		switch github.WebHookType(r) {
		case "ping":
			// For ping events, we do not set wes.Accepted
			// to true because they're not relevant
			// business events.
			wes.Error = false
			processPingEvent(ctx, rawWBPayload)
		case "installation":
			wes.Accepted = true
			results, processingErr = processInstallationAppEvent(ctx, rawWBPayload)
		case "installation_repositories":
			wes.Accepted = true
			results, processingErr = processInstallationRepositoriesAppEvent(ctx, store, rawWBPayload)
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

		for _, res := range results {
			l.Info().Str("message-id", m.UUID).Msg("publishing event for execution")
			if res.wrapper != nil {
				if err := res.wrapper.ToMessage(m); err != nil {
					wes.Error = true
					l.Error().Err(err).Msg("Error creating event")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}

			// This ensures that loggers on downstream
			// processors have all log attributes
			// available.
			m.SetContext(ctx)

			if err := publisher.Publish(res.topic, m); err != nil {
				wes.Error = true
				l.Error().Err(err).Msg("Error publishing message")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		wes.Error = false
		w.WriteHeader(http.StatusOK)
	}
}

// processInstallationAppEvent processes events related to changes to
// the app itself as well as the list of accessible repositories.
//
// There are several possible actions, but in the current user flows
// we only process deletion.
func processInstallationAppEvent(
	_ context.Context,
	payload []byte,
) ([]*processingResult, error) {
	var event *installationEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	// Check fields mandatory for processing the event
	if event.GetAction() == "" {
		return nil, errors.New("invalid event: action is nil")
	}
	if event.GetAction() != webhookActionEventDeleted {
		return nil, newErrNotHandled(`event "installation" with action %s not handled`,
			event.GetAction(),
		)
	}
	if event.GetInstallation() == nil {
		return nil, errors.New("invalid event: installation is nil")
	}
	if event.GetInstallation().GetID() == 0 {
		return nil, errors.New("invalid installation: id is 0")
	}

	payloadBytes, err := json.Marshal(
		service.GitHubAppInstallationDeletedPayload{
			InstallationID: event.GetInstallation().GetID(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload: %w", err)
	}

	iiw := installations.NewInstallationInfoWrapper().
		WithProviderClass(db.ProviderClassGithubApp).
		WithPayload(payloadBytes)

	return []*processingResult{
		{
			topic:   installations.ProviderInstallationTopic,
			wrapper: iiw,
		},
	}, nil
}

// processInstallationRepositoriesAppEvent processes events related to
// changes to the list of repositories that the app can access.
//
//nolint:gocyclo
func processInstallationRepositoriesAppEvent(
	ctx context.Context,
	store db.Store,
	payload []byte,
) ([]*processingResult, error) {
	var event *installationRepositoriesEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	// Check fields mandatory for processing the event
	if event.GetAction() == "" {
		return nil, errors.New("invalid event: action is nil")
	}
	if event.GetInstallation() == nil {
		return nil, errors.New("invalid event: installation is nil")
	}
	if event.GetInstallation().GetID() == 0 {
		return nil, errors.New("invalid installation: id is 0")
	}

	installationID := event.GetInstallation().GetID()
	installation, err := store.GetInstallationIDByAppID(ctx, installationID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("no installation found for id %d", installationID)
	}
	if err != nil {
		return nil, fmt.Errorf("could not determine provider id: %v", err)
	}
	if !installation.ProviderID.Valid {
		return nil, errors.New("invalid provider id")
	}
	if !installation.ProjectID.Valid {
		return nil, errors.New("invalid project id")
	}

	dbProv, err := store.GetProviderByID(ctx, installation.ProviderID.UUID)
	if err != nil {
		return nil, fmt.Errorf("could not determine provider id: %v", err)
	}

	providerConfig, _, err := clients.ParseAndMergeV1AppConfig(dbProv.Definition)
	if err != nil {
		return nil, fmt.Errorf("could not parse provider config: %v", err)
	}

	addedRepos := make([]*repo, 0)
	autoRegEntities := providerConfig.GetAutoRegistration().GetEntities()
	repoAutoReg, ok := autoRegEntities[string(pb.RepositoryEntity)]
	if ok && repoAutoReg.GetEnabled() {
		addedRepos = event.GetRepositoriesAdded()
	} else {
		zerolog.Ctx(ctx).Info().Msg("auto-registration is disabled for repositories")
	}

	results := make([]*processingResult, 0)
	for _, repo := range addedRepos {
		// caveat: we're accessing the database once for every
		// repository, which might be inefficient at scale.
		res, err := repositoryAdded(
			ctx,
			repo,
			installation,
		)
		if err != nil {
			return nil, err
		}

		results = append(results, res)
	}

	// We might want to ignore this case since we can only delete
	// repositories there were previously registered, which would
	// be deleted by means of "meta" and "repository" events as
	// well.
	for _, repo := range event.GetRepositoriesRemoved() {
		// caveat: we're accessing the database once for every
		// repository, which might be inefficient at scale.
		res, err := repositoryRemoved(
			repo,
		)
		if errors.Is(err, errRepoNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}

		results = append(results, res)
	}

	return results, nil
}

func repositoryRemoved(
	repo *repo,
) (*processingResult, error) {
	return sendEvaluateRepoMessage(repo, constants.TopicQueueGetEntityAndDelete)
}

func repositoryAdded(
	_ context.Context,
	repo *repo,
	installation db.ProviderGithubAppInstallation,
) (*processingResult, error) {
	if repo.GetName() == "" {
		return nil, errors.New("invalid repository name")
	}

	addRepoProps, err := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: properties.NumericalValueToUpstreamID(repo.GetID()),
		properties.PropertyName:       repo.GetFullName(),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating repository properties: %w", err)
	}

	event := messages.NewMinderEvent().
		WithProjectID(installation.ProjectID.UUID).
		WithProviderID(installation.ProviderID.UUID).
		WithEntityType(pb.Entity_ENTITY_REPOSITORIES).
		WithProperties(addRepoProps)

	return &processingResult{
		topic:   constants.TopicQueueReconcileEntityAdd,
		wrapper: event,
	}, nil
}
