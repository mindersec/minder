//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package db

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/util/rand"
)

func TestListEvaluationHistoryFilters(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, proj.ID)
	repo1 := createRandomRepository(t, proj.ID, prov)
	ruleType1 := createRandomRuleType(t, proj.ID)
	profile1 := createRandomProfile(t, proj.ID, []string{})
	riID1 := createRandomRuleInstance(
		t,
		proj.ID,
		profile1.ID,
		ruleType1.ID,
	)
	ere1 := createRandomEvaluationRuleEntity(t, riID1, repo1.ID)
	es1 := createRandomEvaluationStatus(t, ere1)

	tests := []struct {
		name   string
		params ListEvaluationHistoryParams
		checkf func(*testing.T, []ListEvaluationHistoryRow)
		error  bool
	}{
		{
			name: "no filters",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 1)
				row := rows[0]
				require.Equal(t, es1, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, repo1.RepoOwner, row.RepoOwner.String)
				require.Equal(t, repo1.RepoName, row.RepoName.String)
			},
		},

		// entity type filter
		{
			name: "entity type filter include repository",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Entitytypes: []Entities{EntitiesRepository},
				Projectid:   proj.ID,
				Size:        5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 1)
				row := rows[0]
				require.Equal(t, es1, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, repo1.RepoOwner, row.RepoOwner.String)
				require.Equal(t, repo1.RepoName, row.RepoName.String)
			},
		},
		{
			name: "entity type filter exclude repository",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Notentitytypes: []Entities{EntitiesRepository},
				Projectid:      proj.ID,
				Size:           5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 0)
			},
		},
		{
			name: "entity type filter include pr",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Entitytypes: []Entities{EntitiesPullRequest},
				Projectid:   proj.ID,
				Size:        5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 0)
			},
		},
		{
			name: "entity type filter exclude pr",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Notentitytypes: []Entities{EntitiesPullRequest},
				Projectid:      proj.ID,
				Size:           5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 1)
				row := rows[0]
				require.Equal(t, es1, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, repo1.RepoOwner, row.RepoOwner.String)
				require.Equal(t, repo1.RepoName, row.RepoName.String)
			},
		},
		{
			name: "entity type filter multi include",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Entitytypes: []Entities{EntitiesRepository, EntitiesPullRequest},
				Projectid:   proj.ID,
				Size:        5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 1)
				row := rows[0]
				require.Equal(t, es1, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, repo1.RepoOwner, row.RepoOwner.String)
				require.Equal(t, repo1.RepoName, row.RepoName.String)
			},
		},
		{
			name: "entity type filter multi exclude",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Notentitytypes: []Entities{EntitiesRepository, EntitiesPullRequest},
				Projectid:      proj.ID,
				Size:           5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 0)
			},
		},

		// entity name filter
		{
			name: "entity name filter include repository",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Entitynames: []string{fullRepoName(repo1)},
				Projectid:   proj.ID,
				Size:        5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 1)
				row := rows[0]
				require.Equal(t, es1, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, repo1.RepoOwner, row.RepoOwner.String)
				require.Equal(t, repo1.RepoName, row.RepoName.String)
			},
		},
		{
			name: "entity name filter exclude repository",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Notentitynames: []string{fullRepoName(repo1)},
				Projectid:      proj.ID,
				Size:           5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 0)
			},
		},
		{
			name: "entity name filter include absent",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Entitynames: []string{"stacklok/minder"},
				Projectid:   proj.ID,
				Size:        5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 0)
			},
		},
		{
			name: "entity name filter exclude absent",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Notentitynames: []string{"stacklok/minder"},
				Projectid:      proj.ID,
				Size:           5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 1)
				row := rows[0]
				require.Equal(t, es1, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, repo1.RepoOwner, row.RepoOwner.String)
				require.Equal(t, repo1.RepoName, row.RepoName.String)
			},
		},
		{
			name: "entity name filter multi include",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Entitynames: []string{fullRepoName(repo1), "stacklok/minder"},
				Projectid:   proj.ID,
				Size:        5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 1)
				row := rows[0]
				require.Equal(t, es1, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, repo1.RepoOwner, row.RepoOwner.String)
				require.Equal(t, repo1.RepoName, row.RepoName.String)
			},
		},
		{
			name: "entity name filter multi exclude",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Notentitynames: []string{fullRepoName(repo1), "stacklok/minder"},
				Projectid:      proj.ID,
				Size:           5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 0)
			},
		},

		// profile name filter
		{
			name: "profile name filter include repository",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Profilenames: []string{profile1.Name},
				Projectid:    proj.ID,
				Size:         5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 1)
				row := rows[0]
				require.Equal(t, es1, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, repo1.RepoOwner, row.RepoOwner.String)
				require.Equal(t, repo1.RepoName, row.RepoName.String)
			},
		},
		{
			name: "profile name filter exclude repository",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Notprofilenames: []string{profile1.Name},
				Projectid:       proj.ID,
				Size:            5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 0)
			},
		},
		{
			name: "profile name filter include absent",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Profilenames: []string{"random_profile"},
				Projectid:    proj.ID,
				Size:         5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 0)
			},
		},
		{
			name: "profile name filter exclude absent",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Notprofilenames: []string{"stacklok/minder"},
				Projectid:       proj.ID,
				Size:            5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 1)
				row := rows[0]
				require.Equal(t, es1, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, repo1.RepoOwner, row.RepoOwner.String)
				require.Equal(t, repo1.RepoName, row.RepoName.String)
			},
		},
		{
			name: "profile name filter multi include",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Profilenames: []string{profile1.Name, "stacklok/minder"},
				Projectid:    proj.ID,
				Size:         5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 1)
				row := rows[0]
				require.Equal(t, es1, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, repo1.RepoOwner, row.RepoOwner.String)
				require.Equal(t, repo1.RepoName, row.RepoName.String)
			},
		},
		{
			name: "profile name filter multi exclude",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Notprofilenames: []string{profile1.Name, "stacklok/minder"},
				Projectid:       proj.ID,
				Size:            5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 0)
			},
		},

		// time range filter
		{
			name: "time range filter from +1h",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Fromts: sql.NullTime{
					Time:  time.Now().Add(+1 * time.Hour),
					Valid: true,
				},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 0)
			},
		},
		{
			name: "time range filter from -1h",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Fromts: sql.NullTime{
					Time:  time.Now().Add(-1 * time.Hour),
					Valid: true,
				},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 1)
				row := rows[0]
				require.Equal(t, es1, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, repo1.RepoOwner, row.RepoOwner.String)
				require.Equal(t, repo1.RepoName, row.RepoName.String)
			},
		},
		{
			name: "time range filter to +1h",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Tots: sql.NullTime{
					Time:  time.Now().Add(+1 * time.Hour),
					Valid: true,
				},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 1)
				row := rows[0]
				require.Equal(t, es1, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, repo1.RepoOwner, row.RepoOwner.String)
				require.Equal(t, repo1.RepoName, row.RepoName.String)
			},
		},
		{
			name: "time range filter to -1h",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Tots: sql.NullTime{
					Time:  time.Now().Add(-1 * time.Hour),
					Valid: true,
				},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 0)
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rows, err := testQueries.ListEvaluationHistory(
				context.Background(),
				tt.params,
			)

			if tt.error {
				require.Error(t, err)
				require.Nil(t, rows)
				return
			}

			require.NoError(t, err)
			tt.checkf(t, rows)
		})
	}
}

func TestListEvaluationHistoryPagination(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, proj.ID)

	repos := make([]Repository, 0)
	for i := 0; i < 10; i++ {
		repos = append(repos, createRandomRepository(t, proj.ID, prov))
	}
	ruleType1 := createRandomRuleType(t, proj.ID)
	profile1 := createRandomProfile(t, proj.ID, []string{})
	riID1 := createRandomRuleInstance(
		t,
		proj.ID,
		profile1.ID,
		ruleType1.ID,
	)

	ess := make([]uuid.UUID, 0)
	for i := 0; i < 10; i++ {
		ere := createRandomEvaluationRuleEntity(t, riID1, repos[i].ID)
		ess = append(ess, createRandomEvaluationStatus(t, ere))
	}

	tests := []struct {
		name   string
		params ListEvaluationHistoryParams
		checkf func(*testing.T, []ListEvaluationHistoryRow)
		error  bool
	}{
		{
			name: "default page",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 5)
			},
		},
		{
			name: "next page most recent first",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 5)
				row := rows[0]
				require.Equal(t, ess[9], row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repos[9].ID, row.EntityID)
				require.Equal(t, repos[9].RepoOwner, row.RepoOwner.String)
				require.Equal(t, repos[9].RepoName, row.RepoName.String)
			},
		},
		{
			name: "next page normal ordering",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 5)
				prev := time.UnixMicro(999999999999999999).UTC()
				for _, row := range rows {
					require.Truef(t, prev.After(row.EvaluatedAt),
						"%s\n%s\n",
						prev,
						row.EvaluatedAt,
					)
					prev = row.EvaluatedAt
				}
			},
		},
		{
			name: "prev page least recent first",
			params: ListEvaluationHistoryParams{
				Prev: sql.NullTime{
					Time:  time.UnixMicro(0).UTC(),
					Valid: true,
				},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 5)
				row := rows[0]
				require.Equal(t, ess[0], row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repos[0].ID, row.EntityID)
				require.Equal(t, repos[0].RepoOwner, row.RepoOwner.String)
				require.Equal(t, repos[0].RepoName, row.RepoName.String)
			},
		},
		{
			name: "prev page reverse ordering",
			params: ListEvaluationHistoryParams{
				Prev: sql.NullTime{
					Time:  time.UnixMicro(0).UTC(),
					Valid: true,
				},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 5)
				prev := time.UnixMicro(0).UTC()
				for _, row := range rows {
					require.Truef(t, row.EvaluatedAt.After(prev),
						"%s\n%s\n",
						prev,
						row.EvaluatedAt,
					)
					prev = row.EvaluatedAt
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rows, err := testQueries.ListEvaluationHistory(
				context.Background(),
				tt.params,
			)

			if tt.error {
				require.Error(t, err)
				require.Nil(t, rows)
				return
			}

			require.NoError(t, err)
			tt.checkf(t, rows)
		})
	}
}

func fullRepoName(r Repository) string {
	return fmt.Sprintf("%s/%s", r.RepoOwner, r.RepoName)
}

func createRandomEvaluationRuleEntity(
	t *testing.T,
	ruleID uuid.UUID,
	entityID uuid.UUID,
) uuid.UUID {
	t.Helper()

	ereID, err := testQueries.InsertEvaluationRuleEntity(
		context.Background(),
		InsertEvaluationRuleEntityParams{
			RuleID: ruleID,
			RepositoryID: uuid.NullUUID{
				UUID:  entityID,
				Valid: true,
			},
			EntityType: EntitiesRepository,
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, ereID)

	return ereID
}

func createRandomEvaluationStatus(
	t *testing.T,
	ereID uuid.UUID,
) uuid.UUID {
	t.Helper()

	seed := time.Now().UnixNano()
	evaluationStatus := rand.RandomFrom(
		[]EvalStatusTypes{
			EvalStatusTypesSuccess,
			EvalStatusTypesFailure,
			EvalStatusTypesError,
			EvalStatusTypesSkipped,
			EvalStatusTypesPending,
		},
		seed,
	)
	esID, err := testQueries.InsertEvaluationStatus(
		context.Background(),
		InsertEvaluationStatusParams{
			RuleEntityID: ereID,
			Status:       evaluationStatus,
			Details:      "",
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, esID)

	return esID
}
