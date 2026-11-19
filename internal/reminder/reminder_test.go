// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminder

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
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

			got, _, err := r.getEntityBatch(context.Background(), db.EntitiesRepository)
			if test.err != "" {
				require.ErrorContains(t, err, test.err)
				return
			}
			require.NoError(t, err)
			require.ElementsMatch(t, got, test.expectedOutput.repos)
			require.Equal(t, test.expectedOutput.repoCursor, r.cursors[db.EntitiesRepository])
		})
	}
}

func Test_sendReminders_onlyRepositoriesByDefault(t *testing.T) {
	t.Parallel()

	require.Equal(t, []db.Entities{db.EntitiesRepository}, defaultEntityTypes)

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
		RecurrenceConfig: reminderconfig.RecurrenceConfig{BatchSize: 5, MinElapsed: time.Minute},
	})
	pub := &fakePublisher{}
	r.eventPublisher = pub

	require.NoError(t, r.sendReminders(context.Background()))

	require.Len(t, pub.published, 1)
	require.Equal(t, constants.TopicQueueRepoReminder, pub.published[0].topic)
	require.Len(t, pub.published[0].messages, 2)
	require.Equal(t, map[db.Entities]uuid.UUID{db.EntitiesRepository: uuid.Nil}, r.cursors)
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
	ctx := context.Background()

	got, _, err := r.getEntityBatch(ctx, db.EntitiesRepository)
	require.NoError(t, err)
	require.ElementsMatch(t, repos[:2], got)
	require.Equal(t, repos[1].ID, r.cursors[db.EntitiesRepository])

	got, _, err = r.getEntityBatch(ctx, db.EntitiesArtifact)
	require.NoError(t, err)
	require.ElementsMatch(t, artifacts, got)
	require.Equal(t, uuid.Nil, r.cursors[db.EntitiesArtifact])
	require.Equal(t, repos[1].ID, r.cursors[db.EntitiesRepository])

	got, _, err = r.getEntityBatch(ctx, db.EntitiesRepository)
	require.NoError(t, err)
	require.ElementsMatch(t, repos[2:], got)
	require.Equal(t, uuid.Nil, r.cursors[db.EntitiesRepository])
	require.Equal(t, uuid.Nil, r.cursors[db.EntitiesArtifact])
}

func Test_sendReminders_errorInOneTypeDoesNotSkipOthers(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	store := mockdb.NewMockStore(ctrl)
	artifacts := []db.EntityInstance{{ID: generateUUIDFromNum(t, 11), EntityType: db.EntitiesArtifact}}

	gomock.InOrder(
		store.EXPECT().ListEntitiesAfterID(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone),
		store.EXPECT().ListEntitiesAfterID(gomock.Any(), gomock.Any()).Return(artifacts, nil),
		store.EXPECT().ListOldestRuleEvaluationsByEntityID(gomock.Any(), gomock.Any()).
			Return(getStandardOldestRuleEvals(t, artifacts), nil),
		store.EXPECT().EntityExistsAfterID(gomock.Any(), gomock.Any()).Return(false, nil),
	)

	r := createTestReminder(t, store, &reminderconfig.Config{
		RecurrenceConfig: reminderconfig.RecurrenceConfig{BatchSize: 5, MinElapsed: time.Minute},
	})
	r.entityTypes = []db.Entities{db.EntitiesRepository, db.EntitiesArtifact}
	pub := &fakePublisher{}
	r.eventPublisher = pub

	err := r.sendReminders(context.Background())
	require.ErrorIs(t, err, sql.ErrConnDone)
	require.ErrorContains(t, err, string(db.EntitiesRepository))
	require.Len(t, pub.published, 1)
	require.Len(t, pub.published[0].messages, 1)
}

type publishedBatch struct {
	topic    string
	messages []*message.Message
}

type fakePublisher struct {
	published []publishedBatch
}

func (f *fakePublisher) Publish(topic string, messages ...*message.Message) error {
	f.published = append(f.published, publishedBatch{topic: topic, messages: messages})
	return nil
}

func (*fakePublisher) Close() error {
	return nil
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

	return &reminder{
		store:       store,
		cfg:         config,
		entityTypes: defaultEntityTypes,
		cursors:     make(map[db.Entities]uuid.UUID),
	}
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
