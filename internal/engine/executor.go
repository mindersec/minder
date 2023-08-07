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
	"strconv"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-playground/validator/v10"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stacklok/mediator/internal/events"
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
}

// InitEvent is an event that is sent to the init topic
// Note that this event assumes the `provider` is set in the metadata
type InitEvent struct {
	// Group is the group that the event is relevant to
	Group int32 `json:"group" validate:"gte=0"`
	// Policy is the policy that the event is relevant to
	Policy int32 `json:"policy" validate:"gte=0"`
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

	cli, _, err := e.buildClient(ctx, prov, evt.Group)
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
		relevant, err := GetRulesForEntity(pol, RepositoryEntity)
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

			// Let's evaluate all the rules for this policy
			err = TraverseRules(relevant, func(rule *pb.PipelinePolicy_Rule) error {
				rt, rte, err := e.getEvaluator(ctx, *pol.Id, prov, cli, "", ectx, rule)
				if err != nil {
					return err
				}

				return e.createOrUpdateRepositoryEvalStatus(ctx, *pol.Id, dbrepo.ID, *rt.Id,
					rte.Eval(ctx, repo, rule.Def.AsMap(), rule.Params.AsMap()))
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
	isArtifactPublished := (payload["package"] != nil && payload["action"] == "published")
	if isArtifactPublished {
		return e.handleArtifactPublishedEvent(ctx, ghclient.Github, payload)
	}

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

func (e *Executor) handleArtifactPublishedEvent(ctx context.Context, prov string, payload map[string]any) error {
	// we need to have information about package and repository
	if payload["package"] == nil || payload["repository"] == nil {
		log.Printf("could not determine relevant entity for event. Skipping execution.")
		return nil
	}

	// extract information about repository so we can identity the group and associated rules
	dbrepo, err := e.getRepoInformationFromPayload(ctx, prov, payload)
	if err != nil {
		return err
	}
	g := dbrepo.GroupID

	// get group info
	group, err := e.querier.GetGroupByID(ctx, g)
	if err != nil {
		return fmt.Errorf("error getting group: %w", err)
	}

	cli, token, err := e.buildClient(ctx, prov, g)
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
		relevant, err := GetRulesForEntity(pol, ArtifactEntity)
		if err != nil {
			return fmt.Errorf("error getting rules for entity: %w", err)
		}

		// Let's evaluate all the rules for this policy
		err = TraverseRules(relevant, func(rule *pb.PipelinePolicy_Rule) error {
			_, rte, err := e.getEvaluator(ctx, *pol.Id, prov, cli, token, ectx, rule)
			if err != nil {
				return err
			}
			package_info, ok := payload["package"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("error getting package info")
			}
			package_version, ok := package_info["package_version"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("error getting package version info")
			}
			container_metadata, ok := package_version["container_metadata"].(map[string]interface{})
			if !ok {
				return fmt.Errorf("error getting container metadata")
			}
			tag := ""
			if container_metadata["tag"] != nil {
				tag_info := container_metadata["tag"].(map[string]interface{})
				tag = tag_info["name"].(string)
			}

			artifact := &pb.ArtifactEventPayload{
				ArtifactId:   int64(package_info["id"].(float64)),
				ArtifactName: package_info["name"].(string),
				ArtifactType: package_info["package_type"].(string),
				OwnerLogin:   package_info["owner"].(map[string]interface{})["login"].(string),
				OwnerType:    package_info["owner"].(map[string]interface{})["type"].(string),
				VersionId:    int64(package_version["id"].(float64)),
				VersionSha:   package_version["version"].(string),
				Tag:          tag,
				PackageUrl:   package_version["package_url"].(string),
			}

			result := rte.Eval(ctx, artifact, rule.Def.AsMap(), rule.Params.AsMap())
			fmt.Println(result)
			return nil

			//return e.createOrUpdateRepositoryEvalStatus(ctx, *pol.Id, dbrepo.ID, *rt.Id,
			//	rte.Eval(ctx, repo, rule.Def.AsMap(), rule.Params.AsMap()))
		})
		if err != nil {
			return fmt.Errorf("error traversing rules for policy %d: %w", pol.Id, err)
		}

	}

	return nil
}

func (e *Executor) getRepoInformationFromPayload(ctx context.Context, prov string, payload map[string]any) (db.Repository, error) {
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

	cli, _, err := e.buildClient(ctx, prov, g)
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
		relevant, err := GetRulesForEntity(pol, RepositoryEntity)
		if err != nil {
			return fmt.Errorf("error getting rules for entity: %w", err)
		}

		// Let's evaluate all the rules for this policy
		err = TraverseRules(relevant, func(rule *pb.PipelinePolicy_Rule) error {
			rt, rte, err := e.getEvaluator(ctx, *pol.Id, prov, cli, "", ectx, rule)
			if err != nil {
				return err
			}

			return e.createOrUpdateRepositoryEvalStatus(ctx, *pol.Id, dbrepo.ID, *rt.Id,
				rte.Eval(ctx, repo, rule.Def.AsMap(), rule.Params.AsMap()))
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
) (ghclient.RestAPI, string, error) {
	encToken, err := e.querier.GetAccessTokenByGroupID(ctx,
		db.GetAccessTokenByGroupIDParams{Provider: prov, GroupID: groupID})
	if err != nil {
		return nil, "", fmt.Errorf("error getting access token: %w", err)
	}

	decryptedToken, err := crypto.DecryptOAuthToken(encToken.EncryptedToken)
	if err != nil {
		return nil, "", fmt.Errorf("error decrypting access token: %w", err)
	}

	cli, err := ghclient.NewRestClient(ctx, ghclient.GitHubConfig{
		Token: decryptedToken.AccessToken,
	})
	if err != nil {
		return nil, "", fmt.Errorf("error creating github client: %w", err)
	}

	return cli, decryptedToken.AccessToken, nil
}

func (e *Executor) createOrUpdateRepositoryEvalStatus(
	ctx context.Context,
	policyID int32,
	repoID int32,
	ruleTypeID int32,
	evalErr error,
) error {
	return e.querier.UpsertRuleEvaluationStatusForRepository(ctx, db.UpsertRuleEvaluationStatusForRepositoryParams{
		PolicyID: policyID,
		RepositoryID: sql.NullInt32{
			Int32: repoID,
			Valid: true,
		},
		RuleTypeID: ruleTypeID,
		EvalStatus: errorAsEvalStatus(evalErr),
		Details:    errorAsDetails(evalErr),
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
	if errors.Is(err, ErrEvaluationFailed) {
		return db.EvalStatusTypesFailure
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
