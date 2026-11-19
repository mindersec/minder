// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package reminder sends reminders to the minder server to process entities in background.
package reminder

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"

	"github.com/mindersec/minder/internal/db"
	remindermessages "github.com/mindersec/minder/internal/reminder/messages"
	"github.com/mindersec/minder/internal/reminder/metrics"
	reminderconfig "github.com/mindersec/minder/pkg/config/reminder"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

// Interface is an interface over the reminder service
type Interface interface {
	// Start starts the reminder by sending reminders at regular intervals
	Start(ctx context.Context) error

	// Stop stops the reminder service
	Stop()
}

// reminder sends reminders to the minder server to process entities in background.
type reminder struct {
	store    db.Store
	cfg      *reminderconfig.Config
	stop     chan struct{}
	stopOnce sync.Once

	entityTypes []db.Entities
	cursors     map[db.Entities]uuid.UUID
	nextDue     map[db.Entities]time.Time

	ticker *time.Ticker

	eventPublisher message.Publisher

	metrics           *metrics.Metrics
	metricsServerDone chan struct{}
}

// supportedEntityTypes are the entity types reminders can be sent for. Each
// one is sent only when RecurrenceConfig.ForEntityType reports it enabled.
var supportedEntityTypes = []db.Entities{db.EntitiesRepository, db.EntitiesArtifact}

func enabledEntityTypes(cfg reminderconfig.RecurrenceConfig) []db.Entities {
	enabled := make([]db.Entities, 0, len(supportedEntityTypes))
	for _, entityType := range supportedEntityTypes {
		if cfg.ForEntityType(entityType).Enabled {
			enabled = append(enabled, entityType)
		}
	}
	return enabled
}

// NewReminder creates a new reminder instance
func NewReminder(ctx context.Context, store db.Store, config *reminderconfig.Config) (Interface, error) {
	r := &reminder{
		store:             store,
		cfg:               config,
		stop:              make(chan struct{}),
		metricsServerDone: make(chan struct{}),
		entityTypes:       enabledEntityTypes(config.RecurrenceConfig),
		cursors:           make(map[db.Entities]uuid.UUID, len(supportedEntityTypes)),
		nextDue:           make(map[db.Entities]time.Time, len(supportedEntityTypes)),
	}

	logger := zerolog.Ctx(ctx)
	for entityType := range config.RecurrenceConfig.Entities {
		if config.RecurrenceConfig.ForEntityType(entityType).Enabled && !slices.Contains(supportedEntityTypes, entityType) {
			logger.Warn().Str("entity_type", string(entityType)).
				Msg("reminders are not supported for this entity type, ignoring")
		}
	}

	// Set to a random UUID to start
	for _, entityType := range r.entityTypes {
		r.cursors[entityType] = uuid.New()
		logger.Info().Str("entity_type", string(entityType)).
			Msgf("initial cursor: %s", r.cursors[entityType])
	}

	pub, err := r.getMessagePublisher(ctx)
	if err != nil {
		return nil, err
	}

	r.eventPublisher = pub
	return r, nil
}

// Start starts the reminder by sending reminders at regular intervals
func (r *reminder) Start(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	select {
	case <-r.stop:
		return errors.New("reminder stopped, cannot start again")
	default:
	}
	// Reminder only stops in case of error or context cancellation (or when Stop() is called)
	// An errored out reminder cannot be started again, so it is stopped here
	// This also prevents resource leaks if user doesn't explicitly stop the reminder
	defer r.Stop()

	interval, err := r.tickInterval()
	if err != nil {
		return err
	}

	if r.cfg.MetricsConfig.Enabled {
		if err := r.startMetricServer(ctx); err != nil {
			logger.Err(err).Msg("failed to start metrics server")
		}

		var err error
		r.metrics, err = metrics.NewMetrics(otel.Meter("reminder"))
		if err != nil {
			return err
		}
	} else {
		close(r.metricsServerDone)
	}

	logger.Info().Msgf("reminder ticking every %s for entity types %v", interval, r.entityTypes)
	r.ticker = time.NewTicker(interval)

	for {
		select {
		case <-ctx.Done():
			<-r.metricsServerDone
			logger.Info().Msg("reminder stopped")
			return nil
		case <-r.stop:
			<-r.metricsServerDone
			logger.Info().Msg("reminder stopped")
			return nil
		case now := <-r.ticker.C:
			// In-case sending reminders i.e. iterating over entities consumes more time than the
			// interval, the ticker will adjust the time interval or drop ticks to make up for
			// slow receivers.
			if err := r.sendReminders(ctx, now); err != nil {
				logger.Error().Err(err).Msg("reconciliation request unsuccessful")
			}
		}
	}
}

// Stop stops the reminder service
// Stopping the reminder service closes the stop channel and stops the ticker (if not nil).
// It also closes the event publisher database connection which means that only reminders
// that were sent successfully will be processed. Any reminders that were not sent will be lost.
// Stopping the reminder service while the service is starting up may cause the ticker to not be
// stopped as ticker might not have been created yet. Ticker will be stopped after Start returns
// as defer statement in Start will stop the ticker.
func (r *reminder) Stop() {
	if r.ticker != nil {
		defer r.ticker.Stop()
	}
	r.stopOnce.Do(func() {
		close(r.stop)
		err := r.eventPublisher.Close()
		if err != nil {
			zerolog.Ctx(context.Background()).Error().Err(err).Msg("error closing event publisher")
		}
	})
	// Wait for the metrics server to stop
	<-r.metricsServerDone
}

// tickInterval returns the shortest interval across the enabled entity types.
func (r *reminder) tickInterval() (time.Duration, error) {
	if len(r.entityTypes) == 0 {
		return 0, errors.New("no entity types are enabled for reminders")
	}

	var shortest time.Duration
	for _, entityType := range r.entityTypes {
		interval := r.cfg.RecurrenceConfig.ForEntityType(entityType).Interval
		if interval <= 0 {
			return 0, fmt.Errorf("invalid interval for %s: %s", entityType, interval)
		}
		if shortest == 0 || interval < shortest {
			shortest = interval
		}
	}
	return shortest, nil
}

// A type is due once its next due time has passed. The small slack absorbs
// late tick delivery without letting a type run early by most of a tick.
func (r *reminder) isDue(entityType db.Entities, now time.Time, tick time.Duration) bool {
	return !now.Add(tick / 10).Before(r.nextDue[entityType])
}

func (r *reminder) sendReminders(ctx context.Context, now time.Time) error {
	tick, err := r.tickInterval()
	if err != nil {
		return err
	}

	var errs []error
	for _, entityType := range r.entityTypes {
		if !r.isDue(entityType, now, tick) {
			zerolog.Ctx(ctx).Debug().Str("entity_type", string(entityType)).
				Msgf("not due until %s, skipping", r.nextDue[entityType])
			continue
		}
		r.nextDue[entityType] = now.Add(r.cfg.RecurrenceConfig.ForEntityType(entityType).Interval)

		if err := r.sendRemindersForEntityType(ctx, entityType); err != nil {
			errs = append(errs, fmt.Errorf("error sending %s reminders: %w", entityType, err))
		}
	}

	return errors.Join(errs...)
}

func (r *reminder) sendRemindersForEntityType(ctx context.Context, entityType db.Entities) error {
	logger := zerolog.Ctx(ctx).With().Str("entity_type", string(entityType)).Logger()
	minElapsed := r.cfg.RecurrenceConfig.ForEntityType(entityType).MinElapsed
	entityTypeAttr := metrics.EntityTypeAttribute(string(entityType))

	entities, entityToLastUpdated, err := r.getEntityBatch(ctx, entityType)
	if err != nil {
		return fmt.Errorf("error fetching entity batch: %w", err)
	}

	if len(entities) == 0 {
		logger.Debug().Msg("no entities to send reminders for")
		return nil
	}

	logger.Info().Msgf("created entity batch of size: %d", len(entities))

	messages, err := createReminderMessages(ctx, entities)
	if err != nil {
		return fmt.Errorf("error creating reminder messages: %w", err)
	}

	if r.metrics != nil {
		r.metrics.BatchSize.Record(ctx, int64(len(entities)), entityTypeAttr)
	}

	err = r.eventPublisher.Publish(constants.TopicQueueRepoReminder, messages...)
	if err != nil {
		return fmt.Errorf("error publishing messages: %w", err)
	}

	for _, entity := range entities {
		if r.metrics != nil {
			sendDelay := time.Since(entityToLastUpdated[entity.ID]) - minElapsed
			// TODO: Track whether this is a new vs existing reminder
			// Previously used repo.ReminderLastSent field which is now removed
			recorder := r.metrics.SendDelay
			recorder.Record(ctx, sendDelay.Seconds(), entityTypeAttr)
		}
	}

	// Note: The legacy reminder_last_sent timestamp tracking has been removed.
	// We rely solely on the MinElapsed check against evaluation history for throttling.
	// This provides sufficient rate limiting while keeping the reminder service stateless
	// and horizontally scalable.

	return nil
}

func (r *reminder) getEntityBatch(
	ctx context.Context, entityType db.Entities,
) ([]db.EntityInstance, map[uuid.UUID]time.Time, error) {
	logger := zerolog.Ctx(ctx).With().Str("entity_type", string(entityType)).Logger()

	logger.Debug().Msgf("fetching entities after cursor: %s", r.cursors[entityType])

	recurrence := r.cfg.RecurrenceConfig.ForEntityType(entityType)
	entities, err := r.store.ListEntitiesAfterID(ctx, db.ListEntitiesAfterIDParams{
		EntityType: entityType,
		ID:         r.cursors[entityType],
		Limit:      int64(recurrence.BatchSize),
	})
	if err != nil {
		return nil, nil, err
	}

	eligibleEntities, eligibleEntitiesLastUpdated, err := r.getEligibleEntities(ctx, entities, recurrence.MinElapsed)
	if err != nil {
		return nil, nil, err
	}
	logger.Debug().Msgf("%d/%d entities are eligible for reminders", len(eligibleEntities), len(entities))

	r.updateCursor(ctx, entityType, entities)

	return eligibleEntities, eligibleEntitiesLastUpdated, nil
}

func (r *reminder) getEligibleEntities(ctx context.Context, entities []db.EntityInstance, minElapsed time.Duration) (
	[]db.EntityInstance, map[uuid.UUID]time.Time, error,
) {
	eligibleEntities := make([]db.EntityInstance, 0, len(entities))

	// We have a slice of entity instances, extract UUIDs for evaluation lookup
	entityIds := make([]uuid.UUID, 0, len(entities))
	for _, entity := range entities {
		entityIds = append(entityIds, entity.ID)
	}

	// Use entity_instance_id based query instead of repository_id
	oldestRuleEvals, err := r.store.ListOldestRuleEvaluationsByEntityID(ctx, entityIds)
	if err != nil {
		return nil, nil, err
	}
	idToLastUpdate := make(map[uuid.UUID]time.Time, len(oldestRuleEvals))
	for _, ruleEval := range oldestRuleEvals {
		idToLastUpdate[ruleEval.EntityInstanceID] = ruleEval.OldestLastUpdated
	}

	cutoff := time.Now().Add(-1 * minElapsed)
	for _, entity := range entities {
		if t, ok := idToLastUpdate[entity.ID]; ok && t.Before(cutoff) {
			eligibleEntities = append(eligibleEntities, entity)
		}
	}

	return eligibleEntities, idToLastUpdate, nil
}

func (r *reminder) updateCursor(ctx context.Context, entityType db.Entities, entities []db.EntityInstance) {
	logger := zerolog.Ctx(ctx).With().Str("entity_type", string(entityType)).Logger()

	if len(entities) == 0 {
		r.cursors[entityType] = uuid.Nil
	} else {
		r.cursors[entityType] = entities[len(entities)-1].ID
		r.adjustCursorForEndOfList(ctx, entityType)
	}

	logger.Debug().Msgf("updated cursor to: %s", r.cursors[entityType])
}

func (r *reminder) adjustCursorForEndOfList(ctx context.Context, entityType db.Entities) {
	logger := zerolog.Ctx(ctx).With().Str("entity_type", string(entityType)).Logger()

	cursor := r.cursors[entityType]
	exists, err := r.store.EntityExistsAfterID(ctx, db.EntityExistsAfterIDParams{
		EntityType: entityType,
		ID:         cursor,
	})
	if err != nil {
		logger.Error().Err(err).Msgf("unable to check if entity exists after cursor: %s"+
			", resetting cursor to zero uuid", cursor)
		r.cursors[entityType] = uuid.Nil
		return
	}

	if !exists {
		logger.Info().Msgf("cursor %s is at the end of the list, resetting cursor to zero uuid", cursor)
		r.cursors[entityType] = uuid.Nil
	}
}

func createReminderMessages(ctx context.Context, entities []db.EntityInstance) ([]*message.Message, error) {
	logger := zerolog.Ctx(ctx)

	messages := make([]*message.Message, 0, len(entities))
	for _, entity := range entities {
		reminderMsg, err := remindermessages.NewEntityReminderMessage(
			entity.ProviderID, entity.ID, entity.ProjectID,
		)
		if err != nil {
			return nil, fmt.Errorf("error creating reminder message: %w", err)
		}

		logger.Debug().
			Str("entity_id", entity.ID.String()).
			Msg("created reminder message")

		messages = append(messages, reminderMsg)
	}

	return messages, nil
}
