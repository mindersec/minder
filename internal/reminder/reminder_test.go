// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	remindermessages "github.com/mindersec/minder/internal/reminder/messages"
	"github.com/mindersec/minder/internal/reminder/metrics"
	reminderconfig "github.com/mindersec/minder/pkg/config/reminder"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

func Test_getEntityBatch(t *testing.T) {
	t.Parallel()

	type expectedOutput struct {
		repos      []db.EntityInstance
		repoCursor uuid.UUID
	}

	type input struct {
		repos []db.EntityInstance
		cfg   reminderconfig.RecurrenceConfig
	}

	tests := []struct {
		name           string
		input          input
		expectedOutput expectedOutput
		setup          func(store *mockdb.MockStore, in input)
		err            string
	}{
		{
			name: "no repos",
			input: input{
				cfg: reminderconfig.RecurrenceConfig{
					BatchSize:  5,
					MinElapsed: time.Hour,
				},
			},
			setup: func(store *mockdb.MockStore, _ input) {
				store.EXPECT().ListEntitiesAfterID(gomock.Any(), gomock.Any()).Return(nil, nil)
				store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), []uuid.UUID{}).Return(nil, nil)
			},
		},
		{
			name: "error listing repos",
			input: input{
				cfg: reminderconfig.RecurrenceConfig{
					BatchSize:  5,
					MinElapsed: time.Hour,
				},
			},
			setup: func(store *mockdb.MockStore, _ input) {
				store.EXPECT().ListEntitiesAfterID(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
			},
			err: sql.ErrConnDone.Error(),
		},
		{
			name: "repo exists after ID",
			input: input{
				repos: getReposTillId(t, 2),
				cfg: reminderconfig.RecurrenceConfig{
					BatchSize:  5,
					MinElapsed: time.Minute,
				},
			},
			expectedOutput: expectedOutput{
				repos:      getReposTillId(t, 2),
				repoCursor: generateUUIDFromNum(t, 2),
			},
			setup: func(store *mockdb.MockStore, in input) {
				store.EXPECT().ListEntitiesAfterID(gomock.Any(), gomock.Any()).Return(in.repos, nil)
				store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), gomock.Any()).Return(getStandardOldestRuleEvals(t, in.repos), nil)
				store.EXPECT().EntityExistsAfterID(gomock.Any(), gomock.Any()).Return(true, nil)
			},
		},
		{
			name: "repo does not exist after ID",
			input: input{
				repos: getReposTillId(t, 2),
				cfg: reminderconfig.RecurrenceConfig{
					BatchSize:  5,
					MinElapsed: time.Minute,
				},
			},
			expectedOutput: expectedOutput{
				repos: getReposTillId(t, 2),
			},
			setup: func(store *mockdb.MockStore, in input) {
				store.EXPECT().ListEntitiesAfterID(gomock.Any(), gomock.Any()).Return(in.repos, nil)
				store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), gomock.Any()).Return(getStandardOldestRuleEvals(t, in.repos), nil)
				store.EXPECT().EntityExistsAfterID(gomock.Any(), gomock.Any()).Return(false, nil)
			},
		},
		{
			name: "error checking if repo exists after ID",
			input: input{
				repos: getReposTillId(t, 3),
				cfg: reminderconfig.RecurrenceConfig{
					BatchSize:  5,
					MinElapsed: time.Minute,
				},
			},
			expectedOutput: expectedOutput{
				repos: getReposTillId(t, 3),
			},
			setup: func(store *mockdb.MockStore, in input) {
				store.EXPECT().ListEntitiesAfterID(gomock.Any(), gomock.Any()).Return(in.repos, nil)
				store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), gomock.Any()).Return(getStandardOldestRuleEvals(t, in.repos), nil)
				store.EXPECT().EntityExistsAfterID(gomock.Any(), gomock.Any()).Return(false, sql.ErrConnDone)
			},
		},
		{
			name: "some repos are eligible",
			input: input{
				repos: getReposTillId(t, 3),
				cfg: reminderconfig.RecurrenceConfig{
					BatchSize:  5,
					MinElapsed: 10 * time.Minute,
				},
			},
			expectedOutput: expectedOutput{
				repos:      getReposTillId(t, 2),
				repoCursor: generateUUIDFromNum(t, 3),
			},
			setup: func(store *mockdb.MockStore, in input) {
				store.EXPECT().ListEntitiesAfterID(gomock.Any(), gomock.Any()).Return(in.repos, nil)
				oldestRuleEvals := getStandardOldestRuleEvals(t, in.repos)
				oldestRuleEvals[2].OldestLastUpdated = time.Now().Add(-time.Second)
				store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), gomock.Any()).Return(oldestRuleEvals, nil)
				store.EXPECT().EntityExistsAfterID(gomock.Any(), gomock.Any()).Return(true, nil)
			},
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			store := mockdb.NewMockStore(ctrl)
			test.setup(store, test.input)
			cfg := &reminderconfig.Config{
				RecurrenceConfig: test.input.cfg,
			}

			r := createTestReminder(t, store, cfg)
			er := newTestEntityReminder(t, r, db.EntitiesRepository)

			got, _, err := er.getEntityBatch(context.Background())
			if test.err != "" {
				require.ErrorContains(t, err, test.err)
				return
			}
			require.NoError(t, err)
			require.ElementsMatch(t, got, test.expectedOutput.repos)
			require.Equal(t, test.expectedOutput.repoCursor, er.cursor)
		})
	}
}

func Test_sendReminders_onlyRepositoriesByDefault(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	repos := getReposTillId(t, 2)

	store.EXPECT().ListEntitiesAfterID(gomock.Any(), db.ListEntitiesAfterIDParams{
		EntityType: db.EntitiesRepository,
		ID:         uuid.Nil,
		Limit:      5,
	}).Return(repos, nil)
	store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), gomock.Any()).
		Return(getStandardOldestRuleEvals(t, repos), nil)
	store.EXPECT().EntityExistsAfterID(gomock.Any(), db.EntityExistsAfterIDParams{
		EntityType: db.EntitiesRepository,
		ID:         generateUUIDFromNum(t, 2),
	}).Return(false, nil)

	r := createTestReminder(t, store, &reminderconfig.Config{
		RecurrenceConfig: reminderconfig.RecurrenceConfig{Interval: time.Hour, BatchSize: 5, MinElapsed: time.Minute},
	})
	require.Len(t, r.entityReminders, 1)
	er := r.entityReminders[0]
	require.Equal(t, db.EntitiesRepository, er.entityType)
	pub := &fakePublisher{}
	r.eventPublisher = pub

	require.NoError(t, er.sendReminders(context.Background()))

	require.Len(t, pub.published, 1)
	require.Equal(t, constants.TopicQueueRepoReminder, pub.published[0].topic)
	require.Len(t, pub.published[0].messages, 2)
	require.Equal(t, uuid.Nil, er.cursor)
}

func Test_getEntityBatch_perTypeCursorAcrossBatches(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	repos := getReposTillId(t, 3)
	artifacts := []db.EntityInstance{
		{ID: generateUUIDFromNum(t, 11), EntityType: db.EntitiesArtifact},
		{ID: generateUUIDFromNum(t, 12), EntityType: db.EntitiesArtifact},
	}

	listParams := func(entityType db.Entities, cursor uuid.UUID) db.ListEntitiesAfterIDParams {
		return db.ListEntitiesAfterIDParams{EntityType: entityType, ID: cursor, Limit: 2}
	}
	existsParams := func(entityType db.Entities, cursor uuid.UUID) db.EntityExistsAfterIDParams {
		return db.EntityExistsAfterIDParams{EntityType: entityType, ID: cursor}
	}
	evals := func(ents []db.EntityInstance) []db.ListOldestRuleEvaluationsByEntityIDRow {
		return getStandardOldestRuleEvals(t, ents)
	}

	gomock.InOrder(
		store.EXPECT().ListEntitiesAfterID(gomock.Any(), listParams(db.EntitiesRepository, uuid.Nil)).
			Return(repos[:2], nil),
		store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), gomock.Any()).Return(evals(repos[:2]), nil),
		store.EXPECT().EntityExistsAfterID(gomock.Any(), existsParams(db.EntitiesRepository, repos[1].ID)).
			Return(true, nil),

		store.EXPECT().ListEntitiesAfterID(gomock.Any(), listParams(db.EntitiesArtifact, uuid.Nil)).
			Return(artifacts, nil),
		store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), gomock.Any()).Return(evals(artifacts), nil),
		store.EXPECT().EntityExistsAfterID(gomock.Any(), existsParams(db.EntitiesArtifact, artifacts[1].ID)).
			Return(false, nil),

		store.EXPECT().ListEntitiesAfterID(gomock.Any(), listParams(db.EntitiesRepository, repos[1].ID)).
			Return(repos[2:], nil),
		store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), gomock.Any()).Return(evals(repos[2:]), nil),
		store.EXPECT().EntityExistsAfterID(gomock.Any(), existsParams(db.EntitiesRepository, repos[2].ID)).
			Return(false, nil),
	)

	r := createTestReminder(t, store, &reminderconfig.Config{
		RecurrenceConfig: reminderconfig.RecurrenceConfig{BatchSize: 2, MinElapsed: time.Minute},
	})
	repoReminder := newTestEntityReminder(t, r, db.EntitiesRepository)
	artifactReminder := newTestEntityReminder(t, r, db.EntitiesArtifact)
	ctx := context.Background()

	got, _, err := repoReminder.getEntityBatch(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, repos[:2], got)
	require.Equal(t, repos[1].ID, repoReminder.cursor)

	got, _, err = artifactReminder.getEntityBatch(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, artifacts, got)
	require.Equal(t, uuid.Nil, artifactReminder.cursor)
	require.Equal(t, repos[1].ID, repoReminder.cursor)

	got, _, err = repoReminder.getEntityBatch(ctx)
	require.NoError(t, err)
	require.ElementsMatch(t, repos[2:], got)
	require.Equal(t, uuid.Nil, repoReminder.cursor)
	require.Equal(t, uuid.Nil, artifactReminder.cursor)
}

type publishedBatch struct {
	topic    string
	messages []*message.Message
}

type fakePublisher struct {
	mu        sync.Mutex
	published []publishedBatch
	closed    bool
}

func (f *fakePublisher) Publish(topic string, messages ...*message.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return errors.New("publisher closed")
	}
	f.published = append(f.published, publishedBatch{topic: topic, messages: messages})
	return nil
}

func (f *fakePublisher) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func (f *fakePublisher) isClosed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

func generateUUIDFromNum(t *testing.T, num int) uuid.UUID {
	t.Helper()

	numberStr := fmt.Sprintf("%d", num)

	uuidStr := fmt.Sprintf("00000000-0000-0000-0000-%012s", numberStr)

	u, err := uuid.Parse(uuidStr)
	if err != nil {
		t.Errorf("error parsing UUID: %v", err)
	}

	return u
}

func getReposTillId(t *testing.T, id int) []db.EntityInstance {
	t.Helper()

	repos := make([]db.EntityInstance, 0, id)
	for i := 1; i <= id; i++ {
		repos = append(repos, db.EntityInstance{
			ID:         generateUUIDFromNum(t, i),
			EntityType: db.EntitiesRepository,
		})
	}

	return repos
}

func createTestReminder(t *testing.T, store db.Store, config *reminderconfig.Config) *reminder {
	t.Helper()

	r := &reminder{
		store:             store,
		cfg:               config,
		stop:              make(chan struct{}),
		metricsServerDone: make(chan struct{}),
		newTicker:         realTicker,
		eventPublisher:    &fakePublisher{},
	}
	if types := enabledEntityTypes(config.RecurrenceConfig); len(types) > 0 && config.RecurrenceConfig.Interval > 0 {
		ers, err := r.newEntityReminders(types)
		require.NoError(t, err)
		for _, er := range ers {
			er.cursor = uuid.Nil
		}
		r.entityReminders = ers
	}
	return r
}

func newTestEntityReminder(t *testing.T, r *reminder, entityType db.Entities) *entityReminder {
	t.Helper()

	return &entityReminder{
		reminder:       r,
		entityType:     entityType,
		recurrence:     r.cfg.RecurrenceConfig.ForEntityType(entityType),
		entityTypeAttr: metrics.EntityTypeAttribute(string(entityType)),
	}
}

func entityReminderFor(t *testing.T, r *reminder, entityType db.Entities) *entityReminder {
	t.Helper()

	for _, er := range r.entityReminders {
		if er.entityType == entityType {
			return er
		}
	}
	t.Fatalf("no entity reminder for %s", entityType)
	return nil
}

func getStandardOldestRuleEvals(t *testing.T, repos []db.EntityInstance) []db.ListOldestRuleEvaluationsByEntityIDRow {
	t.Helper()

	oldestRuleEvals := make([]db.ListOldestRuleEvaluationsByEntityIDRow, 0, len(repos))
	for _, repo := range repos {
		oldestRuleEvals = append(oldestRuleEvals, db.ListOldestRuleEvaluationsByEntityIDRow{
			EntityInstanceID:  repo.ID,
			OldestLastUpdated: time.Now().Add(-time.Hour),
		})
	}

	return oldestRuleEvals
}

func Test_enabledEntityTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entities map[db.Entities]reminderconfig.EntityRecurrenceConfig
		want     []db.Entities
	}{
		{
			name: "default config only enables repositories",
			want: []db.Entities{db.EntitiesRepository},
		},
		{
			name: "artifact override without enabled stays off",
			entities: map[db.Entities]reminderconfig.EntityRecurrenceConfig{
				db.EntitiesArtifact: {BatchSize: ptr(10)},
			},
			want: []db.Entities{db.EntitiesRepository},
		},
		{
			name: "artifact opted in",
			entities: map[db.Entities]reminderconfig.EntityRecurrenceConfig{
				db.EntitiesArtifact: {Enabled: ptr(true)},
			},
			want: []db.Entities{db.EntitiesRepository, db.EntitiesArtifact},
		},
		{
			name: "repositories disabled, artifact opted in",
			entities: map[db.Entities]reminderconfig.EntityRecurrenceConfig{
				db.EntitiesRepository: {Enabled: ptr(false)},
				db.EntitiesArtifact:   {Enabled: ptr(true)},
			},
			want: []db.Entities{db.EntitiesArtifact},
		},
		{
			name: "unsupported type is ignored",
			entities: map[db.Entities]reminderconfig.EntityRecurrenceConfig{
				db.EntitiesPullRequest: {Enabled: ptr(true)},
			},
			want: []db.Entities{db.EntitiesRepository},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := reminderconfig.RecurrenceConfig{Interval: time.Hour, Entities: tt.entities}
			require.Equal(t, tt.want, enabledEntityTypes(cfg))
		})
	}
}

func Test_newEntityReminders(t *testing.T) {
	t.Parallel()

	build := func(cfg reminderconfig.RecurrenceConfig) ([]*entityReminder, error) {
		r := &reminder{cfg: &reminderconfig.Config{RecurrenceConfig: cfg}}
		return r.newEntityReminders(enabledEntityTypes(cfg))
	}
	intervals := func(ers []*entityReminder) map[db.Entities]time.Duration {
		out := make(map[db.Entities]time.Duration, len(ers))
		for _, er := range ers {
			out[er.entityType] = er.recurrence.Interval
		}
		return out
	}

	ers, err := build(reminderconfig.RecurrenceConfig{Interval: time.Hour})
	require.NoError(t, err)
	require.Equal(t, map[db.Entities]time.Duration{db.EntitiesRepository: time.Hour}, intervals(ers))

	ers, err = build(reminderconfig.RecurrenceConfig{
		Interval: time.Hour,
		Entities: map[db.Entities]reminderconfig.EntityRecurrenceConfig{
			db.EntitiesArtifact: {Enabled: ptr(true), Interval: ptr(6 * time.Hour)},
		},
	})
	require.NoError(t, err)
	require.Equal(t, map[db.Entities]time.Duration{
		db.EntitiesRepository: time.Hour,
		db.EntitiesArtifact:   6 * time.Hour,
	}, intervals(ers), "each type keeps its own interval")
	for _, er := range ers {
		require.NotEqual(t, uuid.Nil, er.cursor, "cursor starts at a random UUID")
	}

	_, err = build(reminderconfig.RecurrenceConfig{
		Interval: time.Hour,
		Entities: map[db.Entities]reminderconfig.EntityRecurrenceConfig{
			db.EntitiesRepository: {Enabled: ptr(false)},
		},
	})
	require.ErrorContains(t, err, "no entity types are enabled")

	_, err = build(reminderconfig.RecurrenceConfig{})
	require.ErrorContains(t, err, "invalid interval for repository")
}

func Test_sendReminders_artifactEnabledViaConfig(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	artifact := db.EntityInstance{
		ID:         generateUUIDFromNum(t, 11),
		EntityType: db.EntitiesArtifact,
		ProviderID: uuid.New(),
		ProjectID:  uuid.New(),
	}
	recentArtifact := db.EntityInstance{ID: generateUUIDFromNum(t, 12), EntityType: db.EntitiesArtifact}

	store.EXPECT().ListEntitiesAfterID(gomock.Any(), db.ListEntitiesAfterIDParams{
		EntityType: db.EntitiesArtifact, ID: uuid.Nil, Limit: 3,
	}).Return([]db.EntityInstance{artifact, recentArtifact}, nil)
	store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), []uuid.UUID{artifact.ID, recentArtifact.ID}).
		Return([]db.ListOldestRuleEvaluationsByEntityIDRow{
			{EntityInstanceID: artifact.ID, OldestLastUpdated: time.Now().Add(-3 * time.Hour)},
			{EntityInstanceID: recentArtifact.ID, OldestLastUpdated: time.Now().Add(-90 * time.Minute)},
		}, nil)
	store.EXPECT().EntityExistsAfterID(gomock.Any(), db.EntityExistsAfterIDParams{
		EntityType: db.EntitiesArtifact, ID: recentArtifact.ID,
	}).Return(false, nil)

	r := createTestReminder(t, store, &reminderconfig.Config{
		RecurrenceConfig: reminderconfig.RecurrenceConfig{
			Interval: time.Hour, BatchSize: 100, MinElapsed: time.Hour,
			Entities: map[db.Entities]reminderconfig.EntityRecurrenceConfig{
				db.EntitiesArtifact: {Enabled: ptr(true), BatchSize: ptr(3), MinElapsed: ptr(2 * time.Hour)},
			},
		},
	})
	pub := &fakePublisher{}
	r.eventPublisher = pub

	require.NoError(t, entityReminderFor(t, r, db.EntitiesArtifact).sendReminders(context.Background()))

	require.Len(t, pub.published, 1)
	require.Equal(t, constants.TopicQueueRepoReminder, pub.published[0].topic)
	require.Len(t, pub.published[0].messages, 1, "only the artifact past its own min_elapsed is sent")

	evt, err := remindermessages.EntityReminderEventFromMessage(pub.published[0].messages[0])
	require.NoError(t, err)
	require.Equal(t, remindermessages.EntityReminderEvent{
		Project: artifact.ProjectID, ProviderID: artifact.ProviderID, EntityID: artifact.ID,
	}, *evt)
}

func Test_sendReminders_metricsTaggedWithEntityType(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)

	repo := getReposTillId(t, 1)
	artifacts := []db.EntityInstance{
		{ID: generateUUIDFromNum(t, 11), EntityType: db.EntitiesArtifact},
		{ID: generateUUIDFromNum(t, 12), EntityType: db.EntitiesArtifact},
	}
	store.EXPECT().ListEntitiesAfterID(gomock.Any(), gomock.Any()).Return(repo, nil)
	store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), gomock.Any()).
		Return(getStandardOldestRuleEvals(t, repo), nil)
	store.EXPECT().ListEntitiesAfterID(gomock.Any(), gomock.Any()).Return(artifacts, nil)
	store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), gomock.Any()).
		Return(getStandardOldestRuleEvals(t, artifacts), nil)
	store.EXPECT().EntityExistsAfterID(gomock.Any(), gomock.Any()).Return(false, nil).Times(2)

	reader := sdkmetric.NewManualReader()
	m, err := metrics.NewMetrics(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)).Meter("test"))
	require.NoError(t, err)

	r := createTestReminder(t, store, &reminderconfig.Config{
		RecurrenceConfig: reminderconfig.RecurrenceConfig{
			Interval: time.Hour, BatchSize: 10, MinElapsed: time.Minute,
			Entities: map[db.Entities]reminderconfig.EntityRecurrenceConfig{db.EntitiesArtifact: {Enabled: ptr(true)}},
		},
	})
	r.eventPublisher = &fakePublisher{}
	r.metrics = m

	ctx := context.Background()
	for _, er := range r.entityReminders {
		require.NoError(t, er.sendReminders(ctx))
	}

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	batchSizes := map[string]int64{}
	sendDelayCounts := map[string]uint64{}
	for _, sm := range rm.ScopeMetrics {
		for _, md := range sm.Metrics {
			switch md.Name {
			case "batch_size":
				data, ok := md.Data.(metricdata.Histogram[int64])
				require.True(t, ok)
				for _, dp := range data.DataPoints {
					v, ok := dp.Attributes.Value(metrics.EntityTypeKey)
					require.True(t, ok, "batch_size data point without entity_type")
					batchSizes[v.AsString()] = dp.Sum
				}
			case "send_delay":
				data, ok := md.Data.(metricdata.Histogram[float64])
				require.True(t, ok)
				for _, dp := range data.DataPoints {
					v, ok := dp.Attributes.Value(metrics.EntityTypeKey)
					require.True(t, ok, "send_delay data point without entity_type")
					sendDelayCounts[v.AsString()] = dp.Count
				}
			}
		}
	}

	require.Equal(t, map[string]int64{"repository": 1, "artifact": 2}, batchSizes)
	require.Equal(t, map[string]uint64{"repository": 1, "artifact": 2}, sendDelayCounts)
}

func ptr[T any](v T) *T {
	return &v
}
