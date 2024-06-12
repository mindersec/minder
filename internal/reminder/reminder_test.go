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

package reminder

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	reminderconfig "github.com/stacklok/minder/internal/config/reminder"
	"github.com/stacklok/minder/internal/db"
)

func Test_getRepositoryBatch(t *testing.T) {
	t.Parallel()

	type expectedOutput struct {
		repos      []db.Repository
		repoCursor uuid.UUID
	}

	type input struct {
		repos []db.Repository
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
				store.EXPECT().ListRepositoriesAfterID(gomock.Any(), gomock.Any()).Return(nil, nil)
				store.EXPECT().ListOldestRuleEvaluationsByRepositoryId(gomock.Any(), []uuid.UUID{}).Return(nil, nil)
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
				store.EXPECT().ListRepositoriesAfterID(gomock.Any(), gomock.Any()).Return(nil, sql.ErrConnDone)
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
				store.EXPECT().ListRepositoriesAfterID(gomock.Any(), gomock.Any()).Return(in.repos, nil)
				store.EXPECT().ListOldestRuleEvaluationsByRepositoryId(gomock.Any(), gomock.Any()).Return(getStandardOldestRuleEvals(t, in.repos), nil)
				store.EXPECT().RepositoryExistsAfterID(gomock.Any(), gomock.Any()).Return(true, nil)
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
				store.EXPECT().ListRepositoriesAfterID(gomock.Any(), gomock.Any()).Return(in.repos, nil)
				store.EXPECT().ListOldestRuleEvaluationsByRepositoryId(gomock.Any(), gomock.Any()).Return(getStandardOldestRuleEvals(t, in.repos), nil)
				store.EXPECT().RepositoryExistsAfterID(gomock.Any(), gomock.Any()).Return(false, nil)
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
				store.EXPECT().ListRepositoriesAfterID(gomock.Any(), gomock.Any()).Return(in.repos, nil)
				store.EXPECT().ListOldestRuleEvaluationsByRepositoryId(gomock.Any(), gomock.Any()).Return(getStandardOldestRuleEvals(t, in.repos), nil)
				store.EXPECT().RepositoryExistsAfterID(gomock.Any(), gomock.Any()).Return(false, sql.ErrConnDone)
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
				store.EXPECT().ListRepositoriesAfterID(gomock.Any(), gomock.Any()).Return(in.repos, nil)
				oldestRuleEvals := getStandardOldestRuleEvals(t, in.repos)
				oldestRuleEvals[2].OldestLastUpdated = time.Now().Add(-time.Second)
				store.EXPECT().ListOldestRuleEvaluationsByRepositoryId(gomock.Any(), gomock.Any()).Return(oldestRuleEvals, nil)
				store.EXPECT().RepositoryExistsAfterID(gomock.Any(), gomock.Any()).Return(true, nil)
			},
		},
	}

	for _, test := range tests {
		test := test

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

			got, err := r.getRepositoryBatch(context.Background())
			if test.err != "" {
				require.ErrorContains(t, err, test.err)
				return
			}
			require.NoError(t, err)
			require.ElementsMatch(t, got, test.expectedOutput.repos)
			require.Equal(t, test.expectedOutput.repoCursor, r.repositoryCursor)
		})
	}
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

func getReposTillId(t *testing.T, id int) []db.Repository {
	t.Helper()

	repos := make([]db.Repository, 0, id)
	for i := 1; i <= id; i++ {
		repos = append(repos, db.Repository{ID: generateUUIDFromNum(t, i)})
	}

	return repos
}

func createTestReminder(t *testing.T, store db.Store, config *reminderconfig.Config) *reminder {
	t.Helper()

	return &reminder{
		store: store,
		cfg:   config,
	}
}

func getStandardOldestRuleEvals(t *testing.T, repos []db.Repository) []db.ListOldestRuleEvaluationsByRepositoryIdRow {
	t.Helper()

	oldestRuleEvals := make([]db.ListOldestRuleEvaluationsByRepositoryIdRow, 0, len(repos))
	for _, repo := range repos {
		oldestRuleEvals = append(oldestRuleEvals, db.ListOldestRuleEvaluationsByRepositoryIdRow{
			RepositoryID:      repo.ID,
			OldestLastUpdated: time.Now().Add(-time.Hour),
		})
	}

	return oldestRuleEvals
}
