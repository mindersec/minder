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
	"time"

	"github.com/google/uuid"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/actions"
	"github.com/stacklok/minder/internal/engine/actions/alert"
	"github.com/stacklok/minder/internal/engine/actions/remediate"
	"github.com/stacklok/minder/internal/engine/entities"
	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/ingestcache"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/engine/rtengine"
	"github.com/stacklok/minder/internal/history"
	minderlogger "github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/profiles"
	"github.com/stacklok/minder/internal/providers/manager"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// Executor is the engine that executes the rules for a given event
type Executor interface {
	EvalEntityEvent(ctx context.Context, inf *entities.EntityInfoWrapper) error
}

type executor struct {
	querier         db.Store
	providerManager manager.ProviderManager
	metrics         *ExecutorMetrics
	historyService  history.EvaluationHistoryService
	featureFlags    openfeature.IClient
}

// NewExecutor creates a new executor
func NewExecutor(
	querier db.Store,
	providerManager manager.ProviderManager,
	metrics *ExecutorMetrics,
	historyService history.EvaluationHistoryService,
	featureFlags openfeature.IClient,
) Executor {
	return &executor{
		querier:         querier,
		providerManager: providerManager,
		metrics:         metrics,
		historyService:  historyService,
		featureFlags:    featureFlags,
	}
}

// EvalEntityEvent evaluates the entity specified in the EntityInfoWrapper
// against all relevant rules in the project hierarchy.
func (e *executor) EvalEntityEvent(ctx context.Context, inf *entities.EntityInfoWrapper) error {
	logger := zerolog.Ctx(ctx).Info().
		Str("entity_type", inf.Type.ToString()).
		Str("execution_id", inf.ExecutionID.String()).
		Str("provider_id", inf.ProviderID.String()).
		Str("project_id", inf.ProjectID.String())
	logger.Msg("entity evaluation - started")

	// track the time taken to evaluate each entity
	entityStartTime := time.Now()
	defer e.metrics.TimeProfileEvaluation(ctx, entityStartTime)

	provider, err := e.providerManager.InstantiateFromID(ctx, inf.ProviderID)
	if err != nil {
		return fmt.Errorf("could not instantiate provider: %w", err)
	}

	// This is a cache, so we can avoid querying the ingester upstream
	// for every rule. We use a sync.Map because it's safe for concurrent
	// access.
	var ingestCache ingestcache.Cache
	if inf.Type == pb.Entity_ENTITY_ARTIFACTS {
		// We use a noop cache for artifacts because we don't want to cache
		// anything for them. The signature information is essentially another artifact version,
		// and so we don't want to cache that.
		ingestCache = ingestcache.NewNoopCache()
	} else {
		ingestCache = ingestcache.NewCache()
	}

	defer e.releaseLockAndFlush(ctx, inf)

	entityType := entities.EntityTypeToDB(inf.Type)
	// Load all the relevant rule type engines for this entity
	ruleEngineCache, err := rtengine.NewRuleEngineCache(
		ctx,
		e.querier,
		entityType,
		inf.ProjectID,
		provider,
		ingestCache,
	)
	if err != nil {
		return fmt.Errorf("unable to fetch rule type instances for project: %w", err)
	}

	err = e.forProjectsInHierarchy(
		ctx, inf, func(ctx context.Context, profile *pb.Profile) error {
			profileStartTime := time.Now()
			defer e.metrics.TimeProfileEvaluation(ctx, profileStartTime)
			// Get only these rules that are relevant for this entity type
			relevant, err := profiles.GetRulesForEntity(profile, inf.Type)
			if err != nil {
				return fmt.Errorf("error getting rules for entity: %w", err)
			}

			// Let's evaluate all the rules for this profile
			err = profiles.TraverseRules(relevant, func(rule *pb.Profile_Rule) error {
				// Get the engine evaluator for this rule type
				evalParams, ruleEngine, actionEngine, err := e.getEvaluator(
					ctx, inf, provider, profile, rule, ruleEngineCache)
				if err != nil {
					return err
				}

				// Update the lock lease at the end of the evaluation
				defer e.updateLockLease(ctx, *inf.ExecutionID, evalParams)

				// Evaluate the rule
				evalErr := ruleEngine.Eval(ctx, inf, evalParams)
				evalParams.SetEvalErr(evalErr)

				// Perform actionEngine, if any
				actionsErr := actionEngine.DoActions(ctx, inf.Entity, evalParams)
				evalParams.SetActionsErr(ctx, actionsErr)

				// Log the evaluation
				logEval(ctx, inf, evalParams, ruleEngine.Meta.Name)

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

			return nil
		})

	if err != nil {
		return fmt.Errorf("error evaluating entity event: %w", err)
	}

	return nil
}

func (e *executor) forProjectsInHierarchy(
	ctx context.Context,
	inf *entities.EntityInfoWrapper,
	f func(context.Context, *pb.Profile) error,
) error {
	projList, err := e.querier.GetParentProjects(ctx, inf.ProjectID)
	if err != nil {
		return fmt.Errorf("error getting parent projects: %w", err)
	}

	for _, projID := range projList {
		// Get profiles relevant to project
		dbpols, err := e.querier.ListProfilesByProjectID(ctx, projID)
		if err != nil {
			return fmt.Errorf("error getting profiles: %w", err)
		}

		for _, profile := range profiles.MergeDatabaseListIntoProfiles(dbpols) {
			if err := f(ctx, profile); err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *executor) getEvaluator(
	ctx context.Context,
	inf *entities.EntityInfoWrapper,
	provider provinfv1.Provider,
	profile *pb.Profile,
	rule *pb.Profile_Rule,
	ruleEngineCache rtengine.Cache,
) (*engif.EvalStatusParams, *rtengine.RuleTypeEngine, *actions.RuleActionsEngine, error) {
	// Create eval status params
	params, err := e.createEvalStatusParams(ctx, inf, profile, rule)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating eval status params: %w", err)
	}

	if profile.Id == nil {
		return nil, nil, nil, fmt.Errorf("profile %s missing ID", profile.Name)
	}
	profileID, err := uuid.Parse(*profile.Id)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unable to parse %s as profile ID", *profile.Id)
	}

	// TODO: Once we use the rule instance table, this will no longer be necessary
	ruleTypeID, err := e.querier.GetRuleTypeIDByRuleNameEntityProfile(ctx,
		db.GetRuleTypeIDByRuleNameEntityProfileParams{
			ProfileID:  profileID,
			EntityType: entities.EntityTypeToDB(inf.Type),
			Name:       rule.Name,
		},
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unable to retrieve rule type ID: %w", err)
	}

	params.RuleTypeID = ruleTypeID

	rte, err := ruleEngineCache.GetRuleEngine(ctx, ruleTypeID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error creating rule type engine: %w", err)
	}

	actionEngine, err := actions.NewRuleActions(ctx, profile, rte.GetRuleType(), provider)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cannot create rule actions engine: %w", err)
	}

	// All okay
	params.SetActionsOnOff(actionEngine.GetOnOffState())
	return params, rte, actionEngine, nil
}

func (e *executor) updateLockLease(
	ctx context.Context,
	executionID uuid.UUID,
	params *engif.EvalStatusParams,
) {
	logger := params.DecorateLogger(
		zerolog.Ctx(ctx).With().Str("execution_id", executionID.String()).Logger())

	if err := e.querier.UpdateLease(ctx, db.UpdateLeaseParams{
		Entity:        params.EntityType,
		RepositoryID:  params.RepoID,
		ArtifactID:    params.ArtifactID,
		PullRequestID: params.PullRequestID,
		LockedBy:      executionID,
	}); err != nil {
		logger.Err(err).Msg("error updating lock lease")
		return
	}

	logger.Info().Msg("lock lease updated")
}

func (e *executor) releaseLockAndFlush(
	ctx context.Context,
	inf *entities.EntityInfoWrapper,
) {
	repoID, artID, prID := inf.GetEntityDBIDs()

	logger := zerolog.Ctx(ctx).Info().
		Str("entity_type", inf.Type.ToString()).
		Str("execution_id", inf.ExecutionID.String())
	if repoID.Valid {
		logger = logger.Str("repo_id", repoID.UUID.String())
	}

	if artID.Valid {
		logger = logger.Str("artifact_id", artID.UUID.String())
	}
	if prID.Valid {
		logger = logger.Str("pull_request_id", prID.UUID.String())
	}

	if err := e.querier.ReleaseLock(ctx, db.ReleaseLockParams{
		Entity:        entities.EntityTypeToDB(inf.Type),
		RepositoryID:  repoID,
		ArtifactID:    artID,
		PullRequestID: prID,
		LockedBy:      *inf.ExecutionID,
	}); err != nil {
		logger.Err(err).Msg("error updating lock lease")
	}
}

func logEval(
	ctx context.Context,
	inf *entities.EntityInfoWrapper,
	params *engif.EvalStatusParams,
	ruleTypeName string,
) {
	evalLog := params.DecorateLogger(
		zerolog.Ctx(ctx).With().
			Str("eval_status", string(evalerrors.ErrorAsEvalStatus(params.GetEvalErr()))).
			Str("project_id", inf.ProjectID.String()).
			Logger())

	// log evaluation result and actions status
	evalLog.Info().
		Str("action", string(remediate.ActionType)).
		Str("action_status", string(evalerrors.ErrorAsRemediationStatus(params.GetActionsErr().RemediateErr))).
		Str("action", string(alert.ActionType)).
		Str("action_status", string(evalerrors.ErrorAsAlertStatus(params.GetActionsErr().AlertErr))).
		Msg("entity evaluation - completed")

	// log business logic
	minderlogger.BusinessRecord(ctx).AddRuleEval(params, ruleTypeName)
}
