// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/rs/zerolog"

	datasourceservice "github.com/mindersec/minder/internal/datasources/service"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/actions"
	"github.com/mindersec/minder/internal/engine/actions/alert"
	"github.com/mindersec/minder/internal/engine/actions/remediate"
	"github.com/mindersec/minder/internal/engine/entities"
	evalerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/ingestcache"
	engif "github.com/mindersec/minder/internal/engine/interfaces"
	eoptions "github.com/mindersec/minder/internal/engine/options"
	"github.com/mindersec/minder/internal/engine/rtengine"
	"github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/history"
	minderlogger "github.com/mindersec/minder/internal/logger"
	"github.com/mindersec/minder/internal/providers/manager"
	provsel "github.com/mindersec/minder/internal/providers/selectors"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/selectors"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/profiles"
	"github.com/mindersec/minder/pkg/profiles/models"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
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
	selBuilder      selectors.SelectionBuilder
	propService     service.PropertiesService
}

// NewExecutor creates a new executor
func NewExecutor(
	querier db.Store,
	providerManager manager.ProviderManager,
	metrics *ExecutorMetrics,
	historyService history.EvaluationHistoryService,
	featureFlags openfeature.IClient,
	profileStore profiles.ProfileStore,
	selBuilder selectors.SelectionBuilder,
	propService service.PropertiesService,
) Executor {
	return &executor{
		querier:         querier,
		providerManager: providerManager,
		metrics:         metrics,
		historyService:  historyService,
		featureFlags:    featureFlags,
		profileStore:    profileStore,
		selBuilder:      selBuilder,
		propService:     propService,
	}
}

// EvalEntityEvent evaluates the entity specified in the EntityInfoWrapper
// against all relevant rules in the project hierarchy.
func (e *executor) EvalEntityEvent(ctx context.Context, inf *entities.EntityInfoWrapper) error {
	logger := zerolog.Ctx(ctx).With().
		Str("entity_type", inf.Type.ToString()).
		Str("execution_id", inf.ExecutionID.String()).
		Str("provider_id", inf.ProviderID.String()).
		Str("project_id", inf.ProjectID.String()).
		Logger()
	logger.Info().Msg("entity evaluation - started")
	// Propagate info to remaining log messages
	ctx = logger.WithContext(ctx)

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

	dssvc := datasourceservice.NewDataSourceService(e.querier)

	entityType := entities.EntityTypeToDB(inf.Type)
	// Load all the relevant rule type engines for this entity
	ruleEngineCache, err := rtengine.NewRuleEngineCache(
		ctx,
		e.querier,
		entityType,
		inf.ProjectID,
		provider,
		ingestCache,
		dssvc,
		eoptions.WithFlagsClient(e.featureFlags),
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

	sacctx, sac := actions.WithSharedActionsContext(ctx)

	// For each profile, get the profileEvalStatus first. Then, if the profileEvalStatus is nil
	// evaluate each rule and store the outcome in the database. If profileEvalStatus is non-nil,
	// just store it for all rules without evaluation.
	for _, profile := range profileAggregates {

		profileEvalStatus := e.profileEvalStatus(sacctx, inf, profile)

		for _, rule := range profile.Rules {
			if err := e.evaluateRule(sacctx, inf, provider, &profile, &rule, ruleEngineCache, profileEvalStatus); err != nil {
				return fmt.Errorf("error evaluating entity event: %w", err)
			}
		}
	}

	return sac.Flush()
}

func (e *executor) evaluateRule(
	ctx context.Context,
	inf *entities.EntityInfoWrapper,
	provider provinfv1.Provider,
	profile *models.ProfileAggregate,
	rule *models.RuleInstance,
	ruleEngineCache rtengine.Cache,
	profileEvalStatus error,
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

	// Update the lock lease at the end of the evaluation
	defer e.updateLockLease(ctx, *inf.ExecutionID, evalParams)

	// Evaluate the rule
	var evalErr error
	var result *interfaces.EvaluationResult
	if profileEvalStatus != nil {
		evalErr = profileEvalStatus
	} else {
		// enrich the logger with the entity type and execution ID
		ctx := zerolog.Ctx(ctx).With().
			Str("entity_type", inf.Type.ToString()).
			Str("execution_id", inf.ExecutionID.String()).
			Logger().WithContext(ctx)
		result, evalErr = ruleEngine.Eval(ctx, inf.Entity, evalParams.GetRule().Def, evalParams.GetRule().Params, evalParams)
		evalParams.SetEvalResult(result)
	}
	evalParams.SetEvalErr(evalErr)

	// Perform actionEngine, if any
	actionsErr := actionEngine.DoActions(ctx, inf.Entity, evalParams)
	evalParams.SetActionsErr(ctx, actionsErr)

	// Log the evaluation
	logEval(ctx, inf, evalParams, ruleEngine.GetRuleType().Name)

	// Create or update the evaluation status
	return e.createOrUpdateEvalStatus(ctx, evalParams)
}

func (e *executor) profileEvalStatus(
	ctx context.Context,
	eiw *entities.EntityInfoWrapper,
	aggregate models.ProfileAggregate,
) error {
	// so far this function only handles selectors. In the future we can extend it to handle other
	// profile-global evaluations

	if len(aggregate.Selectors) == 0 {
		return nil
	}

	selection, err := e.selBuilder.NewSelectionFromProfile(eiw.Type, aggregate.Selectors)
	if err != nil {
		return fmt.Errorf("error creating selection from profile: %w", err)
	}

	// get the entity UUID (the primary key in the database)
	entityID, err := eiw.GetID()
	if err != nil {
		return fmt.Errorf("error getting entity id: %w", err)
	}

	// get the entity with properties by the entity UUID
	ewp, err := e.propService.EntityWithPropertiesByID(ctx, entityID,
		service.CallBuilder().WithStoreOrTransaction(e.querier))
	if err != nil {
		return fmt.Errorf("error getting entity with properties: %w", err)
	}

	selEnt := provsel.EntityToSelectorEntity(ctx, e.querier, eiw.Type, ewp)
	if selEnt == nil {
		return fmt.Errorf("error converting entity to selector entity")
	}

	selected, matchedSelector, err := selection.Select(selEnt)
	if err != nil {
		return fmt.Errorf("error selecting entity: %w", err)
	}

	if !selected {
		return evalerrors.NewErrEvaluationSkipped("entity not applicable due to profile selector %s", matchedSelector)
	}

	return nil
}

func (e *executor) updateLockLease(
	ctx context.Context,
	executionID uuid.UUID,
	params *engif.EvalStatusParams,
) {
	logger := params.DecorateLogger(
		zerolog.Ctx(ctx).With().Str("execution_id", executionID.String()).Logger())

	if err := e.querier.UpdateLease(ctx, db.UpdateLeaseParams{
		LockedBy:         executionID,
		EntityInstanceID: params.EntityID,
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
	eID, err := inf.GetID()
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("error getting entity id")
		return
	}

	logger := zerolog.Ctx(ctx).Info().
		Str("entity_type", inf.Type.ToString()).
		Str("execution_id", inf.ExecutionID.String()).
		Str("entity_id", eID.String())

	if err := e.querier.ReleaseLock(ctx, db.ReleaseLockParams{
		EntityInstanceID: eID,
		LockedBy:         *inf.ExecutionID,
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
