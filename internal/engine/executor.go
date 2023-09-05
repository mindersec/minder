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
	"encoding/json"
	"fmt"
	"log"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"github.com/stacklok/mediator/internal/events"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/providers"
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
	r.Register(InternalWebhookEventTopic, e.handleEntityEvent)
	r.Register(InternalInitEventTopic, e.handleEntityEvent)
	r.Register(InternalReconcilerEventTopic, e.handleReconcilerEvent)
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

// handleEntityEvent handles events coming from webhooks/signals
// as well as the init event.
func (e *Executor) handleEntityEvent(msg *message.Message) error {
	prov := msg.Metadata.Get("provider")

	// TODO(jaosorior): get provider from database
	if prov != ghclient.Github {
		log.Printf("provider %s not supported", prov)
		return nil
	}

	inf, err := parseEntityEvent(msg)
	if err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}

	log.Printf("got entity info %+v", inf)

	ctx := msg.Context()

	// get group info
	group, err := e.querier.GetGroupByID(ctx, inf.GroupID)
	if err != nil {
		return fmt.Errorf("error getting group: %w", err)
	}

	cli, err := providers.BuildClient(ctx, prov, inf.GroupID, e.querier)
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

	return e.evalEntityEvent(ctx, inf, ectx, cli)
}

func (e *Executor) evalEntityEvent(
	ctx context.Context,
	inf *entityInfoWrapper,
	ectx *EntityContext,
	cli ghclient.RestAPI,
) error {
	// Get policies relevant to group
	dbpols, err := e.querier.ListPoliciesByGroupID(ctx, inf.GroupID)
	if err != nil {
		return fmt.Errorf("error getting policies: %w", err)
	}

	for _, pol := range MergeDatabaseListIntoPolicies(dbpols, ectx) {
		// Given we're dealing with a repository event, we can assume that the
		// entity is a repository.
		relevant, err := GetRulesForEntity(pol, inf.Type)
		if err != nil {
			return fmt.Errorf("error getting rules for entity: %w", err)
		}

		// Let's evaluate all the rules for this policy
		err = TraverseRules(relevant, func(rule *pb.PipelinePolicy_Rule) error {
			rt, rte, err := e.getEvaluator(ctx, *pol.Id, ectx.Provider, cli, cli.GetToken(), ectx, rule)
			if err != nil {
				return err
			}
			result := rte.Eval(ctx, inf.Entity, rule.Def.AsMap(), rule.Params.AsMap())

			logEval(ctx, pol, rule, inf, result)

			return e.createOrUpdateEvalStatus(ctx, inf.evalStatusParams(
				*pol.Id, *rt.Id, result))
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

func logEval(
	ctx context.Context,
	pol *pb.PipelinePolicy,
	rule *pb.PipelinePolicy_Rule,
	inf *entityInfoWrapper,
	result error,
) {
	logger := zerolog.Ctx(ctx).Debug().
		Str("policy", pol.Name).
		Str("ruleType", rule.Type).
		Int32("groupId", inf.GroupID).
		Int32("repositoryId", inf.OwnershipData[RepositoryIDEventKey])

	if aID, ok := inf.OwnershipData[ArtifactIDEventKey]; ok {
		logger = logger.Int32("artifactId", aID)
	}

	logger.Err(result).Msg("evaluated rule")
}
