// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	reminderconfig "github.com/stacklok/minder/internal/config/reminder"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/events/common"
	remindermessages "github.com/stacklok/minder/internal/reminder/messages"
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
	eventDBCloser  common.DriverCloser
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

	pub, cl, err := r.setupSQLPublisher(ctx)
	if err != nil {
		return nil, err
	}

	r.eventPublisher = pub
	r.eventDBCloser = cl
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

	interval := r.cfg.RecurrenceConfig.Interval
	if interval <= 0 {
		return fmt.Errorf("invalid interval: %s", r.cfg.RecurrenceConfig.Interval)
	}

	r.ticker = time.NewTicker(interval)
	defer r.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("reminder stopped")
			return nil
		case <-r.stop:
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
		r.eventDBCloser()
	})
}

func (r *reminder) sendReminders(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)

	// Fetch a batch of repositories
	repos, err := r.getRepositoryBatch(ctx)
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

	err = r.eventPublisher.Publish(events.TopicQueueRepoReminder, messages...)
	if err != nil {
		return fmt.Errorf("error publishing messages: %w", err)
	}

	repoIds := make([]uuid.UUID, len(repos))
	for _, repo := range repos {
		repoIds = append(repoIds, repo.ID)
	}

	// TODO: Collect Metrics
	// Potential metrics:
	// - Gauge: Number of reminders in the current batch
	// - UpDownCounter: Average reminders sent per batch
	// - Histogram: reminder_last_sent time distribution

	err = r.store.UpdateReminderLastSentForRepositories(ctx, repoIds)
	if err != nil {
		return fmt.Errorf("reminders published but error updating last sent time: %w", err)
	}

	return nil
}

func (r *reminder) getRepositoryBatch(ctx context.Context) ([]db.Repository, error) {
	logger := zerolog.Ctx(ctx)

	logger.Debug().Msgf("fetching repositories after cursor: %s", r.repositoryCursor)
	repos, err := r.store.ListRepositoriesAfterID(ctx, db.ListRepositoriesAfterIDParams{
		ID:    r.repositoryCursor,
		Limit: int64(r.cfg.RecurrenceConfig.BatchSize),
	})
	if err != nil {
		return nil, err
	}

	eligibleRepos, err := r.getEligibleRepositories(ctx, repos)
	if err != nil {
		return nil, err
	}
	logger.Debug().Msgf("%d/%d repositories are eligible for reminders", len(eligibleRepos), len(repos))

	r.updateRepositoryCursor(ctx, repos)

	return eligibleRepos, nil
}

func (r *reminder) getEligibleRepositories(ctx context.Context, repos []db.Repository) ([]db.Repository, error) {
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
		return nil, err
	}
	idToLastUpdate := make(map[uuid.UUID]time.Time, len(oldestRuleEvals))
	for _, times := range oldestRuleEvals {
		idToLastUpdate[times.RepositoryID] = times.OldestLastUpdated
	}

	cutoff := time.Now().Add(-1 * r.cfg.RecurrenceConfig.MinElapsed)
	for _, repo := range repos {
		if t, ok := idToLastUpdate[repo.ID]; ok && t.Before(cutoff) {
			eligibleRepos = append(eligibleRepos, repo)
		}
	}

	return eligibleRepos, nil
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
		repoReconcileMessage, err := remindermessages.NewRepoReminderMessage(
			repo.ProviderID, repo.RepoID, repo.ProjectID,
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
