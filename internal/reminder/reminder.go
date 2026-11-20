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
	"go.opentelemetry.io/otel/metric"

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

// reminder holds the resources shared by every entity type and runs one
// entityReminder per enabled entity type, each on its own ticker.
type reminder struct {
	store db.Store
	cfg   *reminderconfig.Config

	mu        sync.Mutex
	stop      chan struct{}
	stopOnce  sync.Once
	closeOnce sync.Once
	wg        sync.WaitGroup

	entityReminders []*entityReminder
	newTicker       func(time.Duration) (<-chan time.Time, func())

	eventPublisher message.Publisher

	metrics           *metrics.Metrics
	metricsServerDone chan struct{}
}

// entityReminder revisits the entities of a single type. Its cursor is only
// touched by the goroutine running it, so it needs no locking.
type entityReminder struct {
	*reminder
	entityType     db.Entities
	recurrence     reminderconfig.EntityRecurrence
	entityTypeAttr metric.MeasurementOption
	cursor         uuid.UUID
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

func realTicker(d time.Duration) (<-chan time.Time, func()) {
	t := time.NewTicker(d)
	return t.C, t.Stop
}

func (r *reminder) newEntityReminders(entityTypes []db.Entities) ([]*entityReminder, error) {
	if len(entityTypes) == 0 {
		return nil, errors.New("no entity types are enabled for reminders")
	}

	ers := make([]*entityReminder, 0, len(entityTypes))
	for _, entityType := range entityTypes {
		recurrence := r.cfg.RecurrenceConfig.ForEntityType(entityType)
		if recurrence.Interval <= 0 {
			return nil, fmt.Errorf("invalid interval for %s: %s", entityType, recurrence.Interval)
		}
		ers = append(ers, &entityReminder{
			reminder:       r,
			entityType:     entityType,
			recurrence:     recurrence,
			entityTypeAttr: metrics.EntityTypeAttribute(string(entityType)),
			// Start from a random UUID so restarts don't always begin at the same entity
			cursor: uuid.New(),
		})
	}
	return ers, nil
}

// NewReminder creates a new reminder instance
func NewReminder(ctx context.Context, store db.Store, config *reminderconfig.Config) (Interface, error) {
	r := &reminder{
		store:             store,
		cfg:               config,
		stop:              make(chan struct{}),
		metricsServerDone: make(chan struct{}),
		newTicker:         realTicker,
	}

	logger := zerolog.Ctx(ctx)
	for entityType := range config.RecurrenceConfig.Entities {
		if config.RecurrenceConfig.ForEntityType(entityType).Enabled && !slices.Contains(supportedEntityTypes, entityType) {
			logger.Warn().Str("entity_type", string(entityType)).
				Msg("reminders are not supported for this entity type, ignoring")
		}
	}

	ers, err := r.newEntityReminders(enabledEntityTypes(config.RecurrenceConfig))
	if err != nil {
		return nil, err
	}
	r.entityReminders = ers
	for _, er := range ers {
		logger.Info().Str("entity_type", string(er.entityType)).
			Msgf("initial cursor: %s", er.cursor)
	}

	pub, err := r.getMessagePublisher(ctx)
	if err != nil {
		return nil, err
	}

	r.eventPublisher = pub
	return r, nil
}

// Start runs one goroutine per enabled entity type, each sending reminders at
// that type's own interval, and blocks until ctx is cancelled or Stop is called.
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

	if err := r.startEntityReminders(ctx); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
	case <-r.stop:
	}
	logger.Info().Msg("reminder stopped")
	return nil
}

func (r *reminder) startEntityReminders(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	select {
	case <-r.stop:
		return errors.New("reminder stopped while starting")
	default:
	}

	for _, er := range r.entityReminders {
		ticks, stopTicker := r.newTicker(er.recurrence.Interval)
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			defer stopTicker()
			er.run(ctx, ticks)
		}()
	}
	return nil
}

// Stop stops the reminder service and blocks until every entity type's
// goroutine has exited. The event publisher is closed only after that, so a
// batch that is mid-publish when Stop is called is still sent.
func (r *reminder) Stop() {
	r.mu.Lock()
	r.stopOnce.Do(func() {
		close(r.stop)
	})
	r.mu.Unlock()

	r.wg.Wait()

	r.closeOnce.Do(func() {
		if err := r.eventPublisher.Close(); err != nil {
			zerolog.Ctx(context.Background()).Error().Err(err).Msg("error closing event publisher")
		}
	})
	// Wait for the metrics server to stop
	<-r.metricsServerDone
}

func (e *entityReminder) run(ctx context.Context, ticks <-chan time.Time) {
	logger := zerolog.Ctx(ctx).With().Str("entity_type", string(e.entityType)).Logger()
	logger.Info().Msgf("sending reminders every %s", e.recurrence.Interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stop:
			return
		case <-ticks:
			// If a batch takes longer than the interval, the ticker drops ticks
			// rather than queueing them, so this type never runs back to back.
			if err := e.sendReminders(ctx); err != nil {
				logger.Error().Err(err).Msg("reconciliation request unsuccessful")
			}
		}
	}
}

func (e *entityReminder) sendReminders(ctx context.Context) error {
	ctx = zerolog.Ctx(ctx).With().Str("entity_type", string(e.entityType)).Logger().WithContext(ctx)
	logger := zerolog.Ctx(ctx)

	entities, entityToLastUpdated, err := e.getEntityBatch(ctx)
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

	if e.metrics != nil {
		e.metrics.BatchSize.Record(ctx, int64(len(entities)), e.entityTypeAttr)
	}

	err = e.eventPublisher.Publish(constants.TopicQueueRepoReminder, messages...)
	if err != nil {
		return fmt.Errorf("error publishing messages: %w", err)
	}

	for _, entity := range entities {
		if e.metrics != nil {
			sendDelay := time.Since(entityToLastUpdated[entity.ID]) - e.recurrence.MinElapsed
			// TODO: Track whether this is a new vs existing reminder
			// Previously used repo.ReminderLastSent field which is now removed
			e.metrics.SendDelay.Record(ctx, sendDelay.Seconds(), e.entityTypeAttr)
		}
	}

	// Note: The legacy reminder_last_sent timestamp tracking has been removed.
	// We rely solely on the MinElapsed check against evaluation history for throttling.
	// This provides sufficient rate limiting while keeping the reminder service stateless
	// and horizontally scalable.

	return nil
}

func (e *entityReminder) getEntityBatch(ctx context.Context) ([]db.EntityInstance, map[uuid.UUID]time.Time, error) {
	logger := zerolog.Ctx(ctx)

	logger.Debug().Msgf("fetching entities after cursor: %s", e.cursor)

	entities, err := e.store.ListEntitiesAfterID(ctx, db.ListEntitiesAfterIDParams{
		EntityType: e.entityType,
		ID:         e.cursor,
		Limit:      int64(e.recurrence.BatchSize),
	})
	if err != nil {
		return nil, nil, err
	}

	eligibleEntities, eligibleEntitiesLastUpdated, err := e.getEligibleEntities(ctx, entities, e.recurrence.MinElapsed)
	if err != nil {
		return nil, nil, err
	}
	logger.Debug().Msgf("%d/%d entities are eligible for reminders", len(eligibleEntities), len(entities))

	e.updateCursor(ctx, entities)

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

func (e *entityReminder) updateCursor(ctx context.Context, entities []db.EntityInstance) {
	if len(entities) == 0 {
		e.cursor = uuid.Nil
	} else {
		e.cursor = entities[len(entities)-1].ID
		e.adjustCursorForEndOfList(ctx)
	}

	zerolog.Ctx(ctx).Debug().Msgf("updated cursor to: %s", e.cursor)
}

func (e *entityReminder) adjustCursorForEndOfList(ctx context.Context) {
	logger := zerolog.Ctx(ctx)

	exists, err := e.store.EntityExistsAfterID(ctx, db.EntityExistsAfterIDParams{
		EntityType: e.entityType,
		ID:         e.cursor,
	})
	if err != nil {
		logger.Error().Err(err).Msgf("unable to check if entity exists after cursor: %s"+
			", resetting cursor to zero uuid", e.cursor)
		e.cursor = uuid.Nil
		return
	}

	if !exists {
		logger.Info().Msgf("cursor %s is at the end of the list, resetting cursor to zero uuid", e.cursor)
		e.cursor = uuid.Nil
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
