// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package engine

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/internal/crypto"
	"github.com/stacklok/mediator/internal/db"
	evalerrors "github.com/stacklok/mediator/internal/engine/errors"
	"github.com/stacklok/mediator/internal/engine/ingestcache"
	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	"github.com/stacklok/mediator/internal/events"
	"github.com/stacklok/mediator/internal/providers"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

const (
	// InternalEntityEventTopic is the topic for internal webhook events
	InternalEntityEventTopic = "internal.entity.event"
)

// Executor is the engine that executes the rules for a given event
type Executor struct {
	querier  db.Store
	crypteng *crypto.Engine
}

// NewExecutor creates a new executor
func NewExecutor(querier db.Store, authCfg *config.AuthConfig) (*Executor, error) {
	crypteng, err := crypto.EngineFromAuthConfig(authCfg)
	if err != nil {
		return nil, err
	}

	return &Executor{
		querier:  querier,
		crypteng: crypteng,
	}, nil
}

// Register implements the Consumer interface.
func (e *Executor) Register(r events.Registrar) {
	r.Register(InternalEntityEventTopic, e.HandleEntityEvent)
}

// HandleEntityEvent handles events coming from webhooks/signals
// as well as the init event.
func (e *Executor) HandleEntityEvent(msg *message.Message) error {
	inf, err := parseEntityEvent(msg)
	if err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}

	ctx := msg.Context()

	projectID := inf.ProjectID

	// get project info
	project, err := e.querier.GetProjectByID(ctx, *projectID)
	if err != nil {
		return fmt.Errorf("error getting group: %w", err)
	}

	provider, err := e.querier.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      inf.Provider,
		ProjectID: *projectID,
	})

	if err != nil {
		return fmt.Errorf("error getting provider: %w", err)
	}

	cli, err := providers.GetProviderBuilder(ctx, provider, *projectID, e.querier, e.crypteng)
	if err != nil {
		return fmt.Errorf("error building client: %w", err)
	}

	ectx := &EntityContext{
		Project: Project{
			ID:   project.ID,
			Name: project.Name,
		},
		Provider: Provider{
			Name: inf.Provider,
			ID:   provider.ID,
		},
	}

	return e.evalEntityEvent(ctx, inf, ectx, cli)
}

func (e *Executor) evalEntityEvent(
	ctx context.Context,
	inf *EntityInfoWrapper,
	ectx *EntityContext,
	cli *providers.ProviderBuilder,
) error {
	// this is a cache so we can avoid querying the ingester upstream
	// for every rule. We use a sync.Map because it's safe for concurrent
	// access.
	ingestCache := ingestcache.NewCache()

	// Get profiles relevant to group
	dbpols, err := e.querier.ListProfilesByProjectID(ctx, *inf.ProjectID)
	if err != nil {
		return fmt.Errorf("error getting profiles: %w", err)
	}

	for _, profile := range MergeDatabaseListIntoProfiles(dbpols, ectx) {
		// Get only these rules that are relevant for this entity type
		relevant, err := GetRulesForEntity(profile, inf.Type)
		if err != nil {
			return fmt.Errorf("error getting rules for entity: %w", err)
		}

		// Let's evaluate all the rules for this profile
		err = TraverseRules(relevant, func(rule *pb.Profile_Rule) error {
			// Get the engine evaluator for this rule type
			evalParams, rte, err := e.getEvaluator(ctx, inf, ectx, cli, profile, rule, ingestCache)
			if err != nil {
				return err
			}

			// Evaluate the rule
			evalParams.SetEvalErr(rte.Eval(ctx, inf, evalParams))

			// Perform actions, if any
			evalParams.SetActionsErr(ctx, rte.Actions(ctx, inf, evalParams))

			// Log the evaluation
			logEval(ctx, inf, evalParams)

			// Create or update the evaluation status
			return e.createOrUpdateEvalStatus(ctx, evalParams)
		})

		if err != nil {
			p := profile.Name
			if profile.Id != nil {
				p = *profile.Id
			}
			return fmt.Errorf("error traversing rules for profile %s: %w", p, err)
		}
	}
	return nil
}

func (e *Executor) getEvaluator(
	ctx context.Context,
	inf *EntityInfoWrapper,
	ectx *EntityContext,
	cli *providers.ProviderBuilder,
	profile *pb.Profile,
	rule *pb.Profile_Rule,
	ingestCache ingestcache.Cache,
) (*engif.EvalStatusParams, *RuleTypeEngine, error) {
	// Create eval status params
	params, err := e.createEvalStatusParams(ctx, inf, profile, rule)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating eval status params: %w", err)
	}

	// Load Rule Class from database
	// TODO(jaosorior): Rule types should be cached in memory so
	// we don't have to query the database for each rule.
	dbrt, err := e.querier.GetRuleTypeByName(ctx, db.GetRuleTypeByNameParams{
		Provider:  ectx.Provider.Name,
		ProjectID: ectx.Project.ID,
		Name:      rule.Type,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error getting rule type when traversing profile %s: %w", params.ProfileID, err)
	}

	// Parse the rule type
	rt, err := RuleTypePBFromDB(&dbrt, ectx)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing rule type when traversing profile %s: %w", params.ProfileID, err)
	}

	// Save the rule type uuid
	ruleTypeID, err := uuid.Parse(*rt.Id)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing rule type ID: %w", err)
	}
	params.RuleTypeID = ruleTypeID
	params.RuleType = rt

	// Create the rule type engine
	rte, err := NewRuleTypeEngine(profile, rt, cli)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating rule type engine: %w", err)
	}

	rte = rte.WithIngesterCache(ingestCache)

	// All okay
	return params, rte, nil
}

func logEval(
	ctx context.Context,
	inf *EntityInfoWrapper,
	params *engif.EvalStatusParams) {
	logger := zerolog.Ctx(ctx)
	evalLog := logger.Debug().
		Str("profile", params.Profile.Name).
		Str("ruleType", params.Rule.Type).
		Str("eval_status", string(evalerrors.ErrorAsEvalStatus(params.GetEvalErr()))).
		Str("projectId", inf.ProjectID.String()).
		Str("repositoryId", params.RepoID.String())

	if params.ArtifactID.Valid {
		evalLog = evalLog.Str("artifactId", params.ArtifactID.UUID.String())
	}

	// log evaluation
	evalLog.Err(params.GetEvalErr()).Msg("result - evaluation")

	// log remediation
	logger.Debug().
		Str("action", "remediate").
		Str("action_status", string(evalerrors.ErrorAsRemediationStatus(params.GetActionsErr().RemediateErr))).
		Msg("result - action")

	// log alert
	logger.Debug().
		Str("action", "alert").
		Str("action_status", string(evalerrors.ErrorAsAlertStatus(params.GetActionsErr().AlertErr))).
		Msg("result - action")
}
