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
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	reminderconfig "github.com/stacklok/minder/internal/config/reminder"
	"github.com/stacklok/minder/internal/db"
)

func init() {
	gob.Register(map[projectProviderPair]string{})
}

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

	projectListCursor string
	repoListCursor    map[projectProviderPair]string

	ticker *time.Ticker

	eventPublisher message.Publisher
	eventDBCloser  driverCloser
}

type projectProviderPair struct {
	// Exported for gob

	ProjectId uuid.UUID
	Provider  string
}

// NewReminder creates a new reminder instance
func NewReminder(ctx context.Context, store db.Store, config *reminderconfig.Config) (Interface, error) {
	logger := zerolog.Ctx(ctx)
	r := &reminder{
		store:          store,
		cfg:            config,
		stop:           make(chan struct{}),
		repoListCursor: make(map[projectProviderPair]string),
	}
	err := r.restoreCursorState(ctx)
	if err != nil {
		// Non-fatal error, if we can't restore the cursor state, we'll start from scratch.
		logger.Error().Err(err).Msg("error restoring cursor state")
	}

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
			if errs := r.sendReminders(ctx); errs != nil {
				for _, err := range errs {
					logger.Error().Err(err).Msg("reconciliation request unsuccessful")
				}
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

// storeCursorState stores the cursor state to a file
// Not thread-safe, should be called from a single goroutine
func (r *reminder) storeCursorState(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msg("storing cursor state")

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	data := map[string]interface{}{
		"projectListCursor": r.projectListCursor,
		"repoListCursor":    r.repoListCursor,
	}
	if err := enc.Encode(data); err != nil {
		return err
	}

	return os.WriteFile(r.cfg.CursorFile, buf.Bytes(), 0600)
}

// restoreCursorState restores the cursor state from a file
// Not thread-safe, should be called from a single goroutine
func (r *reminder) restoreCursorState(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	logger.Debug().Msg("restoring cursor state")

	if _, err := os.Stat(r.cfg.CursorFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(r.cfg.CursorFile)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)

	cursorData := make(map[string]interface{})

	if err := dec.Decode(&cursorData); err != nil {
		return err
	}

	if val, ok := cursorData["projectListCursor"]; ok {
		if v, ok := val.(string); ok {
			r.projectListCursor = v
		} else {
			return fmt.Errorf("projectListCursor is not a string")
		}
	}

	if val, ok := cursorData["repoListCursor"]; ok {
		if v, ok := val.(map[projectProviderPair]string); ok {
			r.repoListCursor = v
		} else {
			return fmt.Errorf("repoListCursor is not a map[projectProviderPair]string")
		}
	}

	return nil
}

// TODO: Will be implemented in a separate PR
func (_ *reminder) sendReminders(_ context.Context) []error {
	return nil
}
