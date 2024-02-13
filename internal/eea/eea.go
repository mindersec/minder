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
// Package rule provides the CLI subcommand for managing rules

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

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/util"
)

// EEA is the Event Execution Aggregator
type EEA struct {
	querier db.Store
	evt     *events.Eventer
	cfg     *serverconfig.AggregatorConfig
}

// NewEEA creates a new EEA
func NewEEA(querier db.Store, evt *events.Eventer, cfg *serverconfig.AggregatorConfig) *EEA {
	return &EEA{
		querier: querier,
		evt:     evt,
		cfg:     cfg,
	}
}

// Register implements the Consumer interface.
func (e *EEA) Register(r events.Registrar) {
	r.Register(events.FlushEntityEventTopic, e.FlushMessageHandler)
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

	repoID, artifactID, pullRequestID := inf.GetEntityDBIDs()

	logger := zerolog.Ctx(ctx).Info()
	logger = logger.Str("event", msg.UUID).
		Str("entity", inf.Type.ToString()).
		Str("repository_id", repoID.String())

	// We need to check that the resources still exist before attempting to lock them.
	// TODO: consider whether we need foreign key checks on the locks.
	if _, err := e.querier.GetRepositoryByID(ctx, repoID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			logger.Msg("Skipping event because repository no longer exists")
			return nil, nil
		}
	}
	if artifactID.Valid {
		if _, err := e.querier.GetArtifactByID(ctx, artifactID.UUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.Msg("Skipping event because artifact no longer exists")
				return nil, nil
			}
		}
	}
	if pullRequestID.Valid {
		if _, err := e.querier.GetPullRequestByID(ctx, pullRequestID.UUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				logger.Msg("Skipping event because pull request no longer exists")
				return nil, nil
			}
		}
	}

	res, err := e.querier.LockIfThresholdNotExceeded(ctx, db.LockIfThresholdNotExceededParams{
		Entity:        entities.EntityTypeToDB(inf.Type),
		RepositoryID:  repoID,
		ArtifactID:    artifactID,
		PullRequestID: pullRequestID,
		Interval:      fmt.Sprintf("%d", e.cfg.LockInterval),
	})

	if artifactID.Valid {
		logger = logger.Str("artifact_id", artifactID.UUID.String())
	}

	if pullRequestID.Valid {
		logger = logger.Str("pull_request_id", pullRequestID.UUID.String())
	}

	// if nothing was retrieved from the database, then we can assume
	// that the event is not ready to be executed.
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		logger.Msg("event not ready to be executed")

		_, err := e.querier.EnqueueFlush(ctx, db.EnqueueFlushParams{
			Entity:        entities.EntityTypeToDB(inf.Type),
			RepositoryID:  repoID,
			ArtifactID:    artifactID,
			PullRequestID: pullRequestID,
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

	logger.Str("execution_id", res.LockedBy.String()).Msg("event ready to be executed")
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

	repoID, artifactID, pullRequestID := inf.GetEntityDBIDs()

	zerolog.Ctx(ctx).Info().
		Str("event", msg.UUID).
		Str("entity", inf.Type.ToString()).
		Str("repository_id", repoID.String()).Msg("flushing event")

	_, err = e.querier.FlushCache(ctx, db.FlushCacheParams{
		Entity:        entities.EntityTypeToDB(inf.Type),
		RepositoryID:  repoID,
		ArtifactID:    artifactID,
		PullRequestID: pullRequestID,
	})
	// Nothing to do here. If we can't flush the cache, it means
	// that the event has already been executed.
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		zerolog.Ctx(ctx).Info().
			Str("event", msg.UUID).
			Str("entity", inf.Type.ToString()).
			Str("repository_id", repoID.String()).Msg("no flushing needed")
		return nil
	} else if err != nil {
		return fmt.Errorf("error flushing cache: %w", err)
	}

	zerolog.Ctx(ctx).Info().
		Str("event", msg.UUID).
		Str("entity", inf.Type.ToString()).
		Str("repository_id", repoID.String()).Msg("re-publishing event because of flush")

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
		// No rows to flush, this is fine.
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("error listing flush cache: %w", err)
	}

	for _, cache := range caches {
		cache := cache

		// ensure that the eiw has a project ID (invariant checked elsewhere)
		r, err := e.querier.GetRepositoryByID(ctx, cache.RepositoryID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				zerolog.Ctx(ctx).Info().Msg("No project found for repository, skipping")
				continue
			}
			return fmt.Errorf("unable to look up project for repository %s: %w", cache.RepositoryID, err)
		}

		eiw, err := e.buildEntityWrapper(ctx, cache.Entity,
			cache.RepositoryID, r.ProjectID, r.Provider, cache.ArtifactID, cache.PullRequestID)
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			continue
		} else if err != nil {
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
	repoID uuid.UUID,
	projID uuid.UUID,
	provider string,
	artID, prID uuid.NullUUID,
) (*entities.EntityInfoWrapper, error) {
	switch entity {
	case db.EntitiesRepository:
		return e.buildRepositoryInfoWrapper(ctx, repoID, projID, provider)
	case db.EntitiesArtifact:
		return e.buildArtifactInfoWrapper(ctx, repoID, projID, provider, artID)
	case db.EntitiesPullRequest:
		return e.buildPullRequestInfoWrapper(ctx, repoID, projID, provider, prID)
	case db.EntitiesBuildEnvironment:
		return nil, fmt.Errorf("build environment entity not supported")
	default:
		return nil, fmt.Errorf("unknown entity type: %s", entity)
	}
}

func (e *EEA) buildRepositoryInfoWrapper(
	ctx context.Context,
	repoID uuid.UUID,
	projID uuid.UUID,
	provider string,
) (*entities.EntityInfoWrapper, error) {
	r, err := util.GetRepository(ctx, e.querier, repoID)
	if err != nil {
		return nil, fmt.Errorf("error getting repository: %w", err)
	}

	return entities.NewEntityInfoWrapper().
		WithRepository(r).
		WithRepositoryID(repoID).
		WithProjectID(projID).
		WithProvider(provider), nil
}

func (e *EEA) buildArtifactInfoWrapper(
	ctx context.Context,
	repoID uuid.UUID,
	projID uuid.UUID,
	provider string,
	artID uuid.NullUUID,
) (*entities.EntityInfoWrapper, error) {
	a, err := util.GetArtifact(ctx, e.querier, repoID, artID.UUID)
	if err != nil {
		return nil, fmt.Errorf("error getting artifact with versions: %w", err)
	}

	return entities.NewEntityInfoWrapper().
		WithRepositoryID(repoID).
		WithProjectID(projID).
		WithArtifact(a).
		WithArtifactID(artID.UUID).
		WithProvider(provider), nil
}

func (e *EEA) buildPullRequestInfoWrapper(
	ctx context.Context,
	repoID uuid.UUID,
	projID uuid.UUID,
	provider string,
	prID uuid.NullUUID,
) (*entities.EntityInfoWrapper, error) {
	pr, err := util.GetPullRequest(ctx, e.querier, repoID, prID.UUID)
	if err != nil {
		return nil, fmt.Errorf("error getting pull request: %w", err)
	}

	return entities.NewEntityInfoWrapper().
		WithRepositoryID(repoID).
		WithProjectID(projID).
		WithPullRequest(pr).
		WithPullRequestID(prID.UUID).
		WithProvider(provider), nil
}
