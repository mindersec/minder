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
	"github.com/stacklok/minder/internal/profiles/models"
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
	profileStore    profiles.ProfileStore
}

// NewExecutor creates a new executor
func NewExecutor(
	querier db.Store,
	providerManager manager.ProviderManager,
	metrics *ExecutorMetrics,
	historyService history.EvaluationHistoryService,
	featureFlags openfeature.IClient,
	profileStore profiles.ProfileStore,
) Executor {
	return &executor{
		querier:         querier,
		providerManager: providerManager,
		metrics:         metrics,
		historyService:  historyService,
		featureFlags:    featureFlags,
		profileStore:    profileStore,
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

	// Get the profiles in the project hierarchy which have rules for this entity type
	// along with the relevant rule instances
	profileAggregates, err := e.profileStore.GetProfilesForEvaluation(ctx, inf.ProjectID, entityType)
	if err != nil {
		return fmt.Errorf("error while retrieving profiles and rule instances: %w", err)
	}

	// For each profile, evaluate each rule and store the outcome in the database
	for _, profile := range profileAggregates {
		for _, rule := range profile.Rules {
			if err := e.evaluateRule(ctx, inf, provider, &profile, &rule, ruleEngineCache); err != nil {
				return fmt.Errorf("error evaluating entity event: %w", err)
			}
		}
	}

	return nil
}

func (e *executor) evaluateRule(
	ctx context.Context,
	inf *entities.EntityInfoWrapper,
	provider provinfv1.Provider,
	profile *models.ProfileAggregate,
	rule *models.RuleInstance,
	ruleEngineCache rtengine.Cache,
) error {
	// Create eval status params
	evalParams, err := e.createEvalStatusParams(ctx, inf, profile, rule)
	if err != nil {
		return fmt.Errorf("error creating eval status params: %w", err)
	}

	// retrieve the rule type engine from the cache
	ruleEngine, err := ruleEngineCache.GetRuleEngine(ctx, rule.RuleTypeID)
	if err != nil {
		return fmt.Errorf("error creating rule type engine: %w", err)
	}

	// create the action engine for this rule instance
	// unlike the rule type engine, this cannot be cached
	actionEngine, err := actions.NewRuleActions(ctx, ruleEngine.GetRuleType(), provider, &profile.ActionConfig)
	if err != nil {
		return fmt.Errorf("cannot create rule actions engine: %w", err)
	}

	evalParams.SetActionsOnOff(actionEngine.GetOnOffState())

	// Update the lock lease at the end of the evaluation
	defer e.updateLockLease(ctx, *inf.ExecutionID, evalParams)

	// Evaluate the rule
	evalErr := ruleEngine.Eval(ctx, inf, evalParams)
	evalParams.SetEvalErr(evalErr)

	// Perform actionEngine, if any
	actionsErr := actionEngine.DoActions(ctx, inf.Entity, evalParams)
	evalParams.SetActionsErr(ctx, actionsErr)

	// Log the evaluation
	logEval(ctx, inf, evalParams, ruleEngine.GetRuleType().Name)

	// Create or update the evaluation status
	return e.createOrUpdateEvalStatus(ctx, evalParams)
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
