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
	urlparser "net/url"
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

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/providers"
	githubprovider "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/reconcilers"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/verifier"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

type tagIsASignatureError struct {
	message      string
	signatureTag string
}

func (e *tagIsASignatureError) Error() string {
	return e.message
}

func newTagIsASignatureError(msg, signatureTag string) *tagIsASignatureError {
	return &tagIsASignatureError{message: msg, signatureTag: signatureTag}
}

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
		wes := webhookEventState{
			typ:      "unknown",
			accepted: false,
			error:    true,
		}
		defer func() {
			s.mt.webhookEventTypeCount(r.Context(), wes)
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
			fmt.Printf("Error validating webhook payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		wes.typ = github.WebHookType(r)
		if wes.typ == "ping" {
			log.Printf("ping received")
			wes.error = false
			return
		}

		// TODO: extract sender and event time from payload portably
		m := message.NewMessage(uuid.New().String(), nil)
		m.Metadata.Set(events.ProviderDeliveryIdKey, github.DeliveryID(r))
		m.Metadata.Set(events.ProviderTypeKey, string(db.ProviderTypeGithub))
		m.Metadata.Set(events.ProviderSourceKey, "https://api.github.com/") // TODO: handle other sources
		m.Metadata.Set(events.GithubWebhookEventTypeKey, wes.typ)
		// m.Metadata.Set("subject", ghEvent.GetRepo().GetFullName())
		// m.Metadata.Set("time", ghEvent.GetCreatedAt().String())

		log.Printf("publishing of type: %s", m.Metadata["type"])

		if err := s.parseGithubEventForProcessing(rawWBPayload, m); err != nil {
			wes = handleParseError(wes.typ, err)
			if wes.error {
				w.WriteHeader(http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			return
		}

		wes.accepted = true

		if err := s.evt.Publish(events.ExecuteEntityEventTopic, m); err != nil {
			wes.error = true
			log.Printf("Error publishing message: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		wes.error = false
		w.WriteHeader(http.StatusOK)
	}
}

func handleParseError(typ string, parseErr error) webhookEventState {
	state := webhookEventState{typ: typ, accepted: false, error: true}

	var logMsg string
	switch {
	case errors.Is(parseErr, errRepoNotFound):
		state.error = false
		logMsg = "repository not found"
	case errors.Is(parseErr, errArtifactNotFound):
		state.error = false
		logMsg = "artifact not found"
	case errors.Is(parseErr, errRepoIsPrivate):
		state.error = false
		logMsg = "repository is private"
	case errors.Is(parseErr, errNotHandled):
		state.error = false
		logMsg = fmt.Sprintf("webhook event not handled (%v)", parseErr)
	case errors.Is(parseErr, errArtifactVersionSkipped):
		state.error = false
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
func (s *Server) registerWebhookForRepository(
	ctx context.Context,
	pbuild *providers.ProviderBuilder,
	projectID uuid.UUID,
	repo *pb.UpstreamRepositoryRef,
	ghEvents []string,
) (*pb.RegisterRepoResult, error) {

	if !pbuild.Implements(db.ProviderTypeGithub) {
		return nil, fmt.Errorf("provider %s is not supported for github webhook", pbuild.GetName())
	}

	client, err := pbuild.GetGitHub(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating github provider: %w", err)
	}

	url := s.cfg.WebhookConfig.ExternalWebhookURL
	ping := s.cfg.WebhookConfig.ExternalPingURL
	secret := s.cfg.WebhookConfig.WebhookSecret

	regResult := &pb.RegisterRepoResult{
		Repository: &pb.Repository{
			Name:   repo.Name,
			Owner:  repo.Owner,
			RepoId: repo.RepoId,
		},
		Status: &pb.RegisterRepoResult_Status{
			Success: false,
		},
	}

	logger := zerolog.Ctx(ctx).With().
		Str("repoName", repo.Name).
		Str("repoOwner", repo.Owner).Logger()

	// let's verify that the repository actually exists.
	repoGet, err := client.GetRepository(ctx, repo.Owner, repo.Name)
	if err != nil {
		errorStr := err.Error()
		regResult.Status.Error = &errorStr
		return regResult, nil
	}

	// skip if we try to register a private repository
	if repoGet.GetPrivate() && !projectAllowsPrivateRepos(ctx, s.store, projectID) {
		errorStr := "repository is private"
		regResult.Status.Error = &errorStr
		return regResult, nil
	}

	urlUUID := uuid.New().String()

	webhookUrl := fmt.Sprintf("%s/%s", url, urlUUID)
	parsedOriginalURL, err := urlparser.Parse(webhookUrl)
	if err != nil {
		errStr := err.Error()
		regResult.Status.Error = &errStr
		return regResult, nil
	}

	hook := &github.Hook{
		Config: map[string]interface{}{
			"url":          webhookUrl,
			"content_type": "json",
			"ping_url":     ping,
			"secret":       secret,
		},
		Events: ghEvents,
	}

	// if we have an existing hook for same repo, delete it
	hooks, err := client.ListHooks(ctx, repo.Owner, repo.Name)
	if errors.Is(err, githubprovider.ErrNotFound) {
		logger.Debug().Msg("no hooks found")
	} else if err != nil {
		errorStr := err.Error()
		regResult.Status.Error = &errorStr
		logger.Error().Msg("error listing hooks")
		return regResult, nil
	}
	for _, h := range hooks {
		config_url := h.Config["url"].(string)
		if config_url != "" {
			parsedURL, err := urlparser.Parse(config_url)
			if err != nil {
				errorStr := err.Error()
				regResult.Status.Error = &errorStr
				return regResult, nil
			}
			if parsedURL.Host == parsedOriginalURL.Host {
				// it is our hook, we can remove it
				_, err = client.DeleteHook(ctx, repo.Owner, repo.Name, h.GetID())
				if err != nil {
					errorStr := err.Error()
					regResult.Status.Error = &errorStr
					logger.Error().Msg("error deleting hook")
					return regResult, nil
				}
			}
		}
	}

	// Attempt to register webhook
	mhook, err := client.CreateHook(ctx, repo.Owner, repo.Name, hook)
	if err != nil {
		errorStr := err.Error()
		regResult.Status.Error = &errorStr
		logger.Error().Msg("error creating hook")
		return regResult, nil
	}

	regResult.Status.Success = true
	regResult.Repository.HookId = mhook.GetID()
	regResult.Repository.HookUrl = mhook.GetURL()
	regResult.Repository.DeployUrl = webhookUrl
	regResult.Repository.CloneUrl = *repoGet.CloneURL
	regResult.Repository.HookType = mhook.GetType()
	regResult.Repository.HookName = mhook.GetName()
	regResult.Repository.HookUuid = urlUUID
	regResult.Repository.IsPrivate = repoGet.GetPrivate()
	regResult.Repository.IsFork = repoGet.GetFork()
	regResult.Repository.DefaultBranch = repoGet.GetDefaultBranch()

	return regResult, nil
}

func (s *Server) deleteWebhookFromRepository(
	ctx context.Context,
	provider db.Provider,
	projectID uuid.UUID,
	dbrepo db.Repository,
) error {
	pbOpts := []providers.ProviderBuilderOption{
		providers.WithProviderMetrics(s.provMt),
	}
	providerBuilder, err := providers.GetProviderBuilder(ctx, provider, projectID, s.store, s.cryptoEngine, pbOpts...)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot get provider builder: %v", err)
	}

	client, err := providerBuilder.GetGitHub(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot create github client: %v", err)
	}

	webhookId := dbrepo.WebhookID
	if webhookId.Valid {
		resp, err := client.DeleteHook(ctx, dbrepo.RepoOwner, dbrepo.RepoName, int64(webhookId.Int32))
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusNotFound {
				// if the hook is not found, we can ignore the error, user might have deleted it manually
				return nil
			}
			return status.Errorf(codes.Internal, "cannot delete webhook: %v", err)
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

	// get the provider for the repository
	prov, err := s.store.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      dbRepo.Provider,
		ProjectID: dbRepo.ProjectID,
	})
	if err != nil {
		return fmt.Errorf("error getting provider: %w", err)
	}

	pbOpts := []providers.ProviderBuilderOption{
		providers.WithProviderMetrics(s.provMt),
	}
	provBuilder, err := providers.GetProviderBuilder(ctx, prov, dbRepo.ProjectID, s.store, s.cryptoEngine, pbOpts...)
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

	cli, err := prov.GetGitHub(ctx)
	if err != nil {
		log.Printf("error creating github provider: %v", err)
		return err
	}

	tempArtifact, err := gatherArtifact(ctx, cli, s.store, whPayload)
	if err != nil {
		return fmt.Errorf("error gathering versioned artifact: %w", err)
	}

	dbArtifact, _, err := upsertVersionedArtifact(ctx, dbrepo, tempArtifact, s.store)
	if err != nil {
		return fmt.Errorf("error upserting artifact from payload: %w", err)
	}

	pbArtifact, err := util.GetArtifactWithVersions(ctx, s.store, dbrepo.ID, dbArtifact.ID)
	if err != nil {
		return fmt.Errorf("error getting artifact with versions: %w", err)
	}

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

	cli, err := prov.GetGitHub(ctx)
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
	ghArtifact, err := client.GetPackageByName(ctx, isOrg, artifact.Owner, string(verifier.ArtifactTypeContainer), artifact.Name)
	if err != nil {
		return nil, fmt.Errorf("error extracting artifact from repo: %w", err)
	}

	artifact.Visibility = *ghArtifact.Visibility
	return artifact, nil
}

func transformTag(tag string) string {
	// Define the prefix to match and its replacement
	const prefixToMatch = "sha256-"
	const prefixReplacement = "sha256:"

	// If the tag starts with the prefix to match, replace it with the replacement prefix
	if strings.HasPrefix(tag, prefixToMatch) {
		tag = prefixReplacement + tag[len(prefixToMatch):]
	}

	// If the tag has a trailing ".sig", strip it off
	return strings.TrimSuffix(tag, ".sig")
}

// handles the case when we get a notification about an image,
// but a signature arrives a bit later. In that case, we need to:
// -- search for a version whose sha matches the signature tag
// -- if found, update the signature verification field
func lookUpVersionBySignature(
	ctx context.Context,
	store db.Store,
	sigTag string,
) (*pb.ArtifactVersion, error) {
	storedVersion, err := store.GetArtifactVersionBySha(ctx, transformTag(sigTag))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("error looking up version by signature: %w", err)
	}

	return &pb.ArtifactVersion{
		VersionId: int64(storedVersion.Version),
		Tags:      strings.Split(storedVersion.Tags.String, ","),
		Sha:       storedVersion.Sha,
		CreatedAt: timestamppb.New(storedVersion.CreatedAt),
	}, nil
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
	store db.Store,
	payload map[string]any,
) (*pb.Artifact, error) {
	artifact, err := gatherArtifactInfo(ctx, cli, payload)
	if err != nil {
		return nil, fmt.Errorf("error gatherinfo artifact info: %w", err)
	}

	var tagIsSigErr *tagIsASignatureError
	version, err := gatherArtifactVersionInfo(ctx, cli, payload, artifact.Owner, artifact.Name)
	if errors.As(err, &tagIsSigErr) {
		storedVersion, lookupErr := lookUpVersionBySignature(ctx, store, tagIsSigErr.signatureTag)
		if lookupErr != nil {
			return nil, fmt.Errorf("error looking up version by signature tag: %w", lookupErr)
		}
		if storedVersion == nil {
			// return an error that would be caught by the webhook HTTP handler and not retried
			return nil, errArtifactNotFound
		}
		// let's continue with the stored version
		// now get information for signature and workflow
		/*
			just to remember I have to remove this later and handle in
			the new code
		*/
		/*
			err = storeSignatureAndWorkflowInVersion(
				ctx, cli, artifact.Owner, artifact.Name, transformTag(tagIsSigErr.signatureTag), storedVersion)
			if err != nil {
				return nil, fmt.Errorf("error storing signature and workflow in version: %w", err)
			}
		*/

		version = storedVersion
	} else if err != nil {
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
		artifactOwnerLogin, string(verifier.ArtifactTypeContainer), artifactName, version.VersionId)
	if err != nil {
		return fmt.Errorf("error getting package version from repository: %w", err)
	}

	tags := ghVersion.Metadata.Container.Tags
	// if the artifact has no tags, skip it
	if len(tags) == 0 {
		return errArtifactVersionSkipped
	}
	sigTag := verifier.GetSignatureTag(tags)
	if sigTag != "" {
		// handle the case where a signature arrives later than the image
		return newTagIsASignatureError("version is a signature", sigTag)
	}
	sort.Strings(tags)

	version.Tags = tags
	if ghVersion.CreatedAt != nil {
		version.CreatedAt = timestamppb.New(*ghVersion.CreatedAt.GetTime())
	}
	return nil
}

func upsertVersionedArtifact(
	ctx context.Context,
	dbrepo db.Repository,
	artifact *pb.Artifact,
	store db.Store,
) (*db.Artifact, *db.ArtifactVersion, error) {
	// we expect to have only one version at this point, the one from this webhook update
	newArtifactVersion := artifact.Versions[0]

	tx, err := store.BeginTransaction()
	if err != nil {
		return nil, nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := store.GetQuerierWithTransaction(tx)

	dbArtifact, err := qtx.UpsertArtifact(ctx, db.UpsertArtifactParams{
		RepositoryID:       dbrepo.ID,
		ArtifactName:       artifact.GetName(),
		ArtifactType:       artifact.GetType(),
		ArtifactVisibility: artifact.Visibility,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error upserting artifact: %w", err)
	}

	err = qtx.DeleteOldArtifactVersions(ctx,
		db.DeleteOldArtifactVersionsParams{ArtifactID: dbArtifact.ID, CreatedAt: reconcilers.ArtifactTypeContainerRetentionPeriod})
	if err != nil {
		// just log error, we will not remove older for now
		log.Printf("error removing older artifact versions: %v", err)
	}

	// To avoid conflicts, we search for all existing entries that have the incoming tag in their Tags field.
	// If found, the existing artifact is updated by removing the incoming tag from its tags column.
	// Loop through all incoming tags
	for _, incomingTag := range newArtifactVersion.Tags {
		if err = processArtifactVersionConflicts(ctx, qtx, dbArtifact, incomingTag); err != nil {
			return nil, nil, fmt.Errorf("error processing artifact version conflicts: %w", err)
		}
	}

	// Proceed storing the new versioned artifact
	dbVersion, err := qtx.UpsertArtifactVersion(ctx, db.UpsertArtifactVersionParams{
		ArtifactID: dbArtifact.ID,
		Version:    newArtifactVersion.VersionId,
		Tags: sql.NullString{
			String: strings.Join(newArtifactVersion.Tags, ","),
			Valid:  true,
		},
		Sha:       newArtifactVersion.Sha,
		CreatedAt: newArtifactVersion.CreatedAt.AsTime(),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error upserting artifact version: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return &dbArtifact, &dbVersion, nil
}

func processArtifactVersionConflicts(ctx context.Context, qtx db.Querier, dbArtifact db.Artifact, incomingTag string) error {
	// Search artifact versions having the incoming tag (there should be at most 1 or no matches at all)
	existingArtifactVersions, err := qtx.ListArtifactVersionsByArtifactIDAndTag(ctx,
		db.ListArtifactVersionsByArtifactIDAndTagParams{ArtifactID: dbArtifact.ID,
			Tags:  sql.NullString{Valid: true, String: incomingTag},
			Limit: sql.NullInt32{Valid: false, Int32: 0}})
	if errors.Is(err, sql.ErrNoRows) {
		// There are no tagged versions matching the incoming tag, all okay
		return nil
	} else if err != nil {
		// Unexpected failure
		return fmt.Errorf("failed during repository synchronization: %w", err)
	}
	// Loop through all artifact versions that matched the incoming tag
	for _, existing := range existingArtifactVersions {
		if !existing.Tags.Valid {
			continue
		}
		// Rebuild the list of tags removing anything that would conflict with the incoming tag
		newTags := slices.DeleteFunc(strings.Split(existing.Tags.String, ","),
			func(in string) bool { return in == incomingTag })
		// Delete the versioned artifact row from the store if its list of tags becomes empty
		if len(newTags) == 0 {
			err := qtx.DeleteArtifactVersion(ctx, existing.ID)
			if err != nil {
				return fmt.Errorf("error deleting artifact version %d: %w", existing.ID, err)
			}
			// Deletion went okay, let's continue with the next version
			continue
		}
		// Rebuild the list of remaining tags for the existing versioned artifact
		newTagsSQL := sql.NullString{String: strings.Join(newTags, ",")}
		newTagsSQL.Valid = len(newTagsSQL.String) > 0
		// Update the versioned artifact row in the store (we shouldn't change anything else except the tags value)
		_, err := qtx.UpsertArtifactVersion(ctx, db.UpsertArtifactVersionParams{
			ArtifactID:            existing.ArtifactID,
			Version:               existing.Version,
			Tags:                  newTagsSQL,
			Sha:                   existing.Sha,
			CreatedAt:             existing.CreatedAt,
			SignatureVerification: existing.SignatureVerification.RawMessage,
			GithubWorkflow:        existing.GithubWorkflow.RawMessage,
		})
		if err != nil {
			return fmt.Errorf("error upserting artifact %s with version %d: %w", existing.ArtifactID, existing.Version, err)
		}

	}
	// All okay
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
		Number:   int32(prNumber),
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
			PrNumber:     int64(prEvalInfo.Number),
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
			PrNumber:     int64(prEvalInfo.Number),
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
		if isPrivate && !projectAllowsPrivateRepos(ctx, store, dbrepo.ProjectID) {
			return db.Repository{}, errRepoIsPrivate
		}
	}

	log.Printf("handling event for repository %d", id)

	return dbrepo, nil
}

func parseRepoID(repoID any) (int32, error) {
	switch v := repoID.(type) {
	case int32:
		return v, nil
	case float64:
		return int32(v), nil
	case string:
		// convert string to int
		asInt32, err := strconv.ParseInt(v, 10, 16)
		if err != nil {
			return 0, fmt.Errorf("error converting string to int: %w", err)
		}
		return int32(asInt32), nil
	default:
		return 0, fmt.Errorf("unknown type for repoID: %T", v)
	}
}

func projectAllowsPrivateRepos(ctx context.Context, store db.Store, projectID uuid.UUID) bool {
	// we're throwing away the result because we're really not interested in what the feature
	// sets, just that it's enabled
	_, err := store.GetFeatureInProject(ctx, db.GetFeatureInProjectParams{
		ProjectID: projectID,
		Feature:   "private_repositories_enabled",
	})
	if errors.Is(err, sql.ErrNoRows) {
		zerolog.Ctx(ctx).Debug().Msg("private repositories not enabled for project")
		return false
	} else if err != nil {
		log.Printf("error getting features for project %s: %v", projectID, err)
		return false
	}

	zerolog.Ctx(ctx).Debug().
		Str("project_id", projectID.String()).
		Msg("project allows private repositories")
	return true
}
