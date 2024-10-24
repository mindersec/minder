// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package eea provides objects and event handlers for the EEA. EEA stands for
// Event Execution Aggregator. The EEA is responsible for aggregating events
// from the webhook and making sure we don't send too many events to the
// executor engine.
package eea

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/entities"
	"github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/events"
	"github.com/mindersec/minder/internal/providers/manager"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

// EEA is the Event Execution Aggregator
type EEA struct {
	querier db.Store
	evt     events.Publisher
	cfg     *serverconfig.AggregatorConfig

	entityFetcher service.PropertiesService
	provMan       manager.ProviderManager
}

// NewEEA creates a new EEA
func NewEEA(querier db.Store, evt events.Publisher, cfg *serverconfig.AggregatorConfig,
	ef service.PropertiesService, provMan manager.ProviderManager) *EEA {
	return &EEA{
		querier:       querier,
		evt:           evt,
		cfg:           cfg,
		entityFetcher: ef,
		provMan:       provMan,
	}
}

// Register implements the Consumer interface.
func (e *EEA) Register(r events.Registrar) {
	r.Register(events.TopicQueueEntityFlush, e.FlushMessageHandler)
}

// AggregateMiddleware will pass on the event to the executor engine
// if the event is ready to be executed. Else it'll cache
// the event until it's ready to be executed.
func (e *EEA) AggregateMiddleware(h message.HandlerFunc) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		msg, err := e.aggregate(msg)
		if err != nil {
			return nil, fmt.Errorf("error aggregating event: %w", err)
		}

		if msg == nil {
			return nil, nil
		}

		return h(msg)
	}
}

// nolint:gocyclo // TODO: hacking in the TODO about foreign keys pushed this over the limit.
func (e *EEA) aggregate(msg *message.Message) (*message.Message, error) {
	ctx := msg.Context()
	inf, err := entities.ParseEntityEvent(msg)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling payload: %w", err)
	}

	projectID := inf.ProjectID

	logger := zerolog.Ctx(ctx).With().
		Str("component", "EEA").
		// This is added for consistency with how watermill
		// tracks message UUID when logging.
		Str("message_uuid", msg.UUID).
		Str("entity", inf.Type.ToString()).
		Logger()

	entityID, err := inf.GetID()
	if err != nil {
		logger.Debug().AnErr("error getting entity ID", err).Msgf("Entity ID was not set for event %s", inf.Type)
		// Nothing we can do after this.
		return nil, nil
	}

	logger = logger.With().Str("entity_id", entityID.String()).Logger()

	tx, err := e.querier.BeginTransaction()
	if err != nil {
		return nil, fmt.Errorf("error beginning transaction: %w", err)
	}
	qtx := e.querier.GetQuerierWithTransaction(tx)

	// We'll only attempt to lock if the entity exists.
	_, err = qtx.GetEntityByID(ctx, entityID)
	if err != nil {
		// explicit rollback if entity had an issue.
		_ = e.querier.Rollback(tx)
		if errors.Is(err, sql.ErrNoRows) {
			logger.Debug().Msg("entity not found")
			return nil, nil
		}
		return nil, fmt.Errorf("error getting entity: %w", err)
	}

	res, err := qtx.LockIfThresholdNotExceeded(ctx, db.LockIfThresholdNotExceededParams{
		Entity:           entities.EntityTypeToDB(inf.Type),
		EntityInstanceID: entityID,
		ProjectID:        projectID,
		Interval:         fmt.Sprintf("%d", e.cfg.LockInterval),
	})
	if err == nil {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("error committing transaction: %w", err)
		}
	} else {
		_ = e.querier.Rollback(tx)
	}

	// if nothing was retrieved from the database, then we can assume
	// that the event is not ready to be executed.
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		logger.Info().Msg("executor not ready to process event. Queuing in flush cache.")

		_, err := e.querier.EnqueueFlush(ctx, db.EnqueueFlushParams{
			Entity:           entities.EntityTypeToDB(inf.Type),
			EntityInstanceID: entityID,
			ProjectID:        projectID,
		})
		if err != nil {
			// We already have this item in the queue.
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, fmt.Errorf("error enqueuing flush: %w", err)
		}

		return nil, nil
	} else if err != nil {
		logger.Err(err).Msg("error locking event")
		return nil, fmt.Errorf("error locking: %w", err)
	}

	logger.Info().Str("execution_id", res.LockedBy.String()).Msg("event ready to be executed")
	msg.Metadata.Set(entities.ExecutionIDKey, res.LockedBy.String())

	return msg, nil
}

// FlushMessageHandler will flush the cache of events to the executor engine
// if the event is ready to be executed.
func (e *EEA) FlushMessageHandler(msg *message.Message) error {
	ctx := msg.Context()

	inf, err := entities.ParseEntityEvent(msg)
	if err != nil {
		return fmt.Errorf("error unmarshalling payload: %w", err)
	}

	eID, err := inf.GetID()
	if err != nil {
		return fmt.Errorf("error getting entity ID: %w", err)
	}

	logger := zerolog.Ctx(ctx).With().
		Str("component", "EEA").
		Str("function", "FlushMessageHandler").
		// This is added for consistency with how watermill
		// tracks message UUID when logging.
		Str("message_uuid", msg.UUID).
		Str("entity", inf.Type.ToString()).Logger()

	logger.Debug().Msg("flushing event")

	_, err = e.querier.FlushCache(ctx, eID)
	// Nothing to do here. If we can't flush the cache, it means
	// that the event has already been executed.
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		zerolog.Ctx(ctx).Debug().Msg("no flushing needed")
		return nil
	} else if err != nil {
		return fmt.Errorf("error flushing cache: %w", err)
	}

	logger.Debug().Msg("re-publishing event because of flush")

	// Now that we've flushed the event, let's try to publish it again
	// which means, go through the locking process again.
	if err := inf.Publish(e.evt); err != nil {
		return fmt.Errorf("error publishing execute event: %w", err)
	}

	return nil
}

// FlushAll will flush all events in the cache to the executor engine
func (e *EEA) FlushAll(ctx context.Context) error {
	caches, err := e.querier.ListFlushCache(ctx)
	if err != nil {
		return fmt.Errorf("error listing flush cache: %w", err)
	}

	for _, cache := range caches {
		cache := cache

		eiw, err := e.buildEntityWrapper(ctx, cache.Entity,
			cache.ProjectID, cache.EntityInstanceID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) || errors.Is(err, service.ErrEntityNotFound) {
				continue
			}
			return fmt.Errorf("error building entity wrapper: %w", err)
		}

		msg, err := eiw.BuildMessage()
		if err != nil {
			return fmt.Errorf("error building message: %w", err)
		}

		msg.SetContext(ctx)

		if err := e.FlushMessageHandler(msg); err != nil {
			return fmt.Errorf("error flushing messages: %w", err)
		}
	}

	return nil
}

func (e *EEA) buildEntityWrapper(
	ctx context.Context,
	entity db.Entities,
	projID uuid.UUID,
	entityID uuid.UUID,
) (*entities.EntityInfoWrapper, error) {
	switch entity {
	case db.EntitiesRepository:
		return e.buildRepositoryInfoWrapper(ctx, entityID, projID)
	case db.EntitiesArtifact:
		return e.buildArtifactInfoWrapper(ctx, entityID, projID)
	case db.EntitiesPullRequest:
		return e.buildPullRequestInfoWrapper(ctx, entityID, projID)
	case db.EntitiesBuildEnvironment, db.EntitiesRelease,
		db.EntitiesPipelineRun, db.EntitiesTaskRun, db.EntitiesBuild:
		return nil, fmt.Errorf("entity type %q not yet supported", entity)
	default:
		return nil, fmt.Errorf("unknown entity type: %q", entity)
	}
}

func (e *EEA) buildRepositoryInfoWrapper(
	ctx context.Context,
	repoID uuid.UUID,
	projID uuid.UUID,
) (*entities.EntityInfoWrapper, error) {
	ent, err := e.entityFetcher.EntityWithPropertiesByID(ctx, repoID, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching entity: %w", err)
	}

	if ent.Entity.ProjectID != projID {
		return nil, fmt.Errorf("entity %s does not belong to project %s", repoID, projID)
	}

	rawRepo, err := e.entityFetcher.EntityWithPropertiesAsProto(ctx, ent, e.provMan)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to protobuf: %w", err)
	}

	r, ok := rawRepo.(*minderv1.Repository)
	if !ok {
		return nil, fmt.Errorf("error converting entity to repository")
	}

	return entities.NewEntityInfoWrapper().
		WithRepository(r).
		WithID(repoID).
		WithProjectID(projID).
		WithProviderID(ent.Entity.ProviderID), nil
}

func (e *EEA) buildArtifactInfoWrapper(
	ctx context.Context,
	artID uuid.UUID,
	projID uuid.UUID,
) (*entities.EntityInfoWrapper, error) {
	ent, err := e.entityFetcher.EntityWithPropertiesByID(ctx, artID, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching entity: %w", err)
	}

	if ent.Entity.ProjectID != projID {
		return nil, fmt.Errorf("entity %s does not belong to project %s", artID, projID)
	}

	rawPR, err := e.entityFetcher.EntityWithPropertiesAsProto(ctx, ent, e.provMan)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to protobuf: %w", err)
	}

	a, ok := rawPR.(*minderv1.Artifact)
	if !ok {
		return nil, fmt.Errorf("error converting entity to artifact")
	}

	eiw := entities.NewEntityInfoWrapper().
		WithProjectID(projID).
		WithArtifact(a).
		WithID(artID).
		WithProviderID(ent.Entity.ProviderID)
	return eiw, nil
}

func (e *EEA) buildPullRequestInfoWrapper(
	ctx context.Context,
	prID uuid.UUID,
	projID uuid.UUID,
) (*entities.EntityInfoWrapper, error) {
	ent, err := e.entityFetcher.EntityWithPropertiesByID(ctx, prID, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching entity: %w", err)
	}

	if ent.Entity.ProjectID != projID {
		return nil, fmt.Errorf("entity %s does not belong to project %s", prID, projID)
	}

	rawPR, err := e.entityFetcher.EntityWithPropertiesAsProto(ctx, ent, e.provMan)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to protobuf: %w", err)
	}

	pr, ok := rawPR.(*minderv1.PullRequest)
	if !ok {
		return nil, fmt.Errorf("error converting entity to pull request")
	}

	return entities.NewEntityInfoWrapper().
		WithProjectID(projID).
		WithPullRequest(pr).
		WithID(prID).
		WithProviderID(ent.Entity.ProviderID), nil
}
