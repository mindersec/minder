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
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	gogithub "github.com/google/go-github/v60/github"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/actions/alert"
	"github.com/stacklok/minder/internal/engine/actions/remediate"
	"github.com/stacklok/minder/internal/engine/entities"
	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/ingestcache"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/events"
	minderlogger "github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/providers"
	"github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/providers/ratecache"
	providertelemetry "github.com/stacklok/minder/internal/providers/telemetry"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	// DefaultExecutionTimeout is the timeout for execution of a set
	// of profiles on an entity.
	DefaultExecutionTimeout = 5 * time.Minute
	// ArtifactSignatureWaitPeriod is the waiting period for potential artifact signature to be available
	// before proceeding with evaluation.
	ArtifactSignatureWaitPeriod = 10 * time.Second
)

// Executor is the engine that executes the rules for a given event
type Executor struct {
	querier                db.Store
	evt                    events.Publisher
	crypteng               crypto.Engine
	provMt                 providertelemetry.ProviderMetrics
	mdws                   []message.HandlerMiddleware
	wgEntityEventExecution *sync.WaitGroup
	// terminationcontext is used to terminate the executor
	// when the server is shutting down.
	terminationcontext  context.Context
	restClientCache     ratecache.RestClientCache
	provCfg             *serverconfig.ProviderConfig
	providerStore       providers.ProviderStore
	fallbackTokenClient *gogithub.Client
}

// ExecutorOption is a function that modifies an executor
type ExecutorOption func(*Executor)

// WithProviderMetrics sets the provider metrics for the executor
func WithProviderMetrics(mt providertelemetry.ProviderMetrics) ExecutorOption {
	return func(e *Executor) {
		e.provMt = mt
	}
}

// WithMiddleware sets the aggregator middleware for the executor
func WithMiddleware(mdw message.HandlerMiddleware) ExecutorOption {
	return func(e *Executor) {
		e.mdws = append(e.mdws, mdw)
	}
}

// WithRestClientCache sets the rest client cache for the executor
func WithRestClientCache(cache ratecache.RestClientCache) ExecutorOption {
	return func(e *Executor) {
		e.restClientCache = cache
	}
}

// NewExecutor creates a new executor
func NewExecutor(
	ctx context.Context,
	querier db.Store,
	authCfg *serverconfig.AuthConfig,
	provCfg *serverconfig.ProviderConfig,
	evt events.Publisher,
	providerStore providers.ProviderStore,
	opts ...ExecutorOption,
) (*Executor, error) {
	crypteng, err := crypto.EngineFromAuthConfig(authCfg)
	if err != nil {
		return nil, err
	}

	fallbackTokenClient := github.NewFallbackTokenClient(*provCfg)

	e := &Executor{
		querier:                querier,
		crypteng:               crypteng,
		provMt:                 providertelemetry.NewNoopMetrics(),
		evt:                    evt,
		wgEntityEventExecution: &sync.WaitGroup{},
		terminationcontext:     ctx,
		mdws:                   []message.HandlerMiddleware{},
		provCfg:                provCfg,
		providerStore:          providerStore,
		fallbackTokenClient:    fallbackTokenClient,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e, nil
}

// Register implements the Consumer interface.
func (e *Executor) Register(r events.Registrar) {
	r.Register(events.ExecuteEntityEventTopic, e.HandleEntityEvent, e.mdws...)
}

// Wait waits for all the entity executions to finish.
func (e *Executor) Wait() {
	e.wgEntityEventExecution.Wait()
}

// HandleEntityEvent handles events coming from webhooks/signals
// as well as the init event.
func (e *Executor) HandleEntityEvent(msg *message.Message) error {
	// Grab the context before making a copy of the message
	msgCtx := msg.Context()
	// Let's not share memory with the caller
	msg = msg.Copy()

	inf, err := entities.ParseEntityEvent(msg)
	if err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}

	e.wgEntityEventExecution.Add(1)
	go func() {
		defer e.wgEntityEventExecution.Done()
		if inf.Type == pb.Entity_ENTITY_ARTIFACTS {
			time.Sleep(ArtifactSignatureWaitPeriod)
		}
		// TODO: Make this timeout configurable
		ctx, cancel := context.WithTimeout(e.terminationcontext, DefaultExecutionTimeout)
		defer cancel()

		ts := minderlogger.BusinessRecord(msgCtx)
		ctx = ts.WithTelemetry(ctx)

		if err := inf.WithExecutionIDFromMessage(msg); err != nil {
			logger := zerolog.Ctx(ctx)
			logger.Info().
				Str("message_id", msg.UUID).
				Msg("message does not contain execution ID, skipping")
			return
		}

		err := e.prepAndEvalEntityEvent(ctx, inf)

		// record telemetry regardless of error. We explicitly record telemetry
		// here even though we also record it in the middleware because the evaluation
		// is done in a separate goroutine which usually still runs after the middleware
		// had already recorded the telemetry.
		logMsg := zerolog.Ctx(ctx).Info()
		if err != nil {
			logMsg = zerolog.Ctx(ctx).Error()
		}
		ts.Record(logMsg).Send()

		if err != nil {
			zerolog.Ctx(ctx).Info().
				Str("project", inf.ProjectID.String()).
				Str("provider", inf.Provider).
				Str("entity", inf.Type.String()).
				Err(err).Msg("got error while evaluating entity event")
		}
	}()

	return nil
}
func (e *Executor) prepAndEvalEntityEvent(ctx context.Context, inf *entities.EntityInfoWrapper) error {
	projectID := inf.ProjectID
	provider, err := e.providerStore.GetByName(ctx, projectID, inf.Provider)
	if err != nil {
		return fmt.Errorf("error getting provider: %w", err)
	}

	pbOpts := []providers.ProviderBuilderOption{
		providers.WithProviderMetrics(e.provMt),
		providers.WithRestClientCache(e.restClientCache),
	}
	cli, err := providers.GetProviderBuilder(ctx, *provider, e.querier, e.crypteng, e.provCfg, e.fallbackTokenClient, pbOpts...)
	if err != nil {
		return fmt.Errorf("error building client: %w", err)
	}

	ectx := &EntityContext{
		Project: Project{
			ID: projectID,
		},
		Provider: Provider{
			Name: inf.Provider,
		},
	}

	return e.evalEntityEvent(ctx, inf, ectx, cli)
}

func (e *Executor) evalEntityEvent(
	ctx context.Context,
	inf *entities.EntityInfoWrapper,
	ectx *EntityContext,
	cli *providers.ProviderBuilder,
) error {
	logger := zerolog.Ctx(ctx).Info().
		Str("entity_type", inf.Type.ToString()).
		Str("execution_id", inf.ExecutionID.String())
	logger.Msg("entity evaluation - started")

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

	// Get profiles relevant to project
	dbpols, err := e.querier.ListProfilesByProjectID(ctx, inf.ProjectID)
	if err != nil {
		return fmt.Errorf("error getting profiles: %w", err)
	}

	for _, profile := range MergeDatabaseListIntoProfiles(dbpols) {
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

			// Update the lock lease at the end of the evaluation
			defer e.updateLockLease(ctx, *inf.ExecutionID, evalParams)

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
	inf *entities.EntityInfoWrapper,
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
		ProjectID: ectx.Project.ID,
		Name:      rule.Type,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error getting rule type when traversing profile %s: %w", params.ProfileID, err)
	}

	// Parse the rule type
	rt, err := RuleTypePBFromDB(&dbrt)
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
	rte, err := NewRuleTypeEngine(ctx, profile, rt, cli)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating rule type engine: %w", err)
	}

	rte = rte.WithIngesterCache(ingestCache)

	// All okay
	params.SetActionsOnOff(rte.GetActionsOnOff())
	return params, rte, nil
}

func (e *Executor) updateLockLease(
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

func (e *Executor) releaseLockAndFlush(
	ctx context.Context,
	inf *entities.EntityInfoWrapper,
) {
	repoID, artID, prID := inf.GetEntityDBIDs()

	logger := zerolog.Ctx(ctx).Info().
		Str("entity_type", inf.Type.ToString()).
		Str("execution_id", inf.ExecutionID.String()).
		Str("repo_id", repoID.String())

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

	// We don't need to unset the execution ID because the event is going to be
	// deleted from the database anyway. The aggregator will take care of that.
	msg, err := inf.BuildMessage()
	if err != nil {
		logger.Err(err).Msg("error building message")
		return
	}

	if err := e.evt.Publish(events.FlushEntityEventTopic, msg); err != nil {
		logger.Err(err).Msg("error publishing flush event")
	}
}

func logEval(
	ctx context.Context,
	inf *entities.EntityInfoWrapper,
	params *engif.EvalStatusParams) {
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
	minderlogger.BusinessRecord(ctx).AddRuleEval(params)
}
