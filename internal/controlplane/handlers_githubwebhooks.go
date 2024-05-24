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

const (
	webhookActionEventDeleted   = "deleted"
	webhookActionEventOpened    = "opened"
	webhookActionEventReopened  = "reopened"
	webhookActionEventClosed    = "closed"
	webhookActionEventPublished = "published"
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
	URL    *string `json:"url,omitempty"`
	Number *int64  `json:"number,omitempty"`
	User   *user   `json:"user,omitempty"`
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

type installation struct {
	ID *int64 `json:"id,omitempty"`
}

func (i *installation) GetID() int64 {
	if i.ID != nil {
		return *i.ID
	}
	return 0
}

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
	eiw *entities.EntityInfoWrapper
	// wrapper object for installation (app) events
	iiw *installations.InstallationInfoWrapper
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
		ctx = l.WithContext(ctx)

		wes.Accepted = true
		var res *processingResult
		var processingErr error

		switch github.WebHookType(r) {
		case "ping":
			// For ping events, we do not set wes.Accepted
			// to true because they're not relevant
			// business events.
			wes.Accepted = false
			wes.Error = false
			s.processPingEvent(ctx, rawWBPayload)
		case "installation":
			res, processingErr = s.processInstallationAppEvent(ctx, rawWBPayload)
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
			l.Info().Str("message-id", m.UUID).Msg("publishing event for execution")
			if err := res.iiw.ToMessage(m); err != nil {
				wes.Error = true
				log.Printf("Error creating event: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			if err := s.evt.Publish(res.topic, m); err != nil {
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
		ctx = l.WithContext(ctx)

		l.Debug().Msg("parsing event")

		wes.Accepted = true
		var res *processingResult
		var processingErr error

		// The following two switch statements are effectively
		// mutually exclusive and the set of events that they
		// manage is non-overlapping.
		//
		// This is not verified statically and it is not yet
		// verified via tests, but should be easy enough to
		// verify by inspection.

		switch github.WebHookType(r) {
		// All these events are related to a repo and usually
		// contain an action. They all trigger a
		// reconciliation or, in some cases, a deletion.
		case "repository", "meta":
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
			res, processingErr = s.processRepositoryEvent(ctx, rawWBPayload)
		case "package":
			// This is an artifact-related event, and can
			// only trigger a reconciliation.
			res, processingErr = s.processPackageEvent(ctx, rawWBPayload)
		case "pull_request":
			res, processingErr = s.processPullRequestEvent(ctx, rawWBPayload)
		// This event is not currently handled.
		case "org_block",
			"organization":
			l.Info().Msgf("webhook events %s do not contain repo", wes.Typ)
		case "ping":
			// For ping events, we do not set wes.Accepted
			// to true because they're not relevant
			// business events.
			wes.Accepted = false
			wes.Error = false
			s.processPingEvent(ctx, rawWBPayload)
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

func (s *Server) processPackageEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
	var event *packageEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	if event.Action == nil {
		return nil, errors.New("invalid event: action is nil")
	}
	if event.Package == nil || event.Repo == nil {
		log.Printf("could not determine relevant entity for event. Skipping execution.")
		return nil, nil // this is awkward
	}

	// We only process events "package" with action "published",
	// i.e. we do not react to action "updated".
	if *event.Action != webhookActionEventPublished {
		return nil, nil
	}

	if event.Package.Owner == nil {
		return nil, errors.New("invalid package: owner is nil")
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

	dbrepo, err := s.fetchRepo(ctx, event.GetRepo())
	if err != nil {
		return nil, err
	}

	// This only makes sense for "meta" event type
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

	// protobufs are our API, so we always execute on these instead of the DB directly.
	pbRepo := repositories.PBRepositoryFromDB(*dbrepo)
	eiw := entities.NewEntityInfoWrapper().
		WithProviderID(dbrepo.ProviderID).
		WithRepository(pbRepo).
		WithProjectID(dbrepo.ProjectID).
		WithRepositoryID(dbrepo.ID).
		WithActionEvent(event.GetAction())

	topic := events.TopicQueueEntityEvaluate
	if event.GetAction() == webhookActionEventDeleted {
		topic = events.TopicQueueReconcileEntityDelete
	}

	return &processingResult{topic: topic, eiw: eiw}, nil
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

	dbrepo, err := s.fetchRepo(ctx, event.GetRepo())
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
		WithActionEvent(event.GetAction())

	return &processingResult{topic: events.TopicQueueEntityEvaluate, eiw: eiw}, nil
}

func (s *Server) processPullRequestEvent(
	ctx context.Context,
	payload []byte,
) (*processingResult, error) {
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
	dbrepo, err := s.fetchRepo(ctx, &repo{
		ID:      ghRepo.ID,
		HTMLURL: ghRepo.HTMLURL,
		Private: ghRepo.Private,
	})
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

	return &processingResult{topic: events.TopicQueueEntityEvaluate, eiw: eiw}, nil
}

// processInstallationAppEvent processes events related to changes to
// the app itself as well as the list of accessible repositories.
//
// There are several possible actions, but in the current user flows
// we only process deletion.
func (_ *Server) processInstallationAppEvent(
	_ context.Context,
	payload []byte,
) (*processingResult, error) {
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
		ghprov.GitHubAppInstallationDeletedPayload{
			InstallationID: event.GetInstallation().GetID(),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload: %w", err)
	}

	iiw := installations.NewInstallationInfoWrapper().
		WithProviderClass(db.ProviderClassGithubApp).
		WithPayload(payloadBytes)

	res := &processingResult{
		topic: installations.ProviderInstallationTopic,
		iiw:   iiw,
	}

	return res, nil
}

func (s *Server) fetchRepo(
	ctx context.Context,
	repo *repo,
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

	artifact := &pb.Artifact{
		Owner:      owner,
		Name:       *event.Package.Name,
		Type:       *event.Package.PackageType,
		Repository: event.Repo.GetFullName(),
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
	// TODO go-github documentation reportes that
	// PullRequestEvents with action "synchronize" are not
	// published, see here
	// https://pkg.go.dev/github.com/google/go-github/v62@v62.0.0/github#PullRequestEvent
	case webhookActionEventOpened, webhookActionEventReopened:
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
