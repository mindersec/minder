// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package reminder sends reminders to the minder server to process entities in background.
package reminder

import (
	"context"
	"errors"
	"fmt"
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

	repositoryCursor uuid.UUID

	ticker *time.Ticker

	eventPublisher message.Publisher

	metrics *metrics.Metrics
}

// NewReminder creates a new reminder instance
func NewReminder(ctx context.Context, store db.Store, config *reminderconfig.Config) (Interface, error) {
	r := &reminder{
		store: store,
		cfg:   config,
		stop:  make(chan struct{}),
	}

	// Set to a random UUID to start
	r.repositoryCursor = uuid.New()
	logger := zerolog.Ctx(ctx)
	logger.Info().Msgf("initial repository cursor: %s", r.repositoryCursor)

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
	defer r.Stop()

	interval := r.cfg.RecurrenceConfig.Interval
	if interval <= 0 {
		return fmt.Errorf("invalid interval: %s", r.cfg.RecurrenceConfig.Interval)
	}

	metricsServerDone := make(chan struct{})

	if r.cfg.MetricsConfig.Enabled {
		metricsProviderReady := make(chan struct{})

		go func() {
			if err := r.startMetricServer(ctx, metricsProviderReady); err != nil {
				logger.Fatal().Err(err).Msg("failed to start metrics server")
			}
			close(metricsServerDone)
		}()

		select {
		case <-metricsProviderReady:
			var err error
			r.metrics, err = metrics.NewMetrics(otel.Meter("reminder"))
			if err != nil {
				return err
			}
		case <-ctx.Done():
			logger.Info().Msg("reminder stopped")
			return nil
		}
	}

	r.ticker = time.NewTicker(interval)

	for {
		select {
		case <-ctx.Done():
			if r.cfg.MetricsConfig.Enabled {
				<-metricsServerDone
			}
			logger.Info().Msg("reminder stopped")
			return nil
		case <-r.stop:
			if r.cfg.MetricsConfig.Enabled {
				<-metricsServerDone
			}
			logger.Info().Msg("reminder stopped")
			return nil
		case <-r.ticker.C:
			// In-case sending reminders i.e. iterating over entities consumes more time than the
			// interval, the ticker will adjust the time interval or drop ticks to make up for
			// slow receivers.
			if err := r.sendReminders(ctx); err != nil {
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
}

func (r *reminder) sendReminders(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)

	// Fetch a batch of repositories
	repos, repoToLastUpdated, err := r.getRepositoryBatch(ctx)
	if err != nil {
		return fmt.Errorf("error fetching repository batch: %w", err)
	}

	if len(repos) == 0 {
		logger.Debug().Msg("no repositories to send reminders for")
		return nil
	}

	logger.Info().Msgf("created repository batch of size: %d", len(repos))

	messages, err := createReminderMessages(ctx, repos)
	if err != nil {
		return fmt.Errorf("error creating reminder messages: %w", err)
	}

	if r.metrics != nil {
		r.metrics.RecordBatch(ctx, int64(len(repos)))
	}

	err = r.eventPublisher.Publish(constants.TopicQueueRepoReminder, messages...)
	if err != nil {
		return fmt.Errorf("error publishing messages: %w", err)
	}

	repoIds := make([]uuid.UUID, len(repos))
	for _, repo := range repos {
		repoIds = append(repoIds, repo.ID)
		if r.metrics != nil {
			sendDelay := time.Since(repoToLastUpdated[repo.ID]) - r.cfg.RecurrenceConfig.MinElapsed

			recorder := r.metrics.SendDelay
			if !repo.ReminderLastSent.Valid {
				recorder = r.metrics.NewSendDelay
			}
			recorder.Record(ctx, sendDelay.Seconds())
		}
	}

	err = r.store.UpdateReminderLastSentForRepositories(ctx, repoIds)
	if err != nil {
		return fmt.Errorf("reminders published but error updating last sent time: %w", err)
	}

	return nil
}

func (r *reminder) getRepositoryBatch(ctx context.Context) ([]db.Repository, map[uuid.UUID]time.Time, error) {
	logger := zerolog.Ctx(ctx)

	logger.Debug().Msgf("fetching repositories after cursor: %s", r.repositoryCursor)
	repos, err := r.store.ListRepositoriesAfterID(ctx, db.ListRepositoriesAfterIDParams{
		ID:    r.repositoryCursor,
		Limit: int64(r.cfg.RecurrenceConfig.BatchSize),
	})
	if err != nil {
		return nil, nil, err
	}

	eligibleRepos, eligibleReposLastUpdated, err := r.getEligibleRepositories(ctx, repos)
	if err != nil {
		return nil, nil, err
	}
	logger.Debug().Msgf("%d/%d repositories are eligible for reminders", len(eligibleRepos), len(repos))

	r.updateRepositoryCursor(ctx, repos)

	return eligibleRepos, eligibleReposLastUpdated, nil
}

func (r *reminder) getEligibleRepositories(ctx context.Context, repos []db.Repository) (
	[]db.Repository, map[uuid.UUID]time.Time, error,
) {
	eligibleRepos := make([]db.Repository, 0, len(repos))

	// We have a slice of repositories, but the sqlc-generated code wants a slice of UUIDs,
	// and similarly returns slices of ID -> date (in possibly different order), so we need
	// to do a bunch of mapping here.
	repoIds := make([]uuid.UUID, 0, len(repos))
	for _, repo := range repos {
		repoIds = append(repoIds, repo.ID)
	}
	oldestRuleEvals, err := r.store.ListOldestRuleEvaluationsByRepositoryId(ctx, repoIds)
	if err != nil {
		return nil, nil, err
	}
	idToLastUpdate := make(map[uuid.UUID]time.Time, len(oldestRuleEvals))
	for _, ruleEval := range oldestRuleEvals {
		idToLastUpdate[ruleEval.RepositoryID] = ruleEval.OldestLastUpdated
	}

	cutoff := time.Now().Add(-1 * r.cfg.RecurrenceConfig.MinElapsed)
	for _, repo := range repos {
		if t, ok := idToLastUpdate[repo.ID]; ok && t.Before(cutoff) {
			eligibleRepos = append(eligibleRepos, repo)
		}
	}

	return eligibleRepos, idToLastUpdate, nil
}

func (r *reminder) updateRepositoryCursor(ctx context.Context, repos []db.Repository) {
	logger := zerolog.Ctx(ctx)

	if len(repos) == 0 {
		r.repositoryCursor = uuid.Nil
	} else {
		r.repositoryCursor = repos[len(repos)-1].ID
		r.adjustCursorForEndOfList(ctx)
	}

	logger.Debug().Msgf("updated repository cursor to: %s", r.repositoryCursor)
}

func (r *reminder) adjustCursorForEndOfList(ctx context.Context) {
	logger := zerolog.Ctx(ctx)
	// Check if the cursor is the last element in the db
	exists, err := r.store.RepositoryExistsAfterID(ctx, r.repositoryCursor)
	if err != nil {
		logger.Error().Err(err).Msgf("unable to check if repository exists after cursor: %s"+
			", resetting cursor to zero uuid", r.repositoryCursor)
		r.repositoryCursor = uuid.Nil
		return
	}

	if !exists {
		logger.Info().Msgf("cursor %s is at the end of the list, resetting cursor to zero uuid",
			r.repositoryCursor)
		r.repositoryCursor = uuid.Nil
	}
}

func createReminderMessages(ctx context.Context, repos []db.Repository) ([]*message.Message, error) {
	logger := zerolog.Ctx(ctx)

	messages := make([]*message.Message, 0, len(repos))
	for _, repo := range repos {
		repoReconcileMessage, err := remindermessages.NewEntityReminderMessage(
			repo.ProviderID, repo.ID, repo.ProjectID,
		)
		if err != nil {
			return nil, fmt.Errorf("error creating reminder message: %w", err)
		}

		logger.Debug().
			Str("repo", repo.ID.String()).
			Msg("created reminder message")

		messages = append(messages, repoReconcileMessage)
	}

	return messages, nil
}
