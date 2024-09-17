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
	"strconv"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-github/v63/github"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/entities/models"
	"github.com/stacklok/minder/internal/entities/properties"
	"github.com/stacklok/minder/internal/entities/properties/service"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/projects/features"
	"github.com/stacklok/minder/internal/providers/github/clients"
	"github.com/stacklok/minder/internal/providers/github/installations"
	ghprop "github.com/stacklok/minder/internal/providers/github/properties"
	ghsvc "github.com/stacklok/minder/internal/providers/github/service"
	"github.com/stacklok/minder/internal/reconcilers/messages"
	reposvc "github.com/stacklok/minder/internal/repositories"
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

const (
	webhookActionEventDeleted     = "deleted"
	webhookActionEventOpened      = "opened"
	webhookActionEventReopened    = "reopened"
	webhookActionEventSynchronize = "synchronize"
	webhookActionEventClosed      = "closed"
	webhookActionEventPublished   = "published"
	webhookActionEventTransferred = "transferred"
)

// pingEvent are messages sent from GitHub to check the status of a
// specific webhook. Minder's processing of these events consists in
// just reporting the source.
type pingEvent struct {
	HookID *int64 `json:"hook_id,omitempty"`
	Repo   *repo  `json:"repository,omitempty"`
	Sender *user  `json:"sender,omitempty"`
}

func (p *pingEvent) GetRepo() *repo {
	return p.Repo
}

func (p *pingEvent) GetHookID() int64 {
	if p.HookID != nil {
		return *p.HookID
	}
	return 0
}

func (p *pingEvent) GetSender() *user {
	return p.Sender
}

// packageEvent represent any event related to a repository and one of
// its packages.
type packageEvent struct {
	Action  *string `json:"action,omitempty"`
	Repo    *repo   `json:"repository,omitempty"`
	Package *pkg    `json:"package,omitempty"`
}

type pkg struct {
	ID             *int64          `json:"id,omitempty"`
	Name           *string         `json:"name,omitempty"`
	PackageType    *string         `json:"package_type,omitempty"`
	PackageVersion *packageVersion `json:"package_version,omitempty"`
	Owner          *user           `json:"owner,omitempty"`
}

type user struct {
	ID      *int64  `json:"id,omitempty"`
	Login   *string `json:"login,omitempty"`
	HTMLURL *string `json:"html_url,omitempty"`
}

func (u *user) GetID() int64 {
	if u.ID != nil {
		return *u.ID
	}
	return 0
}

func (u *user) GetLogin() string {
	if u.Login != nil {
		return *u.Login
	}
	return ""
}

func (u *user) GetHTMLURL() string {
	if u.HTMLURL != nil {
		return *u.HTMLURL
	}
	return ""
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

// repoEvent represents any event related to a repository.
type repoEvent struct {
	Action *string `json:"action,omitempty"`
	Repo   *repo   `json:"repository,omitempty"`
	HookID *int64  `json:"hook_id,omitempty"`
}

func (r *repoEvent) GetAction() string {
	if r.Action != nil {
		return *r.Action
	}
	return ""
}

func (r *repoEvent) GetRepo() *repo {
	return r.Repo
}

func (r *repoEvent) GetHookID() int64 {
	if r.HookID != nil {
		return *r.HookID
	}
	return 0
}

type repo struct {
	ID       *int64  `json:"id,omitempty"`
	Name     *string `json:"name,omitempty"`
	FullName *string `json:"full_name,omitempty"`
	HTMLURL  *string `json:"html_url,omitempty"`
	Private  *bool   `json:"private,omitempty"`
}

func (r *repo) GetID() int64 {
	if r.ID != nil {
		return *r.ID
	}
	return 0
}

func (r *repo) GetName() string {
	if r.Name != nil {
		return *r.Name
	}
	return ""
}

func (r *repo) GetFullName() string {
	if r.FullName != nil {
		return *r.FullName
	}
	return ""
}

func (r *repo) GetHTMLURL() string {
	if r.HTMLURL != nil {
		return *r.HTMLURL
	}
	return ""
}

func (r *repo) GetPrivate() bool {
	if r.Private != nil {
		return *r.Private
	}
	return false
}

func (r *repo) GetOwner() string {
	if r.FullName != nil {
		parts := strings.SplitN(*r.FullName, "/", 2)
		// It is ok to always return the first item since it
		// defaults to empty string in case the string has no
		// separators.
		return parts[0]
	}
	return ""
}

// pullRequestEvent are events related to pull requests issued around
// a specific repository
type pullRequestEvent struct {
	Action      *string      `json:"action,omitempty"`
	Repo        *repo        `json:"repository,omitempty"`
	PullRequest *pullRequest `json:"pull_request,omitempty"`
}

func (p *pullRequestEvent) GetAction() string {
	if p.Action != nil {
		return *p.Action
	}
	return ""
}

func (p *pullRequestEvent) GetRepo() *repo {
	return p.Repo
}

func (p *pullRequestEvent) GetPullRequest() *pullRequest {
	return p.PullRequest
}

type pullRequest struct {
	ID     *int64  `json:"id,omitempty"`
	URL    *string `json:"url,omitempty"`
	Number *int64  `json:"number,omitempty"`
	User   *user   `json:"user,omitempty"`
}

func (p *pullRequest) GetID() int64 {
	if p.ID != nil {
		return *p.ID
	}
	return 0
}

func (p *pullRequest) GetURL() string {
	if p.URL != nil {
		return *p.URL
	}
	return ""
}

func (p *pullRequest) GetNumber() int64 {
	if p.Number != nil {
		return *p.Number
	}
	return 0
}

func (p *pullRequest) GetUser() *user {
	return p.User
}

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

// toMessage interface ensures that payloads returned by processor
// routines can be turned into a message.Message
type toMessage interface {
	ToMessage(*message.Message) error
}

var _ toMessage = (*entities.EntityInfoWrapper)(nil)
var _ toMessage = (*installations.InstallationInfoWrapper)(nil)
var _ toMessage = (*messages.MinderEvent)(nil)

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

// HandleGitHubAppWebhook handles incoming GitHub App webhooks
func (s *Server) HandleGitHubAppWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := zerolog.Ctx(ctx).With().Logger()

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
			l.Info().Err(err).Msg("Error validating webhook payload")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		wes.Typ = github.WebHookType(r)

		m := message.NewMessage(uuid.New().String(), nil)
		m.Metadata.Set(events.ProviderDeliveryIdKey, github.DeliveryID(r))
		// TODO: handle other sources
		m.Metadata.Set(events.ProviderSourceKey, "https://api.github.com/")
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

		var results []*processingResult
		var processingErr error

		switch github.WebHookType(r) {
		case "ping":
			// For ping events, we do not set wes.Accepted
			// to true because they're not relevant
			// business events.
			wes.Error = false
			s.processPingEvent(ctx, rawWBPayload)
		case "installation":
			wes.Accepted = true
			results, processingErr = s.processInstallationAppEvent(ctx, rawWBPayload)
		case "installation_repositories":
			wes.Accepted = true
			results, processingErr = s.processInstallationRepositoriesAppEvent(ctx, rawWBPayload)
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

			if err := s.evt.Publish(res.topic, m); err != nil {
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

// HandleGitHubWebHook handles incoming GitHub webhooks
// See https://docs.github.com/en/developers/webhooks-and-events/webhooks/about-webhooks
// for more information.
// nolint:gocyclo
func (s *Server) HandleGitHubWebHook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := zerolog.Ctx(ctx).With().Logger()

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
			res, processingErr = s.processRelevantRepositoryEvent(ctx, rawWBPayload)
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
			res, processingErr = s.processRepositoryEvent(ctx, rawWBPayload)
		case "package":
			// This is an artifact-related event, and can
			// only trigger a reconciliation.
			wes.Accepted = true
			res, processingErr = s.processPackageEvent(ctx, rawWBPayload)
		case "pull_request":
			wes.Accepted = true
			res, processingErr = s.processPullRequestEvent(ctx, rawWBPayload)
		case "ping":
			// For ping events, we do not set wes.Accepted
			// to true because they're not relevant
			// business events.
			wes.Error = false
			s.processPingEvent(ctx, rawWBPayload)
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
			if err := s.evt.Publish(res.topic, m); err != nil {
				wes.Error = true
				l.Error().Err(err).Msg("Error publishing message")
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
	payload []byte,
) {
	l := zerolog.Ctx(ctx).With().Logger()

	var event *pingEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		l.Info().Err(err).Msg("received malformed ping event")
		return
	}

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

//nolint:gocyclo // This function will be re-simplified later on
func (s *Server) processPackageEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	l := zerolog.Ctx(ctx)

	var event *packageEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	if event.Action == nil {
		return nil, errors.New("invalid event: action is nil")
	}
	if event.Package == nil || event.Repo == nil {
		l.Info().Msg("could not determine relevant entity for event. Skipping execution.")
		return nil, errNotHandled
	}

	// We only process events "package" with action "published",
	// i.e. we do not react to action "updated".
	if *event.Action != webhookActionEventPublished {
		return nil, errNotHandled
	}

	if event.Package.Owner == nil {
		return nil, errors.New("invalid package: owner is nil")
	}

	repoEnt, err := s.fetchRepo(ctx, event.Repo)
	if err != nil {
		return nil, err
	}

	provider, err := s.providerManager.InstantiateFromID(ctx, repoEnt.Entity.ProviderID)
	if err != nil {
		l.Error().Err(err).Msg("error instantiating provider")
		return nil, err
	}

	pkgLookupProps, err := packageEventToProperties(event)
	if err != nil {
		return nil, fmt.Errorf("error converting package event to properties: %w", err)
	}

	pkgName, err := provider.GetEntityName(pb.Entity_ENTITY_ARTIFACTS, pkgLookupProps)
	if err != nil {
		return nil, fmt.Errorf("error getting package name: %w", err)
	}

	var refreshedPkgProperties *properties.Properties
	ei, err := db.WithTransaction(s.store, func(tx db.ExtendQuerier) (*db.EntityInstance, error) {
		// we do two property lookups here: this first one will go away once we migrate artifacts to entities
		// as the only reason is to have the visibility and type of the artifact available.
		refreshedPkgProperties, err = s.props.RetrieveAllProperties(
			ctx, provider,
			repoEnt.Entity.ProjectID, repoEnt.Entity.ProviderID,
			pkgLookupProps, pb.Entity_ENTITY_ARTIFACTS,
			service.ReadBuilder().WithStoreOrTransaction(tx))

		if err != nil {
			return nil, fmt.Errorf("error retrieving properties: %w", err)
		}

		// TODO: remove this once we migrate artifacts to entities. We should get rid of the provider name.
		dbProv, getPrErr := tx.GetProviderByID(ctx, repoEnt.Entity.ProviderID)
		if getPrErr != nil {
			return nil, fmt.Errorf("error getting provider: %w", err)
		}

		dbArtifact, err := tx.UpsertArtifact(ctx, db.UpsertArtifactParams{
			RepositoryID: uuid.NullUUID{
				UUID:  repoEnt.Entity.ID,
				Valid: true,
			},
			ArtifactName:       refreshedPkgProperties.GetProperty(ghprop.ArtifactPropertyName).GetString(),
			ArtifactType:       refreshedPkgProperties.GetProperty(ghprop.ArtifactPropertyType).GetString(),
			ArtifactVisibility: refreshedPkgProperties.GetProperty(ghprop.ArtifactPropertyVisibility).GetString(),
			ProjectID:          repoEnt.Entity.ProjectID,
			ProviderID:         repoEnt.Entity.ProviderID,
			ProviderName:       dbProv.Name,
		})
		if err != nil {
			return nil, fmt.Errorf("error upserting artifact: %w", err)
		}

		ent, err := tx.CreateOrEnsureEntityByID(ctx, db.CreateOrEnsureEntityByIDParams{
			ID:         dbArtifact.ID,
			EntityType: db.EntitiesArtifact,
			Name:       pkgName,
			ProjectID:  repoEnt.Entity.ProjectID,
			ProviderID: repoEnt.Entity.ProviderID,
			OriginatedFrom: uuid.NullUUID{
				UUID:  repoEnt.Entity.ID,
				Valid: true,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("error creating or ensuring entity: %w", err)
		}

		// fetch the properties
		refreshedPkgProperties, err = s.props.RetrieveAllProperties(
			ctx, provider,
			ent.ProjectID, ent.ProviderID,
			refreshedPkgProperties, pb.Entity_ENTITY_ARTIFACTS,
			service.ReadBuilder().WithStoreOrTransaction(tx))
		if err != nil {
			return nil, fmt.Errorf("error retrieving properties: %w", err)
		}

		return &ent, nil
	})
	if err != nil {
		return nil, err
	}

	// refresh the version to attach it to the pb representation we send to the evaluation
	cli, err := provifv1.As[provifv1.GitHub](provider)
	if err != nil {
		l.Error().Err(err).Msg("error instantiating provider")
		return nil, err
	}

	version, err := gatherArtifactVersionInfo(ctx, cli, event,
		refreshedPkgProperties.GetProperty(ghprop.ArtifactPropertyOwner).GetString(),
		refreshedPkgProperties.GetProperty(ghprop.ArtifactPropertyName).GetString(),
	)
	if err != nil {
		return nil, fmt.Errorf("error extracting artifact from payload: %w", err)
	}

	ewp := models.NewEntityWithProperties(*ei, refreshedPkgProperties)
	pbMsg, err := s.props.EntityWithPropertiesAsProto(ctx, ewp, s.providerManager)
	if err != nil {
		return nil, fmt.Errorf("error converting artifact to protobuf: %w", err)
	}

	pbArtifact, ok := pbMsg.(*pb.Artifact)
	if !ok {
		return nil, errors.New("error converting proto message to protobuf")
	}
	pbArtifact.Versions = []*pb.ArtifactVersion{version}

	eiw := entities.NewEntityInfoWrapper().
		WithArtifact(pbArtifact).
		WithArtifactID(ei.ID).
		WithProjectID(repoEnt.Entity.ProjectID).
		WithProviderID(repoEnt.Entity.ProviderID).
		WithRepositoryID(repoEnt.Entity.ID)

	return &processingResult{topic: events.TopicQueueEntityEvaluate, wrapper: eiw}, nil
}

func (s *Server) processRelevantRepositoryEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	var event *repoEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	// Check fields mandatory for processing the event
	if event.GetRepo() == nil {
		return nil, errRepoNotFound
	}

	l := zerolog.Ctx(ctx).With().
		Str("github-event-action", event.GetAction()).
		Int64("github-repository-id", event.GetRepo().GetID()).
		Str("github-repository-url", event.GetRepo().GetHTMLURL()).
		Logger()

	if event.GetRepo().GetID() == 0 {
		return nil, errors.New("invalid repo: id is 0")
	}

	l.Info().Msg("handling event for repository")

	repoEntity, err := s.fetchRepo(ctx, event.GetRepo())
	if err != nil {
		return nil, err
	}

	hookId, hasHookErr := repoEntity.Properties.GetProperty(ghprop.RepoPropertyHookId).AsInt64()
	// This only makes sense for "meta" event type
	if event.GetHookID() != 0 && hasHookErr == nil {
		// Check if the payload webhook ID matches the one we
		// have stored in the DB for this repository
		if event.GetHookID() != hookId {
			// This means we got a deleted event for a
			// webhook ID that doesn't correspond to the
			// one we have stored in the DB.
			return nil, newErrNotHandled("meta event with action %s not handled, hook ID %d does not match stored webhook ID %d",
				event.GetAction(),
				event.GetHookID(),
				hookId,
			)
		}
	}

	// For webhook deletions, repository deletions, and repository
	// transfers, we issue a delete event with the correct message
	// type.
	if event.GetAction() == webhookActionEventDeleted ||
		event.GetAction() == webhookActionEventTransferred {
		repoEvent := messages.NewMinderEvent().
			WithProjectID(repoEntity.Entity.ProjectID).
			WithProviderID(repoEntity.Entity.ProviderID).
			WithEntityType("repository").
			WithEntityID(repoEntity.Entity.ID)

		return &processingResult{
			topic:   events.TopicQueueReconcileEntityDelete,
			wrapper: repoEvent,
		}, nil
	}

	// For all other actions, we trigger an evaluation.
	// protobufs are our API, so we always execute on these instead of the DB directly.
	pbMsg, err := s.props.EntityWithPropertiesAsProto(ctx, repoEntity, s.providerManager)
	if err != nil {
		return nil, fmt.Errorf("error converting repository to protobuf: %w", err)
	}

	pbRepo, ok := pbMsg.(*pb.Repository)
	if !ok {
		return nil, errors.New("error converting proto message to protobuf")
	}

	eiw := entities.NewEntityInfoWrapper().
		WithProjectID(repoEntity.Entity.ProjectID).
		WithProviderID(repoEntity.Entity.ProviderID).
		WithRepository(pbRepo).
		WithRepositoryID(repoEntity.Entity.ID)

	return &processingResult{
		topic:   events.TopicQueueEntityEvaluate,
		wrapper: eiw,
	}, nil
}

func (s *Server) processRepositoryEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	var event *repoEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	// Check fields mandatory for processing the event
	if event.GetRepo() == nil {
		return nil, errRepoNotFound
	}

	l := zerolog.Ctx(ctx).With().
		Str("github-event-action", event.GetAction()).
		Int64("github-repository-id", event.GetRepo().GetID()).
		Str("github-repository-url", event.GetRepo().GetHTMLURL()).
		Logger()

	if event.GetRepo().GetID() == 0 {
		return nil, errors.New("invalid repo: id is 0")
	}

	l.Info().Msg("handling event for repository")

	repoEnt, err := s.fetchRepo(ctx, event.GetRepo())
	if err != nil {
		return nil, err
	}

	// protobufs are our API, so we always execute on these instead of the DB directly.
	pbMsg, err := s.props.EntityWithPropertiesAsProto(ctx, repoEnt, s.providerManager)
	if err != nil {
		return nil, fmt.Errorf("error converting repository to protobuf: %w", err)
	}

	pbRepo, ok := pbMsg.(*pb.Repository)
	if !ok {
		return nil, errors.New("error converting proto message to protobuf")
	}

	eiw := entities.NewEntityInfoWrapper().
		WithProjectID(repoEnt.Entity.ProjectID).
		WithProviderID(repoEnt.Entity.ProviderID).
		WithRepository(pbRepo).
		WithRepositoryID(repoEnt.Entity.ID)

	return &processingResult{topic: events.TopicQueueEntityEvaluate, wrapper: eiw}, nil
}

// nolint:gocyclo // This function will be re-simplified real soon
func (s *Server) processPullRequestEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	l := zerolog.Ctx(ctx)

	var event *pullRequestEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	if event.GetAction() == "" {
		return nil, errors.New("invalid event: action is nil")
	}
	if event.GetRepo() == nil {
		return nil, errors.New("invalid event: repo is nil")
	}
	if event.GetPullRequest() == nil {
		return nil, errors.New("invalid event: pull request is nil")
	}
	if event.GetPullRequest().GetURL() == "" {
		return nil, errors.New("invalid pull request: URL is nil")
	}
	if event.GetPullRequest().GetNumber() == 0 {
		return nil, errors.New("invalid pull request: number is 0")
	}
	if event.GetPullRequest().GetUser() == nil {
		return nil, errors.New("invalid pull request: user is nil")
	}
	if event.GetPullRequest().GetUser().GetID() == 0 {
		return nil, errors.New("invalid user: id is 0")
	}

	ghRepo := event.GetRepo()
	repoEnt, err := s.fetchRepo(ctx, &repo{
		ID:      ghRepo.ID,
		HTMLURL: ghRepo.HTMLURL,
		Private: ghRepo.Private,
	})
	if err != nil {
		return nil, err
	}

	pullProps, err := properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: event.GetPullRequest().GetID(),
		ghprop.PullPropertyRepoName:   ghRepo.GetName(),
		ghprop.PullPropertyRepoOwner:  ghRepo.GetOwner(),
		ghprop.PullPropertyNumber:     event.GetPullRequest().GetNumber(),
		ghprop.PullPropertyAction:     event.GetAction(),
	})
	if err != nil {
		return nil, fmt.Errorf("error creating pull request properties: %w", err)
	}

	provider, err := s.providerManager.InstantiateFromID(ctx, repoEnt.Entity.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("error instantiating provider: %w", err)
	}

	prEntWithProps, err := s.reconcilePrWithDb(ctx, provider, repoEnt, pullProps)
	if errors.Is(err, errNotHandled) {
		return nil, err
	} else if err != nil {
		return nil, fmt.Errorf("error reconciling PR with DB: %w", err)
	}

	l.Info().Msgf("evaluating PR %s\n", event.GetPullRequest().GetURL())

	// protobufs are our API, so we always execute on these instead of the DB directly.
	pbMsg, err := s.props.EntityWithPropertiesAsProto(ctx, prEntWithProps, s.providerManager)
	if err != nil {
		return nil, fmt.Errorf("error converting repository to protobuf: %w", err)
	}

	pbPullRequest, ok := pbMsg.(*pb.PullRequest)
	if !ok {
		return nil, errors.New("error converting proto message to protobuf")
	}

	eiw := entities.NewEntityInfoWrapper().
		WithProjectID(repoEnt.Entity.ProjectID).
		WithProviderID(repoEnt.Entity.ProviderID).
		WithPullRequest(pbPullRequest).
		WithPullRequestID(prEntWithProps.Entity.ID).
		WithRepositoryID(repoEnt.Entity.ID)

	return &processingResult{topic: events.TopicQueueEntityEvaluate, wrapper: eiw}, nil
}

// processInstallationAppEvent processes events related to changes to
// the app itself as well as the list of accessible repositories.
//
// There are several possible actions, but in the current user flows
// we only process deletion.
func (_ *Server) processInstallationAppEvent(
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
		ghsvc.GitHubAppInstallationDeletedPayload{
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
func (s *Server) processInstallationRepositoriesAppEvent(
	ctx context.Context,
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
	installation, err := s.store.GetInstallationIDByAppID(ctx, installationID)
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

	dbProv, err := s.providerStore.GetByID(ctx, installation.ProviderID.UUID)
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
		res, err := s.repositoryAdded(
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
		res, err := s.repositoryRemoved(
			ctx,
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

func (s *Server) repositoryRemoved(
	ctx context.Context,
	repo *repo,
) (*processingResult, error) {
	repoEnt, err := s.fetchRepo(ctx, repo)
	if err != nil && !errors.Is(err, provifv1.ErrEntityNotFound) {
		return nil, err
	}

	event := messages.NewMinderEvent().
		WithProjectID(repoEnt.Entity.ProjectID).
		WithProviderID(repoEnt.Entity.ProviderID).
		WithEntityType("repository").
		WithEntityID(repoEnt.Entity.ID)

	return &processingResult{
		topic:   events.TopicQueueReconcileEntityDelete,
		wrapper: event,
	}, nil
}

func (_ *Server) repositoryAdded(
	_ context.Context,
	repo *repo,
	installation db.ProviderGithubAppInstallation,
) (*processingResult, error) {
	if repo.GetName() == "" {
		return nil, errors.New("invalid repository name")
	}

	event := messages.NewMinderEvent().
		WithProjectID(installation.ProjectID.UUID).
		WithProviderID(installation.ProviderID.UUID).
		WithEntityType("repository").
		WithAttribute("repoName", repo.GetName()).
		WithAttribute("repoOwner", repo.GetOwner())

	return &processingResult{
		topic:   events.TopicQueueReconcileEntityAdd,
		wrapper: event,
	}, nil
}

func (s *Server) fetchRepo(
	ctx context.Context,
	repo *repo,
) (*models.EntityWithProperties, error) {
	l := zerolog.Ctx(ctx)

	repoEnt, err := s.repos.RefreshRepositoryByUpstreamID(ctx, repo.GetID())
	if err != nil {
		if errors.Is(err, provifv1.ErrEntityNotFound) {
			l.Info().Msgf("repository %d not found upstream", repo.GetID())
			return repoEnt, err
		} else if errors.Is(err, reposvc.ErrRepoNotFound) {
			l.Info().Msgf("repository %d not found", repo.GetID())
			// no use in continuing if the repository doesn't exist
			return nil, fmt.Errorf("repository %d not found: %w",
				repo.GetID(),
				errRepoNotFound,
			)
		}
		return nil, fmt.Errorf("error getting repository: %w", err)
	}

	if repoEnt.Properties.GetProperty(properties.RepoPropertyIsPrivate).GetBool() {
		if !features.ProjectAllowsPrivateRepos(ctx, s.store, repoEnt.Entity.ProjectID) {
			return nil, errRepoIsPrivate
		}
	}

	if repoEnt.Entity.ProjectID.String() == "" {
		return nil, fmt.Errorf("no project found for repository %s: %w",
			repoEnt.Properties.GetProperty(properties.PropertyName).GetString(), errRepoNotFound)
	}

	return repoEnt, nil
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

// This routine assumes that all necessary validation is performed on
// the upper layer and accesses package and repo without checking for
// nulls.
func packageEventToProperties(
	event *packageEvent,
) (*properties.Properties, error) {
	if event.Repo.GetFullName() == "" {
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

	return properties.NewProperties(map[string]any{
		properties.PropertyUpstreamID: strconv.FormatInt(*event.Package.ID, 10),
		// we need these to look up the package properties
		ghprop.ArtifactPropertyOwner: owner,
		ghprop.ArtifactPropertyName:  *event.Package.Name,
		ghprop.ArtifactPropertyType:  strings.ToLower(*event.Package.PackageType),
	})
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
	provider provifv1.Provider,
	repoEnt *models.EntityWithProperties,
	pullProps *properties.Properties,
) (*models.EntityWithProperties, error) {
	l := zerolog.Ctx(ctx)

	var retErr error
	var retPr *models.EntityWithProperties

	pullName, err := provider.GetEntityName(pb.Entity_ENTITY_PULL_REQUESTS, pullProps)
	if err != nil {
		return nil, fmt.Errorf("error getting pull request name: %w", err)
	}

	switch pullProps.GetProperty(ghprop.PullPropertyAction).GetString() {
	case webhookActionEventOpened,
		webhookActionEventReopened,
		webhookActionEventSynchronize:
		var err error
		retPr, err = db.WithTransaction(s.store, func(t db.ExtendQuerier) (*models.EntityWithProperties, error) {
			dbPr, err := t.UpsertPullRequest(ctx, db.UpsertPullRequestParams{
				RepositoryID: repoEnt.Entity.ID,
				PrNumber:     pullProps.GetProperty(ghprop.PullPropertyNumber).GetInt64(),
			})
			if err != nil {
				return nil, err
			}

			prEnt, err := t.CreateOrEnsureEntityByID(ctx, db.CreateOrEnsureEntityByIDParams{
				ID:         dbPr.ID,
				EntityType: db.EntitiesPullRequest,
				Name:       pullName,
				ProjectID:  repoEnt.Entity.ProjectID,
				ProviderID: repoEnt.Entity.ProviderID,
				OriginatedFrom: uuid.NullUUID{
					UUID:  repoEnt.Entity.ID,
					Valid: true,
				},
			})
			if err != nil {
				return nil, err
			}

			refreshedProps, err := s.updatePullRequestInfoFromProvider(ctx, provider, repoEnt, pullProps, t)
			if err != nil {
				return nil, fmt.Errorf("error updating pull request information from provider: %w", err)
			}

			return models.NewEntityWithProperties(prEnt, refreshedProps), nil
		})
		if err != nil {
			return nil, fmt.Errorf(
				"cannot upsert PR %d in repo %s: %w",
				pullProps.GetProperty(ghprop.PullPropertyNumber).GetInt64(),
				repoEnt.Properties.GetProperty(properties.PropertyName).GetString(),
				err)
		}
		retErr = nil
	case webhookActionEventClosed:
		_, err := db.WithTransaction(s.store, func(t db.ExtendQuerier) (*db.PullRequest, error) {
			err := t.DeletePullRequest(ctx, db.DeletePullRequestParams{
				RepositoryID: repoEnt.Entity.ID,
				PrNumber:     pullProps.GetProperty(ghprop.PullPropertyNumber).GetInt64(),
			})
			if err != nil {
				return nil, err
			}

			err = t.DeleteEntityByName(ctx, db.DeleteEntityByNameParams{
				Name:      pullName,
				ProjectID: repoEnt.Entity.ProjectID,
			})
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}

			return nil, nil
		})

		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("cannot delete PR record %d in repo %s",
				pullProps.GetProperty(ghprop.PullPropertyNumber).GetInt64(),
				repoEnt.Properties.GetProperty(properties.PropertyName).GetString())
		}
		retPr = nil
		retErr = errNotHandled
	default:
		l.Info().Msgf("action %s is not handled for pull requests",
			pullProps.GetProperty(ghprop.PullPropertyAction).GetString())
		retPr = nil
		retErr = errNotHandled
	}

	return retPr, retErr
}

func (s *Server) updatePullRequestInfoFromProvider(
	ctx context.Context,
	provider provifv1.Provider,
	repoEnt *models.EntityWithProperties,
	pullProps *properties.Properties,
	qtx db.ExtendQuerier,
) (*properties.Properties, error) {
	// create properties.Name for the PR
	prName, err := provider.GetEntityName(pb.Entity_ENTITY_PULL_REQUESTS, pullProps)
	if err != nil {
		return nil, fmt.Errorf("error getting pull request name: %w", err)
	}

	lookupPropertiesMap := map[string]any{
		properties.PropertyName:       prName,
		properties.PropertyUpstreamID: pullProps.GetProperty(properties.PropertyUpstreamID).GetString(),
	}

	lookupProperties, err := properties.NewProperties(lookupPropertiesMap)
	if err != nil {
		return nil, fmt.Errorf("error creating properties: %w", err)
	}

	prProps, err := s.props.RetrieveAllProperties(ctx, provider,
		repoEnt.Entity.ProjectID, repoEnt.Entity.ProviderID,
		lookupProperties, pb.Entity_ENTITY_PULL_REQUESTS,
		service.ReadBuilder().WithStoreOrTransaction(qtx))

	if err != nil {
		return nil, fmt.Errorf("error retrieving properties: %w", err)
	}

	return prProps, nil
}
