// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminder

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	reminderconfig "github.com/mindersec/minder/pkg/config/reminder"
)

const waitFor = 5 * time.Second

type fakeTicker struct {
	interval time.Duration
	ch       chan time.Time
	stopped  atomic.Bool
}

type fakeClock struct {
	mu      sync.Mutex
	tickers []*fakeTicker
}

func (c *fakeClock) newTicker(d time.Duration) (<-chan time.Time, func()) {
	ft := &fakeTicker{interval: d, ch: make(chan time.Time)}
	c.mu.Lock()
	c.tickers = append(c.tickers, ft)
	c.mu.Unlock()
	return ft.ch, func() { ft.stopped.Store(true) }
}

func (c *fakeClock) waitForTickers(t *testing.T, n int) []*fakeTicker {
	t.Helper()
	require.Eventually(t, func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		return len(c.tickers) == n
	}, waitFor, time.Millisecond)
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]*fakeTicker(nil), c.tickers...)
}

func tickerFor(t *testing.T, tickers []*fakeTicker, d time.Duration) *fakeTicker {
	t.Helper()
	for _, ft := range tickers {
		if ft.interval == d {
			return ft
		}
	}
	t.Fatalf("no ticker with interval %s", d)
	return nil
}

func (ft *fakeTicker) fire(t *testing.T, at time.Time) {
	t.Helper()
	select {
	case ft.ch <- at:
	case <-time.After(waitFor):
		t.Fatalf("tick for %s interval was not received", ft.interval)
	}
}

func allStopped(tickers []*fakeTicker) bool {
	for _, ft := range tickers {
		if !ft.stopped.Load() {
			return false
		}
	}
	return true
}

type callCounter struct {
	mu    sync.Mutex
	calls map[db.Entities]int
}

func (c *callCounter) inc(entityType db.Entities) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.calls == nil {
		c.calls = map[db.Entities]int{}
	}
	c.calls[entityType]++
}

func (c *callCounter) get(entityType db.Entities) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[entityType]
}

func startReminder(ctx context.Context, r *reminder) <-chan error {
	done := make(chan error, 1)
	go func() { done <- r.Start(ctx) }()
	return done
}

func waitStart(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(waitFor):
		t.Fatal("Start did not return")
		return nil
	}
}

func TestStart_eachEntityTypeRunsOnItsOwnInterval(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	counter := &callCounter{}
	store.EXPECT().ListEntitiesAfterID(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.ListEntitiesAfterIDParams) ([]db.EntityInstance, error) {
			counter.inc(arg.EntityType)
			return nil, nil
		}).AnyTimes()
	store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	r := createTestReminder(t, store, &reminderconfig.Config{
		RecurrenceConfig: reminderconfig.RecurrenceConfig{
			Interval: 5 * time.Minute, BatchSize: 10, MinElapsed: time.Hour,
			Entities: map[db.Entities]reminderconfig.EntityRecurrenceConfig{
				db.EntitiesArtifact:    {Enabled: ptr(true), Interval: ptr(6 * time.Minute)},
				db.EntitiesPullRequest: {Enabled: ptr(true), Interval: ptr(31 * time.Minute)},
			},
		},
	})
	ers, err := r.newEntityReminders([]db.Entities{db.EntitiesRepository, db.EntitiesArtifact, db.EntitiesPullRequest})
	require.NoError(t, err)
	r.entityReminders = ers
	clock := &fakeClock{}
	r.newTicker = clock.newTicker

	done := startReminder(context.Background(), r)
	tickers := clock.waitForTickers(t, 3)

	intervals := map[db.Entities]time.Duration{
		db.EntitiesRepository:  5 * time.Minute,
		db.EntitiesArtifact:    6 * time.Minute,
		db.EntitiesPullRequest: 31 * time.Minute,
	}
	start := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	for elapsed := time.Minute; elapsed <= 62*time.Minute; elapsed += time.Minute {
		for _, ft := range tickers {
			if elapsed%ft.interval == 0 {
				ft.fire(t, start.Add(elapsed))
			}
		}
		for entityType, interval := range intervals {
			want := int(elapsed / interval)
			require.Eventually(t, func() bool { return counter.get(entityType) == want }, waitFor, time.Millisecond,
				"%s at %s: want %d runs, got %d", entityType, elapsed, want, counter.get(entityType))
		}
	}

	// A shared 5m tick would have run the 6m type every 10m (6 runs) and the
	// 31m type every 35m.
	require.Equal(t, 12, counter.get(db.EntitiesRepository))
	require.Equal(t, 10, counter.get(db.EntitiesArtifact))
	require.Equal(t, 2, counter.get(db.EntitiesPullRequest))

	r.Stop()
	require.NoError(t, waitStart(t, done))
	require.True(t, allStopped(tickers))
}

func TestStop_waitsForInFlightBatchBeforeClosingPublisher(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	repos := getReposTillId(t, 1)
	entered := make(chan struct{})
	release := make(chan struct{})
	store.EXPECT().ListEntitiesAfterID(gomock.Any(), gomock.Any()).
		DoAndReturn(func(context.Context, db.ListEntitiesAfterIDParams) ([]db.EntityInstance, error) {
			close(entered)
			<-release
			return repos, nil
		})
	store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), gomock.Any()).
		Return(getStandardOldestRuleEvals(t, repos), nil)
	store.EXPECT().EntityExistsAfterID(gomock.Any(), gomock.Any()).Return(false, nil)

	r := createTestReminder(t, store, &reminderconfig.Config{
		RecurrenceConfig: reminderconfig.RecurrenceConfig{Interval: time.Hour, BatchSize: 5, MinElapsed: time.Minute},
	})
	pub := &fakePublisher{}
	r.eventPublisher = pub
	clock := &fakeClock{}
	r.newTicker = clock.newTicker

	done := startReminder(context.Background(), r)
	tickers := clock.waitForTickers(t, 1)
	tickers[0].fire(t, time.Now())
	<-entered

	stopped := make(chan struct{})
	go func() {
		r.Stop()
		close(stopped)
	}()

	require.Never(t, func() bool {
		select {
		case <-stopped:
			return true
		default:
			return false
		}
	}, 100*time.Millisecond, 5*time.Millisecond, "Stop returned while a batch was in flight")
	require.False(t, pub.isClosed())

	close(release)
	select {
	case <-stopped:
	case <-time.After(waitFor):
		t.Fatal("Stop did not return after the batch finished")
	}
	require.NoError(t, waitStart(t, done))

	require.True(t, allStopped(tickers))
	require.True(t, pub.isClosed())
	require.Len(t, pub.published, 1, "the in-flight batch is still published")
}

func TestStart_contextCancelStopsEveryEntityType(t *testing.T) {
	t.Parallel()

	r := createTestReminder(t, nil, &reminderconfig.Config{
		RecurrenceConfig: reminderconfig.RecurrenceConfig{
			Interval: time.Hour, BatchSize: 5, MinElapsed: time.Minute,
			Entities: map[db.Entities]reminderconfig.EntityRecurrenceConfig{
				db.EntitiesArtifact: {Enabled: ptr(true), Interval: ptr(6 * time.Hour)},
			},
		},
	})
	pub := &fakePublisher{}
	r.eventPublisher = pub
	clock := &fakeClock{}
	r.newTicker = clock.newTicker

	ctx, cancel := context.WithCancel(context.Background())
	done := startReminder(ctx, r)
	tickers := clock.waitForTickers(t, 2)

	cancel()
	require.NoError(t, waitStart(t, done))
	require.True(t, allStopped(tickers))
	require.True(t, pub.isClosed())

	require.ErrorContains(t, r.Start(context.Background()), "cannot start again")
}

func TestStart_errorInOneTypeDoesNotStopAnyType(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	counter := &callCounter{}
	artifacts := []db.EntityInstance{{ID: generateUUIDFromNum(t, 11), EntityType: db.EntitiesArtifact}}
	store.EXPECT().ListEntitiesAfterID(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, arg db.ListEntitiesAfterIDParams) ([]db.EntityInstance, error) {
			counter.inc(arg.EntityType)
			if arg.EntityType == db.EntitiesRepository {
				return nil, sql.ErrConnDone
			}
			return artifacts, nil
		}).Times(3)
	store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), gomock.Any()).
		Return(getStandardOldestRuleEvals(t, artifacts), nil)
	store.EXPECT().EntityExistsAfterID(gomock.Any(), gomock.Any()).Return(false, nil)

	r := createTestReminder(t, store, &reminderconfig.Config{
		RecurrenceConfig: reminderconfig.RecurrenceConfig{
			Interval: time.Hour, BatchSize: 5, MinElapsed: time.Minute,
			Entities: map[db.Entities]reminderconfig.EntityRecurrenceConfig{
				db.EntitiesArtifact: {Enabled: ptr(true), Interval: ptr(2 * time.Hour)},
			},
		},
	})
	pub := &fakePublisher{}
	r.eventPublisher = pub
	clock := &fakeClock{}
	r.newTicker = clock.newTicker

	done := startReminder(context.Background(), r)
	tickers := clock.waitForTickers(t, 2)
	repoTicker := tickerFor(t, tickers, time.Hour)
	artifactTicker := tickerFor(t, tickers, 2*time.Hour)

	now := time.Now()
	repoTicker.fire(t, now)
	repoTicker.fire(t, now.Add(time.Hour))
	artifactTicker.fire(t, now.Add(2*time.Hour))

	r.Stop()
	require.NoError(t, waitStart(t, done))

	require.Equal(t, 2, counter.get(db.EntitiesRepository), "repository keeps running after an error")
	require.Equal(t, 1, counter.get(db.EntitiesArtifact))
	require.Len(t, pub.published, 1)
	require.Len(t, pub.published[0].messages, 1)
}
