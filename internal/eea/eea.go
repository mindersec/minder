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
	"github.com/mindersec/minder/internal/providers/manager"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer/constants"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

// EEA is the Event Execution Aggregator
type EEA struct {
	querier db.Store
	evt     interfaces.Publisher
	cfg     *serverconfig.AggregatorConfig

	entityFetcher service.PropertiesService
	provMan       manager.ProviderManager
}

// NewEEA creates a new EEA
func NewEEA(querier db.Store, evt interfaces.Publisher, cfg *serverconfig.AggregatorConfig,
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
func (e *EEA) Register(r interfaces.Registrar) {
	r.Register(constants.TopicQueueEntityFlush, e.FlushMessageHandler)
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

		eiw, err := e.buildEntityWrapper(ctx, cache.Entity,
			cache.ProjectID, cache.EntityInstanceID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) || errors.Is(err, service.ErrEntityNotFound) {
				continue
			}
			e.dropFlushCacheEntry(ctx, cache, fmt.Errorf("error building entity wrapper: %w", err))
			continue
		}

		msg, err := eiw.BuildMessage()
		if err != nil {
			e.dropFlushCacheEntry(ctx, cache, fmt.Errorf("error building message: %w", err))
			continue
		}

		msg.SetContext(ctx)

		if err := e.FlushMessageHandler(msg); err != nil {
			return fmt.Errorf("error flushing messages: %w", err)
		}
	}

	return nil
}

// dropFlushCacheEntry logs a flush_cache row that could not be rebuilt into
// an event and deletes it, so that one bad row does not block server startup.
func (e *EEA) dropFlushCacheEntry(ctx context.Context, cache db.FlushCache, cause error) {
	logger := zerolog.Ctx(ctx).With().
		Str("component", "EEA").
		Str("function", "FlushAll").
		Str("entity", string(cache.Entity)).
		Str("entity_id", cache.EntityInstanceID.String()).
		Str("project_id", cache.ProjectID.String()).
		Logger()

	logger.Error().Err(cause).Msg("unable to rebuild queued event, dropping it from flush cache")

	if _, err := e.querier.FlushCache(ctx, cache.EntityInstanceID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error().Err(err).Msg("error deleting entry from flush cache")
	}
}

func (e *EEA) buildEntityWrapper(
	ctx context.Context,
	entity db.Entities,
	projID uuid.UUID,
	entityID uuid.UUID,
) (*entities.EntityInfoWrapper, error) {
	ent, err := e.entityFetcher.EntityWithPropertiesByID(ctx, entityID, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching entity: %w", err)
	}

	if ent.Entity.ProjectID != projID {
		return nil, fmt.Errorf("entity %s does not belong to project %s", entityID, projID)
	}

	entType := entities.EntityTypeFromDB(entity)
	if ent.Entity.Type != entType {
		return nil, fmt.Errorf("entity %s has type %s, expected %s", entityID, ent.Entity.Type, entType)
	}

	pbEnt, err := e.entityFetcher.EntityWithPropertiesAsProto(ctx, ent, e.provMan)
	if err != nil {
		return nil, fmt.Errorf("error converting entity to protobuf: %w", err)
	}

	return entities.NewEntityInfoWrapper().
		WithProtoMessage(entType, pbEnt).
		WithID(entityID).
		WithProjectID(projID).
		WithProviderID(ent.Entity.ProviderID), nil
}
