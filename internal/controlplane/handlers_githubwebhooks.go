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
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-github/v56/github"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/projects/features"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/util"
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

// https://docs.github.com/en/webhooks/webhook-events-and-payloads#about-webhook-events-and-payloads
var repoEvents = []string{
	"branch_protection_configuration",
	"branch_protection_rule",
	"code_scanning_alert",
	"create", // a tag or branch is created
	"member",
	"meta", // webhook itself
	"repository_vulnerability_alert",
	"org_block",
	"organization",
	"public",
	// listening to push makes sure we evaluate on pushes to branches we need to check, but might be too noisy
	// for topic branches
	"push",
	"repository",
	"repository_advisory",
	"repository_import",
	"repository_ruleset",
	"secret_scanning_alert",
	"secret_scanning_alert_location",
	"security_advisory",
	"security_and_analysis",
	"team",
	"team_add",
}

func entityFromWebhookEventTypeKey(m *message.Message) pb.Entity {
	key := m.Metadata.Get(events.GithubWebhookEventTypeKey)
	switch {
	case key == "package":
		return pb.Entity_ENTITY_ARTIFACTS
	case key == "pull_request":
		return pb.Entity_ENTITY_PULL_REQUESTS
	case slices.Contains(repoEvents, key):
		return pb.Entity_ENTITY_REPOSITORIES
	}

	return pb.Entity_ENTITY_UNSPECIFIED
}

// HandleGitHubWebHook handles incoming GitHub webhooks
// See https://docs.github.com/en/developers/webhooks-and-events/webhooks/about-webhooks
// for more information.
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

		rawWBPayload, err := github.ValidatePayload(r, []byte(viper.GetString("webhook-config.webhook_secret")))
		if err != nil {
			log.Printf("Error validating webhook payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		wes.Typ = github.WebHookType(r)
		if wes.Typ == "ping" {
			log.Printf("ping received")
			wes.Error = false
			return
		}

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
		if err := s.parseGithubEventForProcessing(rawWBPayload, m); err != nil {
			wes = handleParseError(wes.Typ, err)
			if wes.Error {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			return
		}

		wes.Accepted = true

		l.Info().Str("message-id", m.UUID).Msg("publishing event for execution")
		if err := s.evt.Publish(events.ExecuteEntityEventTopic, m); err != nil {
			wes.Error = true
			log.Printf("Error publishing message: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		wes.Error = false
		w.WriteHeader(http.StatusOK)
	}
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

// registerWebhookForRepository registers a set repository and sets up the webhook for each of them
// and returns the registration result for each repository.
// If an error occurs, the registration is aborted and the error is returned.
// https://docs.github.com/en/rest/reference/repos#create-a-repository-webhook

// The actual logic for webhook creation lives in the WebhookManager interface
// TODO: the remaining logic should be refactored into a repository
// registration interface
func (s *Server) registerWebhookForRepository(
	ctx context.Context,
	pbuild *providers.ProviderBuilder,
	projectID uuid.UUID,
	repo *pb.UpstreamRepositoryRef,
) (*pb.RegisterRepoResult, error) {
	logger := zerolog.Ctx(ctx).With().
		Str("repoName", repo.Name).
		Str("repoOwner", repo.Owner).
		Logger()
	ctx = logger.WithContext(ctx)

	if !pbuild.Implements(db.ProviderTypeGithub) {
		return nil, fmt.Errorf("provider %s is not supported for github webhook", pbuild.GetName())
	}

	client, err := pbuild.GetGitHub()
	if err != nil {
		return nil, fmt.Errorf("error creating github provider: %w", err)
	}

	regResult := &pb.RegisterRepoResult{
		// We will overwrite this later when we've looked it up from the provider,
		// but existing clients expect a message here, so let's add one.
		Repository: &pb.Repository{
			Name:  repo.Name,  // Not normalized, from client
			Owner: repo.Owner, // Not normalized, from client
		},
		Status: &pb.RegisterRepoResult_Status{
			Success: false,
		},
	}

	// let's verify that the repository actually exists.
	repoGet, err := client.GetRepository(ctx, repo.Owner, repo.Name)
	if err != nil {
		errorStr := err.Error()
		regResult.Status.Error = &errorStr
		return regResult, nil
	}

	// skip if we try to register a private repository
	if repoGet.GetPrivate() && !features.ProjectAllowsPrivateRepos(ctx, s.store, projectID) {
		errorStr := "repository is private"
		regResult.Status.Error = &errorStr
		return regResult, nil
	}

	hookUUID, githubHook, err := s.webhookManager.CreateWebhook(ctx, client, repo.Owner, repo.Name)
	if err != nil {
		logger.Error().Msgf("error while creating webhook: %v", err)
		errorStr := err.Error()
		regResult.Status.Error = &errorStr
		return regResult, nil
	}

	regResult.Status.Success = true

	regResult.Repository = &pb.Repository{
		Name:          repoGet.GetName(),
		Owner:         repoGet.GetOwner().GetLogin(),
		RepoId:        repoGet.GetID(),
		HookId:        githubHook.GetID(),
		HookUrl:       githubHook.GetURL(),
		DeployUrl:     repoGet.GetDeploymentsURL(),
		CloneUrl:      repoGet.GetCloneURL(),
		HookType:      githubHook.GetType(),
		HookName:      githubHook.GetName(),
		HookUuid:      hookUUID,
		IsPrivate:     repoGet.GetPrivate(),
		IsFork:        repoGet.GetFork(),
		DefaultBranch: repoGet.GetDefaultBranch(),
		License:       repoGet.GetLicense().GetSPDXID(),
	}

	return regResult, nil
}

func (s *Server) deleteWebhookFromRepository(
	ctx context.Context,
	provider db.Provider,
	dbrepo db.Repository,
) error {
	pbOpts := []providers.ProviderBuilderOption{
		providers.WithProviderMetrics(s.provMt),
		providers.WithRestClientCache(s.restClientCache),
	}
	providerBuilder, err := providers.GetProviderBuilder(ctx, provider, s.store, s.cryptoEngine, pbOpts...)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get provider builder: %v", err)
	}

	client, err := providerBuilder.GetGitHub()
	if err != nil {
		return status.Errorf(codes.Internal, "cannot create github client: %v", err)
	}

	webhookID := dbrepo.WebhookID
	if webhookID.Valid {
		err = s.webhookManager.DeleteWebhook(
			ctx,
			client,
			dbrepo.RepoOwner,
			dbrepo.RepoName,
			webhookID.Int64,
		)
		if err != nil {
			return status.Errorf(codes.Internal, "cannot delete hook: %v", err)
		}
	}

	return nil
}

func (s *Server) parseGithubEventForProcessing(
	rawWHPayload []byte,
	msg *message.Message,
) error {
	ent := entityFromWebhookEventTypeKey(msg)
	if ent == pb.Entity_ENTITY_UNSPECIFIED {
		return newErrNotHandled("event %s not handled", msg.Metadata.Get(events.GithubWebhookEventTypeKey))
	}

	ctx := context.Background()

	var payload map[string]any
	if err := json.Unmarshal(rawWHPayload, &payload); err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}

	// get information about the repository from the payload
	dbRepo, err := getRepoInformationFromPayload(ctx, s.store, payload)
	if err != nil {
		return fmt.Errorf("error getting repo information from payload: %w", err)
	}

	ph, err := s.store.GetParentProjects(ctx, dbRepo.ProjectID)
	if err != nil {
		return fmt.Errorf("error getting project hierarchy: %w", err)
	}

	// get the provider for the repository
	prov, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:     dbRepo.Provider,
		Projects: ph,
	})
	if err != nil {
		return fmt.Errorf("error getting provider: %w", err)
	}

	pbOpts := []providers.ProviderBuilderOption{
		providers.WithProviderMetrics(s.provMt),
		providers.WithRestClientCache(s.restClientCache),
	}
	provBuilder, err := providers.GetProviderBuilder(ctx, prov, s.store, s.cryptoEngine, pbOpts...)
	if err != nil {
		return fmt.Errorf("error building client: %w", err)
	}

	var action string // explicit declaration to use the default value
	action, err = util.JQReadFrom[string](ctx, ".action", payload)
	if err != nil && !errors.Is(err, util.ErrNoValueFound) {
		return fmt.Errorf("error getting action from payload: %w", err)
	}

	// determine if the payload is an artifact published event
	// TODO: this needs to be managed via signals
	if ent == pb.Entity_ENTITY_ARTIFACTS && action == "published" {
		return s.parseArtifactPublishedEvent(
			ctx, payload, msg, dbRepo, provBuilder)
	} else if ent == pb.Entity_ENTITY_PULL_REQUESTS {
		return parsePullRequestModEvent(
			ctx, payload, msg, dbRepo, s.store, provBuilder)
	} else if ent == pb.Entity_ENTITY_REPOSITORIES {
		return parseRepoEvent(msg, dbRepo, provBuilder.GetName())
	}

	return newErrNotHandled("event %s with action %s not handled",
		msg.Metadata.Get(events.GithubWebhookEventTypeKey), action)
}

func parseRepoEvent(
	msg *message.Message,
	dbrepo db.Repository,
	providerName string,
) error {
	// protobufs are our API, so we always execute on these instead of the DB directly.
	repo := util.PBRepositoryFromDB(dbrepo)
	eiw := entities.NewEntityInfoWrapper().
		WithProvider(providerName).
		WithRepository(repo).
		WithProjectID(dbrepo.ProjectID).
		WithRepositoryID(dbrepo.ID)

	return eiw.ToMessage(msg)
}

func (s *Server) parseArtifactPublishedEvent(
	ctx context.Context,
	whPayload map[string]any,
	msg *message.Message,
	dbrepo db.Repository,
	prov *providers.ProviderBuilder,
) error {
	// we need to have information about package and repository
	if whPayload["package"] == nil || whPayload["repository"] == nil {
		log.Printf("could not determine relevant entity for event. Skipping execution.")
		return nil
	}

	// NOTE(jaosorior): this webhook is very specific to github
	if !prov.Implements(db.ProviderTypeGithub) {
		log.Printf("provider %s is not supported for github webhook", prov.GetName())
		return nil
	}

	cli, err := prov.GetGitHub()
	if err != nil {
		log.Printf("error creating github provider: %v", err)
		return err
	}

	tempArtifact, err := gatherArtifact(ctx, cli, whPayload)
	if err != nil {
		return fmt.Errorf("error gathering versioned artifact: %w", err)
	}

	dbArtifact, err := s.store.UpsertArtifact(ctx, db.UpsertArtifactParams{
		RepositoryID:       dbrepo.ID,
		ArtifactName:       tempArtifact.GetName(),
		ArtifactType:       tempArtifact.GetTypeLower(),
		ArtifactVisibility: tempArtifact.Visibility,
	})
	if err != nil {
		return fmt.Errorf("error upserting artifact: %w", err)
	}

	pbArtifact, err := util.GetArtifact(ctx, s.store, dbrepo.ProjectID, dbrepo.ID, dbArtifact.ID)
	if err != nil {
		return fmt.Errorf("error getting artifact with versions: %w", err)
	}
	// TODO: wrap in a function
	pbArtifact.Versions = tempArtifact.Versions

	eiw := entities.NewEntityInfoWrapper().
		WithArtifact(pbArtifact).
		WithProvider(prov.GetName()).
		WithProjectID(dbrepo.ProjectID).
		WithRepositoryID(dbrepo.ID).
		WithArtifactID(dbArtifact.ID)

	return eiw.ToMessage(msg)
}

func parsePullRequestModEvent(
	ctx context.Context,
	whPayload map[string]any,
	msg *message.Message,
	dbrepo db.Repository,
	store db.Store,
	prov *providers.ProviderBuilder,
) error {
	// NOTE(jaosorior): this webhook is very specific to github
	if !prov.Implements(db.ProviderTypeGithub) {
		log.Printf("provider %s is not supported for github webhook", prov.GetName())
		return nil
	}

	cli, err := prov.GetGitHub()
	if err != nil {
		log.Printf("error creating github provider: %v", err)
		return nil
	}

	prEvalInfo, err := getPullRequestInfoFromPayload(ctx, whPayload)
	if err != nil {
		return fmt.Errorf("error getting pull request information from payload: %w", err)
	}

	dbPr, err := reconcilePrWithDb(ctx, store, dbrepo, prEvalInfo)
	if errors.Is(err, errNotHandled) {
		return err
	} else if err != nil {
		return fmt.Errorf("error reconciling PR with DB: %w", err)
	}

	err = updatePullRequestInfoFromProvider(ctx, cli, dbrepo, prEvalInfo)
	if err != nil {
		return fmt.Errorf("error updating pull request information from provider: %w", err)
	}

	log.Printf("evaluating PR %+v", prEvalInfo)

	eiw := entities.NewEntityInfoWrapper().
		WithPullRequest(prEvalInfo).
		WithPullRequestID(dbPr.ID).
		WithProvider(prov.GetName()).
		WithProjectID(dbrepo.ProjectID).
		WithRepositoryID(dbrepo.ID)

	return eiw.ToMessage(msg)
}

func extractArtifactFromPayload(ctx context.Context, payload map[string]any) (*pb.Artifact, error) {
	artifactName, err := util.JQReadFrom[string](ctx, ".package.name", payload)
	if err != nil {
		return nil, err
	}
	artifactType, err := util.JQReadFrom[string](ctx, ".package.package_type", payload)
	if err != nil {
		return nil, err
	}
	ownerLogin, err := util.JQReadFrom[string](ctx, ".package.owner.login", payload)
	if err != nil {
		return nil, err
	}
	repoName, err := util.JQReadFrom[string](ctx, ".repository.full_name", payload)
	if err != nil {
		return nil, err
	}

	artifact := &pb.Artifact{
		Owner:      ownerLogin,
		Name:       artifactName,
		Type:       artifactType,
		Repository: repoName,
		// visibility and createdAt are not in the payload, we need to get it with a REST call
	}

	return artifact, nil
}

func extractArtifactVersionFromPayload(ctx context.Context, payload map[string]any) (*pb.ArtifactVersion, error) {
	packageVersionId, err := util.JQReadFrom[float64](ctx, ".package.package_version.id", payload)
	if err != nil {
		return nil, err
	}
	packageVersionSha, err := util.JQReadFrom[string](ctx, ".package.package_version.version", payload)
	if err != nil {
		return nil, err
	}
	tag, err := util.JQReadFrom[string](ctx, ".package.package_version.container_metadata.tag.name", payload)
	if err != nil {
		return nil, err
	}

	version := &pb.ArtifactVersion{
		VersionId: int64(packageVersionId),
		Tags:      []string{tag},
		Sha:       packageVersionSha,
	}

	return version, nil
}

func gatherArtifactInfo(
	ctx context.Context,
	client provifv1.GitHub,
	payload map[string]any,
) (*pb.Artifact, error) {
	artifact, err := extractArtifactFromPayload(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("error extracting artifact from payload: %w", err)
	}

	// we also need to fill in the visibility which is not in the payload
	isOrg := client.GetOwner() != ""
	ghArtifact, err := client.GetPackageByName(ctx, isOrg, artifact.Owner, string(verifyif.ArtifactTypeContainer), artifact.Name)
	if err != nil {
		return nil, fmt.Errorf("error extracting artifact from repo: %w", err)
	}

	artifact.Visibility = *ghArtifact.Visibility
	return artifact, nil
}

func gatherArtifactVersionInfo(
	ctx context.Context,
	cli provifv1.GitHub,
	payload map[string]any,
	artifactOwnerLogin, artifactName string,
) (*pb.ArtifactVersion, error) {
	version, err := extractArtifactVersionFromPayload(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("error extracting artifact version from payload: %w", err)
	}

	// not all information is in the payload, we need to get it from the container registry
	// and/or GH API
	err = updateArtifactVersionFromRegistry(ctx, cli, artifactOwnerLogin, artifactName, version)
	if err != nil {
		return nil, fmt.Errorf("error getting upstream information for artifact version: %w", err)
	}

	return version, nil
}

func gatherArtifact(
	ctx context.Context,
	cli provifv1.GitHub,
	payload map[string]any,
) (*pb.Artifact, error) {
	artifact, err := gatherArtifactInfo(ctx, cli, payload)
	if err != nil {
		return nil, fmt.Errorf("error gatherinfo artifact info: %w", err)
	}

	version, err := gatherArtifactVersionInfo(ctx, cli, payload, artifact.Owner, artifact.Name)
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
	isOrg := client.GetOwner() != ""
	ghVersion, err := client.GetPackageVersionById(ctx, isOrg,
		artifactOwnerLogin, string(verifyif.ArtifactTypeContainer), artifactName, version.VersionId)
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

func getPullRequestInfoFromPayload(
	ctx context.Context,
	payload map[string]any,
) (*pb.PullRequest, error) {
	prUrl, err := util.JQReadFrom[string](ctx, ".pull_request.url", payload)
	if err != nil {
		return nil, fmt.Errorf("error getting pull request url from payload: %w", err)
	}

	prNumber, err := util.JQReadFrom[float64](ctx, ".pull_request.number", payload)
	if err != nil {
		return nil, fmt.Errorf("error getting pull request number from payload: %w", err)
	}

	prAuthorId, err := util.JQReadFrom[float64](ctx, ".pull_request.user.id", payload)
	if err != nil {
		return nil, fmt.Errorf("error getting pull request author ID from payload: %w", err)
	}

	action, err := util.JQReadFrom[string](ctx, ".action", payload)
	if err != nil {
		return nil, fmt.Errorf("error getting action from payload: %w", err)
	}

	return &pb.PullRequest{
		Url:      prUrl,
		Number:   int64(prNumber),
		AuthorId: int64(prAuthorId),
		Action:   action,
	}, nil
}

func reconcilePrWithDb(
	ctx context.Context,
	store db.Store,
	dbrepo db.Repository,
	prEvalInfo *pb.PullRequest,
) (*db.PullRequest, error) {
	var retErr error
	var retPr *db.PullRequest

	switch prEvalInfo.Action {
	case "opened", "synchronize":
		dbPr, err := store.UpsertPullRequest(ctx, db.UpsertPullRequestParams{
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
	case "closed":
		err := store.DeletePullRequest(ctx, db.DeletePullRequestParams{
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

func getRepoInformationFromPayload(
	ctx context.Context,
	store db.Store,
	payload map[string]any,
) (db.Repository, error) {
	repoInfo, ok := payload["repository"].(map[string]any)
	if !ok {
		return db.Repository{}, fmt.Errorf("unable to determine repository for event: %w", errRepoNotFound)
	}

	id, err := parseRepoID(repoInfo["id"])
	if err != nil {
		return db.Repository{}, fmt.Errorf("error parsing repository ID: %w", err)
	}

	// At this point, we're unsure what the project ID is, so we need to look it up.
	// It's the same case for the provider. We can gather this information from the
	// repository ID.
	dbrepo, err := store.GetRepositoryByRepoID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("repository %d not found", id)
			// no use in continuing if the repository doesn't exist
			return db.Repository{}, fmt.Errorf("repository %d not found: %w", id, errRepoNotFound)
		}
		return db.Repository{}, fmt.Errorf("error getting repository: %w", err)
	}

	if dbrepo.ProjectID.String() == "" {
		return db.Repository{}, fmt.Errorf("no project found for repository %s/%s: %w",
			dbrepo.RepoOwner, dbrepo.RepoName, errRepoNotFound)
	}

	// ignore processing webhooks for private repositories
	isPrivate, ok := repoInfo["private"].(bool)
	if ok {
		if isPrivate && !features.ProjectAllowsPrivateRepos(ctx, store, dbrepo.ProjectID) {
			return db.Repository{}, errRepoIsPrivate
		}
	}

	log.Printf("handling event for repository %d", id)

	return dbrepo, nil
}

func parseRepoID(repoID any) (int64, error) {
	switch v := repoID.(type) {
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case string:
		// convert string to int
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("unknown type for repoID: %T", v)
	}
}
