package github

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/go-github/v61/github"
	"github.com/rs/zerolog"
	"github.com/stacklok/minder/internal/config/server"
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
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"log"
	"mime"
	"net/http"
	"slices"
	"sort"
	"strings"
)

type tokenHandler struct {
}

// HandleGitHubWebHook handles incoming GitHub webhooks
// See https://docs.github.com/en/developers/webhooks-and-events/webhooks/about-webhooks
// for more information.
func (s *tokenHandler) HandleGitHubWebHook() http.HandlerFunc {
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
		if wes.Typ == "ping" {
			logPingReceivedEvent(r.Context(), rawWBPayload)
			wes.Error = false
			return
		}

		// TODO: extract sender and event time from payload portably
		m := message.NewMessage(uid.New().String(), nil)
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

		// Parse the webhook event and construct a message for the event router
		if err := parseGithubEventForProcessing(rawWBPayload, m); err != nil {
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

		// Channel the event based on the webhook action
		var watermillTopic string
		if shouldIssueDeletionEvent(m) {
			watermillTopic = events.TopicQueueReconcileEntityDelete
		} else {
			watermillTopic = events.TopicQueueEntityEvaluate
		}

		// Publish the message to the event router
		if err := s.evt.Publish(watermillTopic, m); err != nil {
			wes.Error = true
			log.Printf("Error publishing message: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// We successfully published the message
		wes.Error = false
		w.WriteHeader(http.StatusOK)
	}
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

	payload, err = github.ValidatePayloadFromBody(contentType, br, signature, []byte(wc.WebhookSecret))
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

	return
}

func parseGithubEventForProcessing(
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
	prov, err := s.providerStore.GetByName(ctx, dbRepo.ProjectID, dbRepo.Provider)
	if err != nil {
		return fmt.Errorf("error getting provider: %w", err)
	}

	pbOpts := []providers.ProviderBuilderOption{
		providers.WithProviderMetrics(s.provMt),
		providers.WithRestClientCache(s.restClientCache),
	}
	provBuilder, err := providers.GetProviderBuilder(ctx, *prov, s.store, s.cryptoEngine, &s.cfg.Provider,
		s.fallbackTokenClient, pbOpts...)
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
	if ent == pb.Entity_ENTITY_ARTIFACTS && action == WebhookActionEventPublished {
		return parseArtifactPublishedEvent(ctx, payload, msg, dbRepo, provBuilder, action)
	} else if ent == pb.Entity_ENTITY_PULL_REQUESTS {
		return parsePullRequestModEvent(ctx, payload, msg, dbRepo, s.store, provBuilder, action)
	} else if ent == pb.Entity_ENTITY_REPOSITORIES {
		return parseRepoEvent(payload, msg, dbRepo, provBuilder.GetName(), action)
	}

	return newErrNotHandled("event %s with action %s not handled",
		msg.Metadata.Get(events.GithubWebhookEventTypeKey), action)
}

func parseRepoEvent(
	whPayload map[string]any,
	msg *message.Message,
	dbrepo db.Repository,
	providerName string,
	action string,
) error {
	if action == WebhookActionEventDeleted {
		// Find out what kind of repository event we are dealing with
		if whPayload["hook"] != nil || whPayload["hook_id"] != nil {
			// Having these means it's a repository event related to the webhook itself, i.e., deleted, created, etc.
			// Get the webhook ID from the payload
			whID := whPayload["hook_id"].(float64)
			if dbrepo.WebhookID.Valid {
				// Check if the payload webhook ID matches the one we have stored in the DB for this repository
				if int64(whID) != dbrepo.WebhookID.Int64 {
					// This means we got a deleted event for a webhook ID that doesn't correspond to the one we have stored in the DB.
					return newErrNotHandled("delete event %s with action %s not handled, hook ID %d does not match stored webhook ID %d",
						msg.Metadata.Get(events.GithubWebhookEventTypeKey), action, int64(whID), dbrepo.WebhookID.Int64)
				}
			}
			// This means we got a deleted event for a webhook ID that corresponds to the one we have stored in the DB.
			// We will remove the repo from the DB, so we can proceed with the deletion event for this entity (repository)
			// TODO: perhaps handle this better by trying to re-create the webhook if it was deleted manually
		}
		// If we don't have hook/hook_id, continue with the deletion event for this entity (repository)
	}

	// protobufs are our API, so we always execute on these instead of the DB directly.
	repo := util.PBRepositoryFromDB(dbrepo)
	eiw := entities.NewEntityInfoWrapper().
		WithProvider(providerName).
		WithRepository(repo).
		WithProjectID(dbrepo.ProjectID).
		WithRepositoryID(dbrepo.ID).
		WithActionEvent(action)

	return eiw.ToMessage(msg)
}

func parseArtifactPublishedEvent(
	ctx context.Context,
	whPayload map[string]any,
	msg *message.Message,
	dbrepo db.Repository,
	prov *providers.ProviderBuilder,
	action string,
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
		return fmt.Errorf("error upserting artifact: %w", err)
	}

	pbArtifact, err := util.GetArtifact(ctx, s.store, dbrepo.ProjectID, dbArtifact.ID)
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
		WithArtifactID(dbArtifact.ID).
		WithActionEvent(action)

	return eiw.ToMessage(msg)
}

func parsePullRequestModEvent(
	ctx context.Context,
	whPayload map[string]any,
	msg *message.Message,
	dbrepo db.Repository,
	store db.Store,
	prov *providers.ProviderBuilder,
	action string,
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
		WithRepositoryID(dbrepo.ID).
		WithActionEvent(action)

	return eiw.ToMessage(msg)
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
	case WebhookActionEventOpened, WebhookActionEventSynchronize:
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
	case WebhookActionEventClosed:
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

func shouldIssueDeletionEvent(m *message.Message) bool {
	return m.Metadata.Get(entities.ActionEventKey) == WebhookActionEventDeleted &&
		deletionOfRelevantType(m)
}

func deletionOfRelevantType(m *message.Message) bool {
	return m.Metadata.Get(events.GithubWebhookEventTypeKey) == "repository" ||
		m.Metadata.Get(events.GithubWebhookEventTypeKey) == "meta"
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
	ghArtifact, err := client.GetPackageByName(ctx, artifact.Owner, string(verifyif.ArtifactTypeContainer), artifact.Name)
	if err != nil {
		return nil, fmt.Errorf("error extracting artifact from repo: %w", err)
	}

	artifact.Visibility = *ghArtifact.Visibility
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
