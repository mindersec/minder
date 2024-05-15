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

// Package controlplane contains the control plane API for the minder.
package controlplane

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"sort"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-github/v61/github"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/artifacts"
	"github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/projects/features"
	"github.com/stacklok/minder/internal/providers/github/installations"
	ghprov "github.com/stacklok/minder/internal/providers/github/service"
	"github.com/stacklok/minder/internal/repositories"
	"github.com/stacklok/minder/internal/verifier/verifyif"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// errRepoNotFound is returned when a repository is not found
var errRepoNotFound = errors.New("repository not found")

// errArtifactNotFound is returned when an artifact is not found
var errArtifactNotFound = errors.New("artifact not found")

// errArtifactVersionSkipped is returned when an artifact is skipped because it has no tags
var errArtifactVersionSkipped = errors.New("artifact version skipped, has no tags")

// errRepoIsPrivate is returned when a repository is private
var errRepoIsPrivate = errors.New("repository is private")

// errNotHandled is returned when a webhook event is not handled
var errNotHandled = errors.New("webhook event not handled")

// newErrNotHandled returns a new errNotHandled error
func newErrNotHandled(smft string, args ...any) error {
	msg := fmt.Sprintf(smft, args...)
	return fmt.Errorf("%w: %s", errNotHandled, msg)
}

type packageEvent struct {
	Action  *string              `json:"action,omitempty"`
	Repo    *github.Repository   `json:"repository,omitempty"`
	Org     *github.Organization `json:"org,omitempty"`
	Package *pkg                 `json:"package,omitempty"`
}

type pkg struct {
	Name           *string         `json:"name,omitempty"`
	PackageType    *string         `json:"package_type,omitempty"`
	PackageVersion *packageVersion `json:"package_version,omitempty"`
	Owner          *github.User    `json:"owner,omitempty"`
}

type packageVersion struct {
	ID                *int64             `json:"id,omitempty"`
	Version           *string            `json:"version,omitempty"`
	ContainerMetadata *containerMetadata `json:"container_metadata,omitempty"`
}

type containerMetadata struct {
	Tag *tag `json:"tag,omitempty"`
}

type tag struct {
	Digest *string `json:"digest,omitempty"`
	Name   *string `json:"name,omitempty"`
}

type branchProtectionConfigurationEvent struct {
	Action *string              `json:"action,omitempty"`
	Repo   *github.Repository   `json:"repo,omitempty"`
	Org    *github.Organization `json:"org,omitempty"`
}

type repositoryAdvisoryEvent struct {
	Action *string              `json:"action,omitempty"`
	Repo   *github.Repository   `json:"repo,omitempty"`
	Org    *github.Organization `json:"org,omitempty"`
}

type repositoryRulesetEvent struct {
	Action *string              `json:"action,omitempty"`
	Repo   *github.Repository   `json:"repo,omitempty"`
	Org    *github.Organization `json:"org,omitempty"`
}

type secretScanningAlertLocationEvent struct {
	Action *string              `json:"action,omitempty"`
	Repo   *github.Repository   `json:"repo,omitempty"`
	Org    *github.Organization `json:"org,omitempty"`
}

const (
	webhookActionEventDeleted = "deleted"
	webhookActionEventOpened  = "opened"
	webhookActionEventClosed  = "closed"
)

type processingResult struct {
	topic string
	eiw   *entities.EntityInfoWrapper
}

// HandleGitHubAppWebhook handles incoming GitHub App webhooks
func (s *Server) HandleGitHubAppWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		wes := &metrics.WebhookEventState{
			Typ:      "unknown",
			Accepted: false,
			Error:    true,
		}
		defer func() {
			s.mt.AddWebhookEventTypeCount(r.Context(), wes)
		}()

		rawWBPayload, err := s.ghProviders.ValidateGitHubAppWebhookPayload(r)
		if err != nil {
			log.Printf("Error validating webhook payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		wes.Typ = github.WebHookType(r)
		event, err := github.ParseWebHook(github.WebHookType(r), rawWBPayload)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing github webhook message")
		}

		m := message.NewMessage(uuid.New().String(), nil)
		m.Metadata.Set(events.ProviderDeliveryIdKey, github.DeliveryID(r))
		// TODO: handle other sources
		m.Metadata.Set(events.ProviderSourceKey, "https://api.github.com/")
		m.Metadata.Set(events.GithubWebhookEventTypeKey, wes.Typ)

		l := zerolog.Ctx(ctx).With().
			Str("webhook-event-type", m.Metadata[events.GithubWebhookEventTypeKey]).
			Str("providertype", m.Metadata[events.ProviderTypeKey]).
			Str("upstream-delivery-id", m.Metadata[events.ProviderDeliveryIdKey]).
			Logger()

		wes.Accepted = true
		var res *processingResult
		var processingErr error
		switch event := event.(type) {
		case *github.PingEvent:
			// For ping events, we do not set wes.Accepted
			// to true because they're not relevant
			// business events.
			wes.Accepted = false
			wes.Error = false
			s.processPingEvent(ctx, event)
		case *github.InstallationEvent:
			res, processingErr = s.processInstallationAppEvent(ctx, event, m)
		default:
			l.Info().Msgf("webhook event %s not handled", wes.Typ)
		}

		if processingErr != nil {
			wes = handleParseError(wes.Typ, processingErr)
			if wes.Error {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			return
		}

		if res != nil {
			wes.Accepted = true
			l.Info().Str("message-id", m.UUID).Msg("publishing event for execution")

			if err := s.evt.Publish(installations.ProviderInstallationTopic, m); err != nil {
				wes.Error = true
				log.Printf("Error publishing message: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		wes.Error = false
		w.WriteHeader(http.StatusOK)
	}
}

// HandleGitHubWebHook handles incoming GitHub webhooks
// See https://docs.github.com/en/developers/webhooks-and-events/webhooks/about-webhooks
// for more information.
// nolint:gocyclo
func (s *Server) HandleGitHubWebHook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wes := &metrics.WebhookEventState{
			Typ:      "unknown",
			Accepted: false,
			Error:    true,
		}
		defer func() {
			s.mt.AddWebhookEventTypeCount(r.Context(), wes)
		}()

		// Validate the payload signature. This is required for security reasons.
		// See https://docs.github.com/en/developers/webhooks-and-events/webhooks/securing-your-webhooks
		// for more information. Note that this is not required for the GitHub App
		// webhook secret, but it is required for OAuth2 App.
		// it returns a uuid for the webhook, but we are not currently using it
		segments := strings.Split(r.URL.Path, "/")
		_ = segments[len(segments)-1]

		rawWBPayload, err := validatePayloadSignature(r, &s.cfg.WebhookConfig)
		if err != nil {
			log.Printf("Error validating webhook payload: %v", err)
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

		ctx := r.Context()
		l := zerolog.Ctx(ctx).With().
			Str("webhook-event-type", m.Metadata[events.GithubWebhookEventTypeKey]).
			Str("providertype", m.Metadata[events.ProviderTypeKey]).
			Str("upstream-delivery-id", m.Metadata[events.ProviderDeliveryIdKey]).
			Logger()

		l.Debug().Msg("parsing event")

		event, err := github.ParseWebHook(github.WebHookType(r), rawWBPayload)
		if err != nil {
			l.Error().Err(err).Msg("Error parsing github webhook message")
		}

		wes.Accepted = true
		var res *processingResult
		var processingErr error

		switch github.WebHookType(r) {
		// The following events are not available in go-github
		// and must be handled manually.
		case "branch_protection_configuration":
			res, processingErr = s.processBranchProtectionConfigurationEvent(ctx, rawWBPayload)
		case "repository_advisory":
			res, processingErr = s.processRepositoryAdvisoryEvent(ctx, rawWBPayload)
		case "repository_ruleset":
			res, processingErr = s.processRepositoryRulesetEvent(ctx, rawWBPayload)
		case "secret_scanning_alert_location":
			res, processingErr = s.processSecretScanningAlertLocationEvent(ctx, rawWBPayload)
		case "package":
			// This is an artifact-related event, and can
			// only trigger a reconciliation.
			res, processingErr = s.processPackageEvent(ctx, rawWBPayload)
		}

		switch event := event.(type) {
		case *github.PingEvent:
			// For ping events, we do not set wes.Accepted
			// to true because they're not relevant
			// business events.
			wes.Accepted = false
			wes.Error = false
			s.processPingEvent(ctx, event)
		case *github.MetaEvent:
			// As per github documentation, MetaEvent is
			// triggered when the webhook that this event
			// is configured on is deleted.
			//
			// Our action here is to de-register the
			// related repo.
			res, processingErr = s.processMetaEvent(ctx, event)

		// All these events are related to a repo and usually
		// contain an action. They all trigger a
		// reconciliation or, in some cases, a deletion.
		case *github.BranchProtectionRuleEvent:
			res, processingErr = s.processBranchProtectionRuleEvent(ctx, event)
		case *github.CodeScanningAlertEvent:
			res, processingErr = s.processCodeScanningAlertEvent(ctx, event)
		case *github.CreateEvent:
			res, processingErr = s.processCreateEvent(ctx, event)
		case *github.MemberEvent:
			res, processingErr = s.processMemberEvent(ctx, event)
		case *github.PublicEvent:
			res, processingErr = s.processPublicEvent(ctx, event)
		case *github.RepositoryEvent:
			res, processingErr = s.processRepositoryEvent(ctx, event)
		case *github.RepositoryImportEvent:
			res, processingErr = s.processRepositoryImportEvent(ctx, event)
		case *github.SecretScanningAlertEvent:
			res, processingErr = s.processSecretScanningAlertEvent(ctx, event)
		case *github.TeamAddEvent:
			res, processingErr = s.processTeamAddEvent(ctx, event)
		case *github.TeamEvent:
			res, processingErr = s.processTeamEvent(ctx, event)
		case *github.RepositoryVulnerabilityAlertEvent:
			res, processingErr = s.processRepositoryVulnerabilityAlertEvent(ctx, event)
		case *github.SecurityAdvisoryEvent:
			res, processingErr = s.processSecurityAdvisoryEvent(ctx, event)
		case *github.SecurityAndAnalysisEvent:
			res, processingErr = s.processSecurityAndAnalysisEvent(ctx, event)
		case *github.OrgBlockEvent,
			*github.OrganizationEvent:
			l.Info().Msgf("webhook events %s do not contain repo", wes.Typ)
		case *github.PushEvent:
			// TODO has GetRepo but the type is different :|
		case *github.PullRequestEvent:
			res, processingErr = s.processPullRequestEvent(ctx, event)
		default:
			l.Info().Msgf("webhook event %s not handled", wes.Typ)
		}

		if processingErr != nil {
			wes = handleParseError(wes.Typ, processingErr)
			if wes.Error {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			return
		}

		// res is null only when a ping event occurred.
		if res != nil {
			if err := res.eiw.ToMessage(m); err != nil {
				wes.Error = true
				log.Printf("Error creating event: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Publish the message to the event router
			if err := s.evt.Publish(res.topic, m); err != nil {
				wes.Error = true
				log.Printf("Error publishing message: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		// We successfully published the message
		wes.Error = false
		w.WriteHeader(http.StatusOK)
	}
}

// processPingEvent logs the type of token used to authenticate the
// webhook. The idea is to log a link between the repo and the token
// type. Since this is done only for the ping event, we can assume
// that the sender is the app that installed the webhook on the
// repository.
func (_ *Server) processPingEvent(
	ctx context.Context,
	event *github.PingEvent,
) {
	l := zerolog.Ctx(ctx).With().Logger()

	if event.GetRepo() != nil {
		l = l.With().Int64("github-repository-id", event.GetRepo().GetID()).Logger()
		l = l.With().Str("github-repository-url", event.GetRepo().GetHTMLURL()).Logger()
	}
	if event.GetSender() != nil {
		l = l.With().Str("sender-login", event.GetSender().GetLogin()).Logger()
		l = l.With().Str("github-repository-url", event.GetSender().GetHTMLURL()).Logger()
		if strings.Contains(event.GetSender().GetHTMLURL(), "github.com/apps") {
			l = l.With().Str("sender-token-type", "github-app").Logger()
		} else {
			l = l.With().Str("sender-token-type", "oauth-app").Logger()
		}
	}

	l.Debug().Msg("ping received")
}

func (s *Server) processPackageEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	var event *packageEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	if event.Action == nil {
		return nil, errors.New("invalid event action")
	}
	if event.Package == nil || event.Repo == nil {
		log.Printf("could not determine relevant entity for event. Skipping execution.")
		return nil, nil // this is awkward
	}

	if event.Package.Owner == nil {
		return nil, errors.New("could not determine articfact owner")
	}

	dbrepo, err := s.fetchRepo(ctx, event.Repo)
	if err != nil {
		return nil, err
	}

	provider, err := s.providerManager.InstantiateFromID(ctx, dbrepo.ProviderID)
	if err != nil {
		log.Printf("error instantiating provider: %v", err)
		return nil, err
	}

	cli, err := provifv1.As[provifv1.GitHub](provider)
	if err != nil {
		log.Printf("error instantiating provider: %v", err)
		return nil, err
	}

	tempArtifact, err := gatherArtifact(ctx, cli, event)
	if err != nil {
		return nil, fmt.Errorf("error gathering versioned artifact: %w", err)
	}

	dbArtifact, err := s.store.UpsertArtifact(ctx, db.UpsertArtifactParams{
		RepositoryID: uuid.NullUUID{
			UUID:  dbrepo.ID,
			Valid: true,
		},
		ArtifactName:       tempArtifact.GetName(),
		ArtifactType:       tempArtifact.GetTypeLower(),
		ArtifactVisibility: tempArtifact.Visibility,
		ProjectID:          dbrepo.ProjectID,
		ProviderID:         dbrepo.ProviderID,
		ProviderName:       dbrepo.Provider,
	})
	if err != nil {
		return nil, fmt.Errorf("error upserting artifact: %w", err)
	}

	_, pbArtifact, err := artifacts.GetArtifact(ctx, s.store, dbrepo.ProjectID, dbArtifact.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting artifact with versions: %w", err)
	}
	// TODO: wrap in a function
	pbArtifact.Versions = tempArtifact.Versions

	eiw := entities.NewEntityInfoWrapper().
		WithArtifact(pbArtifact).
		WithProviderID(dbrepo.ProviderID).
		WithProjectID(dbrepo.ProjectID).
		WithRepositoryID(dbrepo.ID).
		WithArtifactID(dbArtifact.ID).
		WithActionEvent(*event.Action)

	return &processingResult{topic: events.TopicQueueEntityEvaluate, eiw: eiw}, nil
}

func (s *Server) processBranchProtectionRuleEvent(
	ctx context.Context,
	event *github.BranchProtectionRuleEvent,
) (*processingResult, error) {
	repo := event.GetRepo()
	action := event.GetAction()
	return s.innerProcessGenericRepositoryEvent(ctx, action, repo)
}

func (s *Server) processCodeScanningAlertEvent(
	ctx context.Context,
	event *github.CodeScanningAlertEvent,
) (*processingResult, error) {
	repo := event.GetRepo()
	action := event.GetAction()
	return s.innerProcessGenericRepositoryEvent(ctx, action, repo)
}

func (s *Server) processCreateEvent(
	ctx context.Context,
	event *github.CreateEvent,
) (*processingResult, error) {
	repo := event.GetRepo()
	action := ""
	return s.innerProcessGenericRepositoryEvent(ctx, action, repo)
}

func (s *Server) processMemberEvent(
	ctx context.Context,
	event *github.MemberEvent,
) (*processingResult, error) {
	repo := event.GetRepo()
	action := event.GetAction()
	return s.innerProcessGenericRepositoryEvent(ctx, action, repo)
}

func (s *Server) processPublicEvent(
	ctx context.Context,
	event *github.PublicEvent,
) (*processingResult, error) {
	repo := event.GetRepo()
	action := ""
	return s.innerProcessGenericRepositoryEvent(ctx, action, repo)
}

func (s *Server) processRepositoryEvent(
	ctx context.Context,
	event *github.RepositoryEvent,
) (*processingResult, error) {
	repo := event.GetRepo()
	action := event.GetAction()
	return s.innerProcessGenericRepositoryEvent(ctx, action, repo)
}

func (s *Server) processRepositoryImportEvent(
	ctx context.Context,
	event *github.RepositoryImportEvent,
) (*processingResult, error) {
	repo := event.GetRepo()
	action := ""
	return s.innerProcessGenericRepositoryEvent(ctx, action, repo)
}

func (s *Server) processSecretScanningAlertEvent(
	ctx context.Context,
	event *github.SecretScanningAlertEvent,
) (*processingResult, error) {
	repo := event.GetRepo()
	action := event.GetAction()
	return s.innerProcessGenericRepositoryEvent(ctx, action, repo)
}

func (s *Server) processTeamAddEvent(
	ctx context.Context,
	event *github.TeamAddEvent,
) (*processingResult, error) {
	repo := event.GetRepo()
	action := ""
	return s.innerProcessGenericRepositoryEvent(ctx, action, repo)
}

func (s *Server) processTeamEvent(
	ctx context.Context,
	event *github.TeamEvent,
) (*processingResult, error) {
	repo := event.GetRepo()
	action := event.GetAction()
	return s.innerProcessGenericRepositoryEvent(ctx, action, repo)
}

func (s *Server) processRepositoryVulnerabilityAlertEvent(
	ctx context.Context,
	event *github.RepositoryVulnerabilityAlertEvent,
) (*processingResult, error) {
	repo := event.GetRepository()
	action := event.GetAction()
	return s.innerProcessGenericRepositoryEvent(ctx, action, repo)
}

func (s *Server) processSecurityAdvisoryEvent(
	ctx context.Context,
	event *github.SecurityAdvisoryEvent,
) (*processingResult, error) {
	repo := event.GetRepository()
	action := event.GetAction()
	return s.innerProcessGenericRepositoryEvent(ctx, action, repo)
}

func (s *Server) processSecurityAndAnalysisEvent(
	ctx context.Context,
	event *github.SecurityAndAnalysisEvent,
) (*processingResult, error) {
	repo := event.GetRepository()
	action := ""
	return s.innerProcessGenericRepositoryEvent(ctx, action, repo)
}

func (s *Server) innerProcessGenericRepositoryEvent(
	ctx context.Context,
	action string,
	repo *github.Repository,
) (*processingResult, error) {
	// Check fields mandatory for processing the event
	if repo == nil {
		return nil, errRepoNotFound
	}
	if repo.GetID() == 0 {
		return nil, errors.New("event repo id is null")
	}

	log.Printf("handling event for repository %d", repo.GetID())

	dbrepo, err := s.fetchRepo(ctx, repo)
	if err != nil {
		return nil, err
	}

	// protobufs are our API, so we always execute on these instead of the DB directly.
	pbRepo := repositories.PBRepositoryFromDB(*dbrepo)
	eiw := entities.NewEntityInfoWrapper().
		WithProviderID(dbrepo.ProviderID).
		WithRepository(pbRepo).
		WithProjectID(dbrepo.ProjectID).
		WithRepositoryID(dbrepo.ID).
		WithActionEvent(action)

	topic := events.TopicQueueEntityEvaluate
	if action == webhookActionEventDeleted {
		topic = events.TopicQueueReconcileEntityDelete
	}

	return &processingResult{topic: topic, eiw: eiw}, nil
}

// processMetaEvent handles events related to the webhook itself. As
// per GitHub's documentation, the only possible action is "deleted",
// in which case we have to de-register the related repo.
func (s *Server) processMetaEvent(
	ctx context.Context,
	event *github.MetaEvent,
) (*processingResult, error) {
	// Check fields mandatory for processing the event
	if event.GetAction() != webhookActionEventDeleted {
		// "deleted" is the only allowed action for "meta"
		// events
		return nil, errors.New(`event action is not "deleted"`)
	}
	if event.GetRepo() == nil {
		return nil, errRepoNotFound
	}
	if event.GetRepo().GetID() == 0 {
		return nil, errors.New("event repo id is null")
	}

	log.Printf("handling event for repository %d", event.GetRepo().GetID())

	dbrepo, err := s.fetchRepo(ctx, event.GetRepo())
	if err != nil {
		return nil, err
	}

	if dbrepo.WebhookID.Valid {
		// Check if the payload webhook ID matches the one we
		// have stored in the DB for this repository
		if event.GetHookID() != dbrepo.WebhookID.Int64 {
			// This means we got a deleted event for a
			// webhook ID that doesn't correspond to the
			// one we have stored in the DB.
			return nil, newErrNotHandled("meta event with action %s not handled, hook ID %d does not match stored webhook ID %d",
				event.GetAction(),
				event.GetHookID(),
				dbrepo.WebhookID.Int64,
			)
		}
	}
	// If we get this far it means we got a deleted event for a
	// webhook ID that corresponds to the one we have stored in
	// the DB.  We will remove the repo from the DB, so we can
	// proceed with the deletion event for this entity
	// (repository).

	// TODO: perhaps handle this better by trying to re-create the
	// webhook if it was deleted manually

	// protobufs are our API, so we always execute on these instead of the DB directly.
	repo := repositories.PBRepositoryFromDB(*dbrepo)
	eiw := entities.NewEntityInfoWrapper().
		WithProviderID(dbrepo.ProviderID).
		WithRepository(repo).
		WithProjectID(dbrepo.ProjectID).
		WithRepositoryID(dbrepo.ID).
		WithActionEvent(event.GetAction())

	return &processingResult{topic: events.TopicQueueReconcileEntityDelete, eiw: eiw}, nil
}

func (s *Server) processBranchProtectionConfigurationEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	event := branchProtectionConfigurationEvent{}
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}
	if event.Action == nil {
		return nil, errors.New("event has no action")
	}

	return s.innerProcessGenericRepositoryEvent(ctx, *event.Action, event.Repo)
}

func (s *Server) processRepositoryAdvisoryEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	event := repositoryAdvisoryEvent{}
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}
	if event.Action == nil {
		return nil, errors.New("event has no action")
	}

	return s.innerProcessGenericRepositoryEvent(ctx, *event.Action, event.Repo)
}

func (s *Server) processRepositoryRulesetEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	event := repositoryRulesetEvent{}
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}
	if event.Action == nil {
		return nil, errors.New("event has no action")
	}

	return s.innerProcessGenericRepositoryEvent(ctx, *event.Action, event.Repo)
}

func (s *Server) processSecretScanningAlertLocationEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	event := secretScanningAlertLocationEvent{}
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}
	if event.Action == nil {
		return nil, errors.New("event has no action")
	}

	return s.innerProcessGenericRepositoryEvent(ctx, *event.Action, event.Repo)
}

func (s *Server) processPullRequestEvent(
	ctx context.Context,
	event *github.PullRequestEvent,
) (*processingResult, error) {
	if event.GetAction() == "" {
		return nil, errors.New("event has no action")
	}
	if event.GetRepo() == nil {
		return nil, errors.New("event has no repo")
	}
	if event.GetPullRequest() == nil {
		return nil, errors.New("pull request is null")
	}
	if event.GetPullRequest().GetURL() == "" {
		return nil, errors.New("pull request has no URL")
	}
	if event.GetPullRequest().GetNumber() == 0 {
		return nil, errors.New("pull request has no number")
	}
	if event.GetPullRequest().GetUser() == nil {
		return nil, errors.New("pull request has no user")
	}
	if event.GetPullRequest().GetUser().GetID() == 0 {
		return nil, errors.New("pull request user has no id")
	}

	dbrepo, err := s.fetchRepo(ctx, event.GetRepo())
	if err != nil {
		return nil, err
	}

	provider, err := s.providerManager.InstantiateFromID(ctx, dbrepo.ProviderID)
	if err != nil {
		log.Printf("error instantiating provider: %v", err)
		return nil, err
	}

	cli, err := provifv1.As[provifv1.GitHub](provider)
	if err != nil {
		log.Printf("error instantiating provider: %v", err)
		return nil, err
	}

	prEvalInfo := &pb.PullRequest{
		Url:      event.GetPullRequest().GetURL(),
		Number:   int64(event.GetPullRequest().GetNumber()),
		AuthorId: int64(event.GetPullRequest().GetUser().GetID()),
		Action:   event.GetAction(),
	}

	dbPr, err := s.reconcilePrWithDb(ctx, *dbrepo, prEvalInfo)
	if errors.Is(err, errNotHandled) {
		return nil, err
	} else if err != nil {
		return nil, fmt.Errorf("error reconciling PR with DB: %w", err)
	}

	err = updatePullRequestInfoFromProvider(ctx, cli, *dbrepo, prEvalInfo)
	if err != nil {
		return nil, fmt.Errorf("error updating pull request information from provider: %w", err)
	}

	log.Printf("evaluating PR %+v", prEvalInfo)

	eiw := entities.NewEntityInfoWrapper().
		WithPullRequest(prEvalInfo).
		WithPullRequestID(dbPr.ID).
		WithProviderID(dbrepo.ProviderID).
		WithProjectID(dbrepo.ProjectID).
		WithRepositoryID(dbrepo.ID).
		WithActionEvent(event.GetAction())

	return &processingResult{topic: events.TopicQueueReconcileEntityDelete, eiw: eiw}, nil
}

// processInstallationAppEvent processes events related to changes to
// the app itself as well as the list of accessible repositories.
//
// There are several possible actions, but in the current user flows
// we only process.
func (_ *Server) processInstallationAppEvent(
	_ context.Context,
	event *github.InstallationEvent,
	msg *message.Message,
) (*processingResult, error) {
	// Check fields mandatory for processing the event
	if event.GetAction() == "" {
		return nil, errors.New("invalid event action")
	}
	if event.GetAction() != webhookActionEventDeleted {
		return nil, newErrNotHandled("event %s with action %s not handled",
			msg.Metadata.Get(events.GithubWebhookEventTypeKey),
			event.GetAction(),
		)
	}
	if event.GetInstallation() == nil {
		return nil, errors.New("event ")
	}
	if event.GetInstallation().GetID() == 0 {
		return nil, fmt.Errorf("installation ID is 0")
	}

	payloadBytes, err := json.Marshal(
		ghprov.GitHubAppInstallationDeletedPayload{
			InstallationID: event.GetInstallation().GetID(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload: %w", err)
	}

	// We could use something like an EntityInfoWrapper here as
	// well.
	installations.ProviderInstanceRemovedMessage(
		msg,
		db.ProviderClassGithubApp,
		payloadBytes)

	res := &processingResult{
		topic: installations.ProviderInstallationTopic,
	}

	return res, nil
}

func (s *Server) fetchRepo(
	ctx context.Context,
	repo *github.Repository,
) (*db.Repository, error) {
	dbrepo, err := s.store.GetRepositoryByRepoID(ctx, repo.GetID())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("repository %d not found", repo.GetID())
			// no use in continuing if the repository doesn't exist
			return nil, fmt.Errorf("repository %d not found: %w",
				repo.GetID(),
				errRepoNotFound,
			)
		}
		return nil, fmt.Errorf("error getting repository: %w", err)
	}

	if repo.GetPrivate() {
		if !features.ProjectAllowsPrivateRepos(ctx, s.store, dbrepo.ProjectID) {
			return nil, errRepoIsPrivate
		}
	}

	if dbrepo.ProjectID.String() == "" {
		return nil, fmt.Errorf("no project found for repository %s/%s: %w",
			dbrepo.RepoOwner, dbrepo.RepoName, errRepoNotFound)
	}

	return &dbrepo, nil
}

// NoopWebhookHandler is a no-op handler for webhooks
func (s *Server) NoopWebhookHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wes := &metrics.WebhookEventState{
			Typ:      "unknown",
			Accepted: false,
			Error:    true,
		}
		defer func() {
			s.mt.AddWebhookEventTypeCount(r.Context(), wes)
		}()

		wes.Typ = github.WebHookType(r)
		wes.Accepted = true
		wes.Error = false
		w.WriteHeader(http.StatusOK)
	}
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

func handleParseError(typ string, parseErr error) *metrics.WebhookEventState {
	state := &metrics.WebhookEventState{Typ: typ, Accepted: false, Error: true}

	var logMsg string
	switch {
	case errors.Is(parseErr, errRepoNotFound):
		state.Error = false
		logMsg = "repository not found"
	case errors.Is(parseErr, errArtifactNotFound):
		state.Error = false
		logMsg = "artifact not found"
	case errors.Is(parseErr, errRepoIsPrivate):
		state.Error = false
		logMsg = "repository is private"
	case errors.Is(parseErr, errNotHandled):
		state.Error = false
		logMsg = fmt.Sprintf("webhook event not handled (%v)", parseErr)
	case errors.Is(parseErr, errArtifactVersionSkipped):
		state.Error = false
		logMsg = "artifact version skipped, has no tags"
	default:
		logMsg = fmt.Sprintf("Error parsing github webhook message: %v", parseErr)
	}
	log.Print(logMsg)
	return state
}

// This routine assumes that all necessary validation is performed on
// the upper layer and accesses package and repo without checking for
// nulls.
func gatherArtifactInfo(
	ctx context.Context,
	client provifv1.GitHub,
	event *packageEvent,
) (*pb.Artifact, error) {
	if event.Repo.FullName == nil {
		return nil, errors.New("invalid package: full name is nil")
	}
	if event.Package.Name == nil {
		return nil, errors.New("invalid package: name is nil")
	}
	if event.Package.PackageType == nil {
		return nil, errors.New("invalid package: package type is nil")
	}

	owner := ""
	if event.Package.Owner != nil {
		owner = event.Package.Owner.GetLogin()
	}

	artifact := &pb.Artifact{
		Owner:      owner,
		Name:       *event.Package.Name,
		Type:       *event.Package.PackageType,
		Repository: *event.Repo.FullName,
		// visibility and createdAt are not in the payload, we need to get it with a REST call
	}

	// we also need to fill in the visibility which is not in the payload
	ghArtifact, err := client.GetPackageByName(
		ctx,
		artifact.Owner,
		string(verifyif.ArtifactTypeContainer),
		artifact.Name,
	)
	if err != nil {
		return nil, fmt.Errorf("error extracting artifact from repo: %w", err)
	}

	artifact.Visibility = *ghArtifact.Visibility
	return artifact, nil
}

// This routine assumes that all necessary validation is performed on
// the upper layer and accesses package and repo without checking for
// nulls.
func gatherArtifactVersionInfo(
	ctx context.Context,
	cli provifv1.GitHub,
	event *packageEvent,
	artifactOwnerLogin, artifactName string,
) (*pb.ArtifactVersion, error) {
	if event.Package.PackageVersion == nil {
		return nil, errors.New("invalid package version: nil")
	}

	pv := event.Package.PackageVersion
	if pv.ID == nil {
		return nil, errors.New("invalid package version: id is nil")
	}
	if pv.Version == nil {
		return nil, errors.New("invalid package version: version is nil")
	}
	if pv.ContainerMetadata == nil {
		return nil, errors.New("invalid package version: container metadata is nil")
	}
	if pv.ContainerMetadata.Tag == nil {
		return nil, errors.New("invalid container metadata: tag is nil")
	}
	if pv.ContainerMetadata.Tag.Name == nil {
		return nil, errors.New("invalid container metadata tag: name is nil")
	}

	version := &pb.ArtifactVersion{
		VersionId: *pv.ID,
		Tags:      []string{*pv.ContainerMetadata.Tag.Name},
		Sha:       *pv.Version,
	}

	// not all information is in the payload, we need to get it from the container registry
	// and/or GH API
	if err := updateArtifactVersionFromRegistry(
		ctx,
		cli,
		artifactOwnerLogin,
		artifactName,
		version,
	); err != nil {
		return nil, fmt.Errorf("error getting upstream information for artifact version: %w", err)
	}

	return version, nil
}

func gatherArtifact(
	ctx context.Context,
	cli provifv1.GitHub,
	event *packageEvent,
) (*pb.Artifact, error) {
	artifact, err := gatherArtifactInfo(ctx, cli, event)
	if err != nil {
		return nil, fmt.Errorf("error gatherinfo artifact info: %w", err)
	}

	version, err := gatherArtifactVersionInfo(ctx, cli, event, artifact.Owner, artifact.Name)
	if err != nil {
		return nil, fmt.Errorf("error extracting artifact from payload: %w", err)
	}
	artifact.Versions = []*pb.ArtifactVersion{version}
	return artifact, nil
}

func updateArtifactVersionFromRegistry(
	ctx context.Context,
	client provifv1.GitHub,
	artifactOwnerLogin, artifactName string,
	version *pb.ArtifactVersion,
) error {
	// we'll grab the artifact version from the REST endpoint because we need the visibility
	// and createdAt fields which are not in the payload
	ghVersion, err := client.GetPackageVersionById(ctx, artifactOwnerLogin, string(verifyif.ArtifactTypeContainer),
		artifactName, version.VersionId)
	if err != nil {
		return fmt.Errorf("error getting package version from repository: %w", err)
	}

	tags := ghVersion.Metadata.Container.Tags
	// if the artifact has no tags, skip it
	if len(tags) == 0 {
		return errArtifactVersionSkipped
	}

	sort.Strings(tags)

	version.Tags = tags
	if ghVersion.CreatedAt != nil {
		version.CreatedAt = timestamppb.New(*ghVersion.CreatedAt.GetTime())
	}
	return nil
}

func (s *Server) reconcilePrWithDb(
	ctx context.Context,
	dbrepo db.Repository,
	prEvalInfo *pb.PullRequest,
) (*db.PullRequest, error) {
	var retErr error
	var retPr *db.PullRequest

	switch prEvalInfo.Action {
	// TODO mic go-github documentation reportes that
	// PullRequestEvents with action "synchronize" are not
	// published, see here
	// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PullRequestEvent
	case webhookActionEventOpened:
		dbPr, err := s.store.UpsertPullRequest(ctx, db.UpsertPullRequestParams{
			RepositoryID: dbrepo.ID,
			PrNumber:     prEvalInfo.Number,
		})
		if err != nil {
			return nil, fmt.Errorf(
				"cannot upsert PR %d in repo %s/%s",
				prEvalInfo.Number, dbrepo.RepoOwner, dbrepo.RepoName)
		}
		retPr = &dbPr
		retErr = nil
	case webhookActionEventClosed:
		err := s.store.DeletePullRequest(ctx, db.DeletePullRequestParams{
			RepositoryID: dbrepo.ID,
			PrNumber:     prEvalInfo.Number,
		})
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("cannot delete PR record %d in repo %s/%s",
				prEvalInfo.Number, dbrepo.RepoOwner, dbrepo.RepoName)
		}
		retPr = nil
		retErr = errNotHandled
	default:
		log.Printf("action %s is not handled for pull requests", prEvalInfo.Action)
		retPr = nil
		retErr = errNotHandled
	}

	return retPr, retErr
}

func updatePullRequestInfoFromProvider(
	ctx context.Context,
	cli provifv1.GitHub,
	dbrepo db.Repository,
	prEvalInfo *pb.PullRequest,
) error {
	prReply, err := cli.GetPullRequest(ctx, dbrepo.RepoOwner, dbrepo.RepoName, int(prEvalInfo.Number))
	if err != nil {
		return fmt.Errorf("error getting pull request: %w", err)
	}

	prEvalInfo.CommitSha = *prReply.Head.SHA
	prEvalInfo.RepoOwner = dbrepo.RepoOwner
	prEvalInfo.RepoName = dbrepo.RepoName
	return nil
}
