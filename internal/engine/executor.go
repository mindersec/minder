// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package engine

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	evalerrors "github.com/stacklok/mediator/internal/engine/errors"
	"github.com/stacklok/mediator/internal/events"
	"github.com/stacklok/mediator/internal/util"
	"github.com/stacklok/mediator/pkg/container"
	"github.com/stacklok/mediator/pkg/crypto"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

const (
	// InternalWebhookEventTopic is the topic for internal webhook events
	InternalWebhookEventTopic = "internal.webhook.event"
	// InternalInitEventTopic is the topic for internal init events
	InternalInitEventTopic = "internal.init.event"
	// InternalReconcilerEventTopic is the topic for internal reconciler events
	InternalReconcilerEventTopic = "internal.reconciler.event"
)

// Executor is the engine that executes the rules for a given event
type Executor struct {
	querier db.Store
}

// NewExecutor creates a new executor
func NewExecutor(querier db.Store) *Executor {
	return &Executor{
		querier: querier,
	}
}

// Register implements the Consumer interface.
func (e *Executor) Register(r events.Registrar) {
	r.Register(InternalWebhookEventTopic, e.handleWebhookEvent)
	r.Register(InternalInitEventTopic, e.handleInitEvent)
	r.Register(InternalReconcilerEventTopic, e.handleReconcilerEvent)
}

// InitEvent is an event that is sent to the init topic
// Note that this event assumes the `provider` is set in the metadata
type InitEvent struct {
	// Group is the group that the event is relevant to
	Group int32 `json:"group" validate:"gte=0"`
	// Policy is the policy that the event is relevant to
	Policy int32 `json:"policy" validate:"gte=0"`
}

// ReconcilerEvent is an event that is sent to the reconciler topic
type ReconcilerEvent struct {
	// Group is the group that the event is relevant to
	Group int32 `json:"group" validate:"gte=0"`
	// Repository is the repository to be reconciled
	Repository int32 `json:"repository" validate:"gte=0"`
}

// handleReconcilerEvent handles events coming from the reconciler topic
func (e *Executor) handleReconcilerEvent(msg *message.Message) error {
	prov := msg.Metadata.Get("provider")

	if prov != ghclient.Github {
		log.Printf("provider %s not supported", prov)
		return nil
	}

	var evt ReconcilerEvent
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}

	// validate event
	validate := validator.New()
	if err := validate.Struct(evt); err != nil {
		// We don't return the event since there's no use
		// retrying it if it's invalid.
		log.Printf("error validating event: %v", err)
		return nil
	}

	ctx := msg.Context()
	log.Printf("handling reconciler event for group %d and repository %d", evt.Group, evt.Repository)
	return e.HandleArtifactsReconcilerEvent(ctx, prov, &evt)
}

// handleInitEvent handles events coming from the init topic
// This allows us to run the engine on policy creation and updates
// without having to wait for an event to come from the provider/signal.
func (e *Executor) handleInitEvent(msg *message.Message) error {
	prov := msg.Metadata.Get("provider")

	if prov != ghclient.Github {
		log.Printf("provider %s not supported", prov)
		return nil
	}

	var evt InitEvent
	if err := json.Unmarshal(msg.Payload, &evt); err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}

	// validate event
	validate := validator.New()
	if err := validate.Struct(evt); err != nil {
		// We don't return the event since there's no use
		// retrying it if it's invalid.
		log.Printf("error validating event: %v", err)
		return nil
	}

	ctx := msg.Context()

	log.Printf("handling init event for group %d", evt.Group)

	// TODO(jaosorior): Handle events that are not repository events
	// TODO(jaosorior): get provider from database
	return e.handleReposInitEvent(ctx, prov, &evt)
}

// handleReposInitEvent handles events coming from the init topic
func (e *Executor) handleReposInitEvent(ctx context.Context, prov string, evt *InitEvent) error {
	// Get repositories for group
	dbrepos, err := e.querier.ListRegisteredRepositoriesByGroupIDAndProvider(ctx,
		db.ListRegisteredRepositoriesByGroupIDAndProviderParams{
			Provider: prov,
			GroupID:  evt.Group,
		})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("no repositories found for group %d", evt.Group)
			return nil
		}
		return fmt.Errorf("error getting repositories: %w", err)
	}

	// Get group info
	group, err := e.querier.GetGroupByID(ctx, evt.Group)
	if err != nil {
		return fmt.Errorf("error getting group: %w", err)
	}

	// Get policy info
	dbpols, err := e.querier.GetPolicyByGroupAndID(ctx, db.GetPolicyByGroupAndIDParams{
		GroupID: evt.Group,
		ID:      evt.Policy,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("policy %d not found", evt.Policy)
			return nil
		}
		return fmt.Errorf("error getting policy: %w", err)
	}

	cli, err := e.buildClient(ctx, prov, evt.Group)
	if err != nil {
		return fmt.Errorf("error building client: %w", err)
	}

	ectx := &EntityContext{
		Group: Group{
			ID:   group.ID,
			Name: group.Name,
		},
		Provider: prov,
	}

	for _, pol := range MergeDatabaseGetIntoPolicies(dbpols, ectx) {
		// Given we're dealing with a repository event, we can assume that the
		// entity is a repository.
		relevant, err := GetRulesForEntity(pol, pb.Entity_ENTITY_REPOSITORIES)
		if err != nil {
			return fmt.Errorf("error getting rules for entity: %w", err)
		}

		for _, dbrepo := range dbrepos {
			// protobufs are our API, so we always execute on these instead of the DB directly.
			repo := &pb.RepositoryResult{
				Owner:      dbrepo.RepoOwner,
				Repository: dbrepo.RepoName,
				RepoId:     dbrepo.RepoID,
				HookUrl:    dbrepo.WebhookUrl,
				DeployUrl:  dbrepo.DeployUrl,
				CreatedAt:  timestamppb.New(dbrepo.CreatedAt),
				UpdatedAt:  timestamppb.New(dbrepo.UpdatedAt),
			}

			// Let's evaluate all the repo rules for this policy
			err = TraverseRules(relevant, func(rule *pb.PipelinePolicy_Rule) error {
				rt, rte, err := e.getEvaluator(ctx, *pol.Id, prov, cli, "", ectx, rule)
				if err != nil {
					return err
				}

				return e.createOrUpdateEvalStatus(ctx, &createOrUpdateEvalStatusParams{
					policyID:       *pol.Id,
					repoID:         dbrepo.ID,
					ruleTypeEntity: db.Entities(rt.Def.GetInEntity()),
					ruleTypeID:     *rt.Id,
					evalErr:        rte.Eval(ctx, repo, rule.Def.AsMap(), rule.Params.AsMap()),
				})
			})
			if err != nil {
				return fmt.Errorf("error traversing rules for policy %d: %w", pol.Id, err)
			}
		}
	}

	return nil
}

// handleWebhookEvent handles events coming from webhooks/signals
func (e *Executor) handleWebhookEvent(msg *message.Message) error {
	prov := msg.Metadata.Get("provider")

	if prov != ghclient.Github {
		log.Printf("provider %s not supported", prov)
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}
	ctx := msg.Context()

	// determine if the payload is an artifact published event
	// TODO: this needs to be managed via signals
	hook_type := msg.Metadata.Get("type")
	if hook_type == "package" {
		if payload["action"] == "published" {
			return e.handleArtifactPublishedEvent(ctx, ghclient.Github, payload)
		}
	} else {
		// determine if the payload is a repository event
		_, isRepo := payload["repository"]

		// TODO(jaosorior): Handle events that are not repository events
		if !isRepo {
			log.Printf("could not determine relevant entity for event. Skipping execution.")
			return nil
		}

		// TODO(jaosorior): Handle events that are not repository events
		// TODO(jaosorior): get provider from database
		return e.handleRepoEvent(ctx, ghclient.Github, payload)
	}
	return nil
}

func extractArtifactFromPayload(ctx context.Context, payload map[string]any) (*pb.Artifact, error) {
	artifactId, err := util.JQGetValuesFromAccessor(ctx, ".package.id", payload)
	if err != nil {
		return nil, err
	}
	artifactName, err := util.JQGetValuesFromAccessor(ctx, ".package.name", payload)
	if err != nil {
		return nil, err
	}
	artifactType, err := util.JQGetValuesFromAccessor(ctx, ".package.package_type", payload)
	if err != nil {
		return nil, err
	}
	ownerLogin, err := util.JQGetValuesFromAccessor(ctx, ".package.owner.login", payload)
	if err != nil {
		return nil, err
	}
	repoName, err := util.JQGetValuesFromAccessor(ctx, ".repository.full_name", payload)
	if err != nil {
		return nil, err
	}
	packageUrl, err := util.JQGetValuesFromAccessor(ctx, ".package.package_version.package_url", payload)
	if err != nil {
		return nil, err
	}

	artifact := &pb.Artifact{
		ArtifactId: int64(artifactId.(float64)),
		Owner:      ownerLogin.(string),
		Name:       artifactName.(string),
		Type:       artifactType.(string),
		Repository: repoName.(string),
		PackageUrl: packageUrl.(string),
		// visibility and createdAt are not in the payload, we need to get it with a REST call
	}

	return artifact, nil
}

func extractArtifactVersionFromPayload(ctx context.Context, payload map[string]any) (*pb.ArtifactVersion, error) {
	packageVersionId, err := util.JQGetValuesFromAccessor(ctx, ".package.package_version.id", payload)
	if err != nil {
		return nil, err
	}
	packageVersionSha, err := util.JQGetValuesFromAccessor(ctx, ".package.package_version.version", payload)
	if err != nil {
		return nil, err
	}
	tag, err := util.JQGetValuesFromAccessor(ctx, ".package.package_version.container_metadata.tag.name", payload)
	if err != nil {
		return nil, err
	}

	version := &pb.ArtifactVersion{
		VersionId:             int64(packageVersionId.(float64)),
		Tags:                  []string{tag.(string)},
		Sha:                   packageVersionSha.(string),
		SignatureVerification: nil, // will be filled later by a call to the container registry
		GithubWorkflow:        nil, // will be filled later by a call to the container registry
	}

	return version, nil
}

func updateArtifactVersionFromRegistry(
	ctx context.Context,
	client ghclient.RestAPI,
	payload map[string]any,
	artifactOwnerLogin, artifactName string,
	version *pb.ArtifactVersion,
) error {
	packageVersionName, err := util.JQGetValuesFromAccessor(ctx, ".package.package_version.name", payload)
	if err != nil {
		return fmt.Errorf("error getting package version name: %w", err)
	}

	// we'll grab the artifact version from the REST endpoint because we need the visibility
	// and createdAt fields which are not in the payload
	isOrg := client.GetOwner() != ""
	ghVersion, err := client.GetPackageVersionById(ctx, isOrg, artifactOwnerLogin, CONTAINER_TYPE, artifactName, version.VersionId)
	if err != nil {
		return fmt.Errorf("error getting package version from repository: %w", err)
	}

	tags := ghVersion.Metadata.Container.Tags
	if isSignature(tags) {
		// we don't care about signatures
		return nil
	}
	sort.Strings(tags)

	// now get information for signature and workflow
	packageVersionNameStr, ok := packageVersionName.(string)
	if !ok {
		return fmt.Errorf("package version name is not a string")
	}

	sigInfo, workflowInfo, err := container.GetArtifactSignatureAndWorkflowInfo(
		ctx, client, artifactOwnerLogin, artifactName, packageVersionNameStr)
	if errors.Is(err, container.ErrSigValidation) || errors.Is(err, container.ErrProtoParse) {
		return err
	} else if err != nil {
		return err
	}

	ghWorkflow := &pb.GithubWorkflow{}
	if err := protojson.Unmarshal(workflowInfo, ghWorkflow); err != nil {
		return err
	}

	sigVerification := &pb.SignatureVerification{}
	if err := protojson.Unmarshal(sigInfo, sigVerification); err != nil {
		return err
	}

	version.SignatureVerification = sigVerification
	version.GithubWorkflow = ghWorkflow
	version.Tags = tags
	if ghVersion.CreatedAt != nil {
		version.CreatedAt = timestamppb.New(*ghVersion.CreatedAt.GetTime())
	}
	return nil
}

func gatherArtifactInfo(
	ctx context.Context,
	client ghclient.RestAPI,
	payload map[string]any,
) (*pb.Artifact, error) {
	artifact, err := extractArtifactFromPayload(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("error extracting artifact from payload: %w", err)
	}

	// we also need to fill in the visibility which is not in the payload
	isOrg := client.GetOwner() != ""
	ghArtifact, err := client.GetPackageByName(ctx, isOrg, artifact.Owner, CONTAINER_TYPE, artifact.Name)
	if err != nil {
		return nil, fmt.Errorf("error extracting artifact from repo: %w", err)
	}

	artifact.Visibility = *ghArtifact.Visibility
	return artifact, nil
}

func gatherArtifactVersionInfo(
	ctx context.Context,
	cli ghclient.RestAPI,
	payload map[string]any,
	artifactOwnerLogin, artifactName string,
) (*pb.ArtifactVersion, error) {
	version, err := extractArtifactVersionFromPayload(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("error extracting artifact version from payload: %w", err)
	}

	// not all information is in the payload, we need to get it from the container registry
	// and/or GH API
	err = updateArtifactVersionFromRegistry(ctx, cli, payload, artifactOwnerLogin, artifactName, version)
	if err != nil {
		return nil, fmt.Errorf("error getting upstream information for artifact version: %w", err)
	}

	return version, nil
}

func gatherVersionedArtifact(
	ctx context.Context,
	cli ghclient.RestAPI,
	payload map[string]any,
) (*pb.VersionedArtifact, error) {
	artifact, err := gatherArtifactInfo(ctx, cli, payload)
	if err != nil {
		return nil, fmt.Errorf("error gatherinfo artifact info: %w", err)
	}

	version, err := gatherArtifactVersionInfo(ctx, cli, payload, artifact.Owner, artifact.Name)
	if err != nil {
		return nil, fmt.Errorf("error extracting artifact from payload: %w", err)
	}

	if version == nil {
		// no point in storing and evaluating just the .sig
		return nil, nil
	}

	return &pb.VersionedArtifact{
		Artifact: artifact,
		Version:  version,
	}, nil
}

func (e *Executor) handleArtifactPublishedEvent(ctx context.Context, prov string, payload map[string]any) error {
	// we need to have information about package and repository
	if payload["package"] == nil || payload["repository"] == nil {
		log.Printf("could not determine relevant entity for event. Skipping execution.")
		return nil
	}

	// extract information about repository so we can identity the group and associated rules
	dbrepo, err := e.getRepoInformationFromPayload(ctx, prov, payload)
	if err != nil {
		return fmt.Errorf("error getting repo information from payload: %w", err)
	}
	g := dbrepo.GroupID

	// get group info
	group, err := e.querier.GetGroupByID(ctx, g)
	if err != nil {
		return fmt.Errorf("error getting group: %w", err)
	}

	cli, err := e.buildClient(ctx, prov, g)
	if err != nil {
		return fmt.Errorf("error building client: %w", err)
	}

	ectx := &EntityContext{
		Group: Group{
			ID:   group.ID,
			Name: group.Name,
		},
		Provider: prov,
	}

	versionedArtifact, err := gatherVersionedArtifact(ctx, cli, payload)
	if err != nil {
		return fmt.Errorf("error gathering versioned artifact: %w", err)
	}

	if versionedArtifact == nil {
		// no error, but the version was just the signature
		return nil
	}

	dbArtifact, _, err := e.upsertVersionedArtifact(ctx, dbrepo.ID, versionedArtifact)
	if err != nil {
		return fmt.Errorf("error upserting artifact from payload: %w", err)
	}

	dbpols, err := e.querier.ListPoliciesByGroupID(ctx, g)
	if err != nil {
		return fmt.Errorf("error getting policies: %w", err)
	}

	for _, pol := range MergeDatabaseListIntoPolicies(dbpols, ectx) {
		relevant, err := GetRulesForEntity(pol, pb.Entity_ENTITY_ARTIFACTS)
		if err != nil {
			return fmt.Errorf("error getting rules for entity: %w", err)
		}

		// Let's evaluate all the rules for this policy
		err = TraverseRules(relevant, func(rule *pb.PipelinePolicy_Rule) error {
			rt, rte, err := e.getEvaluator(ctx, *pol.Id, prov, cli, cli.GetToken(), ectx, rule)
			if err != nil {
				return err
			}

			result := rte.Eval(ctx, versionedArtifact, rule.Def.AsMap(), rule.Params.AsMap())
			zerolog.Ctx(ctx).Debug().
				Str("policy", pol.Name).
				Str("ruleType", rule.Type).
				Int("artifactId", int(dbArtifact.ID)).
				Err(result)

			return e.createOrUpdateEvalStatus(ctx, &createOrUpdateEvalStatusParams{
				policyID:       *pol.Id,
				repoID:         dbrepo.ID,
				artifactID:     dbArtifact.ID,
				ruleTypeEntity: db.Entities(rt.Def.GetInEntity()),
				ruleTypeID:     *rt.Id,
				evalErr:        result,
			})
		})
		if err != nil {
			return fmt.Errorf("error traversing rules for policy %d: %w", pol.Id, err)
		}

	}

	return nil
}

func (e *Executor) getRepoInformationFromPayload(ctx context.Context, prov string,
	payload map[string]any) (db.Repository, error) {
	repoInfo, ok := payload["repository"].(map[string]any)
	if !ok {
		// If the event doesn't have a relevant repository we can't do anything with it.
		log.Printf("unable to determine repository for event. Skipping execution.")
		parsedPayload, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			log.Printf("payload: %+v", payload)
		} else {
			log.Printf("payload: %s", parsedPayload)
		}
		return db.Repository{}, nil
	}

	id, err := parseRepoID(repoInfo["id"])
	if err != nil {
		log.Printf("error parsing repository ID: %v", err)
		parsedPayload, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			log.Printf("payload: %+v", payload)
		} else {
			log.Printf("payload: %s", parsedPayload)
		}
		return db.Repository{}, fmt.Errorf("error parsing repository ID: %w", err)
	}

	log.Printf("handling event for repository %d", id)

	dbrepo, err := e.querier.GetRepositoryByRepoID(ctx, db.GetRepositoryByRepoIDParams{
		Provider: prov,
		RepoID:   id,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("repository %d not found", id)
			// no use in continuing if the repository doesn't exist
			return db.Repository{}, nil
		}
		return db.Repository{}, fmt.Errorf("error getting repository: %w", err)
	}
	return dbrepo, nil
}

func (e *Executor) handleRepoEvent(ctx context.Context, prov string, payload map[string]any) error {
	dbrepo, err := e.getRepoInformationFromPayload(ctx, prov, payload)
	if err != nil {
		return err
	}
	// protobufs are our API, so we always execute on these instead of the DB directly.
	repo := &pb.RepositoryResult{
		Owner:      dbrepo.RepoOwner,
		Repository: dbrepo.RepoName,
		RepoId:     dbrepo.RepoID,
		HookUrl:    dbrepo.WebhookUrl,
		DeployUrl:  dbrepo.DeployUrl,
		CreatedAt:  timestamppb.New(dbrepo.CreatedAt),
		UpdatedAt:  timestamppb.New(dbrepo.UpdatedAt),
	}

	// TODO(jaosorior): This will need to take the hierarchy into account.
	g := dbrepo.GroupID

	// get group info
	group, err := e.querier.GetGroupByID(ctx, g)
	if err != nil {
		return fmt.Errorf("error getting group: %w", err)
	}

	cli, err := e.buildClient(ctx, prov, g)
	if err != nil {
		return fmt.Errorf("error building client: %w", err)
	}

	ectx := &EntityContext{
		Group: Group{
			ID:   group.ID,
			Name: group.Name,
		},
		Provider: prov,
	}

	// Get policies relevant to group
	dbpols, err := e.querier.ListPoliciesByGroupID(ctx, g)
	if err != nil {
		return fmt.Errorf("error getting policies: %w", err)
	}

	for _, pol := range MergeDatabaseListIntoPolicies(dbpols, ectx) {
		// Given we're dealing with a repository event, we can assume that the
		// entity is a repository.
		relevant, err := GetRulesForEntity(pol, pb.Entity_ENTITY_REPOSITORIES)
		if err != nil {
			return fmt.Errorf("error getting rules for entity: %w", err)
		}

		// Let's evaluate all the rules for this policy
		err = TraverseRules(relevant, func(rule *pb.PipelinePolicy_Rule) error {
			rt, rte, err := e.getEvaluator(ctx, *pol.Id, prov, cli, "", ectx, rule)
			if err != nil {
				return err
			}

			return e.createOrUpdateEvalStatus(ctx, &createOrUpdateEvalStatusParams{
				policyID:       *pol.Id,
				repoID:         dbrepo.ID,
				ruleTypeEntity: db.Entities(rt.Def.GetInEntity()),
				ruleTypeID:     *rt.Id,
				evalErr:        rte.Eval(ctx, repo, rule.Def.AsMap(), rule.Params.AsMap()),
			})
		})
		if err != nil {
			return fmt.Errorf("error traversing rules for policy %d: %w", pol.Id, err)
		}

	}

	return nil
}

func (e *Executor) getEvaluator(
	ctx context.Context,
	policyID int32,
	prov string,
	cli ghclient.RestAPI,
	token string,
	ectx *EntityContext,
	rule *pb.PipelinePolicy_Rule,
) (*pb.RuleType, *RuleTypeEngine, error) {
	log.Printf("Evaluating rule: %s for policy %d", rule.Type, policyID)

	dbrt, err := e.querier.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Provider: prov,
		GroupID:  ectx.Group.ID,
		Name:     rule.Type,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("error getting rule type when traversing policy %d: %w", policyID, err)
	}

	rt, err := RuleTypePBFromDB(&dbrt, ectx)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing rule type when traversing policy %d: %w", policyID, err)
	}

	// TODO(jaosorior): Rule types should be cached in memory so
	// we don't have to query the database for each rule.
	rte, err := NewRuleTypeEngine(rt, cli, token)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating rule type engine: %w", err)
	}

	return rt, rte, nil
}

func (e *Executor) buildClient(
	ctx context.Context,
	prov string,
	groupID int32,
) (ghclient.RestAPI, error) {
	encToken, err := e.querier.GetAccessTokenByGroupID(ctx,
		db.GetAccessTokenByGroupIDParams{Provider: prov, GroupID: groupID})
	if err != nil {
		return nil, fmt.Errorf("error getting access token: %w", err)
	}

	decryptedToken, err := crypto.DecryptOAuthToken(encToken.EncryptedToken)
	if err != nil {
		return nil, fmt.Errorf("error decrypting access token: %w", err)
	}

	cli, err := ghclient.NewRestClient(ctx, ghclient.GitHubConfig{
		Token: decryptedToken.AccessToken,
	}, encToken.OwnerFilter.String)
	if err != nil {
		return nil, fmt.Errorf("error creating github client: %w", err)
	}

	return cli, nil
}

func (e *Executor) upsertVersionedArtifact(
	ctx context.Context,
	repoID int32,
	versionedArtifact *pb.VersionedArtifact,
) (*db.Artifact, *db.ArtifactVersion, error) {
	sigInfo, err := protojson.Marshal(versionedArtifact.Version.SignatureVerification)
	if err != nil {
		return nil, nil, fmt.Errorf("error marshalling signature verification: %w", err)
	}

	workflowInfo, err := protojson.Marshal(versionedArtifact.Version.GithubWorkflow)
	if err != nil {
		return nil, nil, fmt.Errorf("error marshalling workflow info: %w", err)
	}

	tx, err := e.querier.BeginTransaction()
	if err != nil {
		return nil, nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := e.querier.GetQuerierWithTransaction(tx)

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	err = qtx.DeleteOldArtifactVersions(ctx,
		db.DeleteOldArtifactVersionsParams{ArtifactID: int32(versionedArtifact.Artifact.ArtifactId), CreatedAt: thirtyDaysAgo})
	if err != nil {
		// just log error, we will not remove older for now
		log.Printf("error removing older artifact versions: %v", err)
	}

	dbArtifact, err := qtx.UpsertArtifact(ctx, db.UpsertArtifactParams{
		RepositoryID:       repoID,
		ArtifactName:       versionedArtifact.Artifact.GetName(),
		ArtifactType:       versionedArtifact.Artifact.GetType(),
		ArtifactVisibility: versionedArtifact.Artifact.Visibility,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error upserting artifact: %w", err)
	}

	dbVersion, err := qtx.UpsertArtifactVersion(ctx, db.UpsertArtifactVersionParams{
		ArtifactID: dbArtifact.ID,
		Version:    versionedArtifact.Version.VersionId,
		Tags: sql.NullString{
			String: strings.Join(versionedArtifact.Version.Tags, ","),
			Valid:  true,
		},
		Sha:                   versionedArtifact.Version.Sha,
		CreatedAt:             versionedArtifact.Version.CreatedAt.AsTime(),
		SignatureVerification: sigInfo,
		GithubWorkflow:        workflowInfo,
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

// createOrUpdateEvalStatusParams is a helper struct to pass parameters to createOrUpdateEvalStatus
// to avoid confusion with the parameters order. Since at the moment all our entities are bound to
// a repo and most policies are expecting a repo, the repoID parameter is mandatory. For entities
// other than artifacts, the artifactID should be 0 which is translated to NULL in the database.
type createOrUpdateEvalStatusParams struct {
	policyID       int32
	repoID         int32
	artifactID     int32
	ruleTypeEntity db.Entities
	ruleTypeID     int32
	evalErr        error
}

func (e *Executor) createOrUpdateEvalStatus(
	ctx context.Context,
	params *createOrUpdateEvalStatusParams,
) error {
	if params == nil {
		return fmt.Errorf("createOrUpdateEvalStatusParams cannot be nil")
	}

	if errors.Is(params.evalErr, evalerrors.ErrEvaluationSkipSilently) {
		return nil
	}

	var sqlArtifactID sql.NullInt32
	if params.artifactID > 0 {
		sqlArtifactID = sql.NullInt32{
			Int32: params.artifactID,
			Valid: true,
		}
	}

	return e.querier.UpsertRuleEvaluationStatus(ctx, db.UpsertRuleEvaluationStatusParams{
		PolicyID: params.policyID,
		RepositoryID: sql.NullInt32{
			Int32: params.repoID,
			Valid: true,
		},
		ArtifactID: sqlArtifactID,
		Entity:     params.ruleTypeEntity,
		RuleTypeID: params.ruleTypeID,
		EvalStatus: errorAsEvalStatus(params.evalErr),
		Details:    errorAsDetails(params.evalErr),
	})
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

func errorAsEvalStatus(err error) db.EvalStatusTypes {
	if errors.Is(err, evalerrors.ErrEvaluationFailed) {
		return db.EvalStatusTypesFailure
	} else if errors.Is(err, evalerrors.ErrEvaluationSkipped) {
		return db.EvalStatusTypesSkipped
	} else if err != nil {
		return db.EvalStatusTypesError
	}
	return db.EvalStatusTypesSuccess
}

func errorAsDetails(err error) string {
	if err != nil {
		return err.Error()
	}

	return ""
}
