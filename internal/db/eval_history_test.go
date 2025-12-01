// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/util/rand"
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

	// Evaluations for this profile should not show up in the
	// results.
	ruleType2 := createRandomRuleType(t, proj.ID)
	profile2 := createRandomProfile(t, proj.ID, []string{"label2"})
	fmt.Println(profile2)
	riID2 := createRandomRuleInstance(
		t,
		proj.ID,
		profile2.ID,
		ruleType2.ID,
	)
	ere2 := createRandomEvaluationRuleEntity(t, riID2, repo1.ID)
	es2 := createRandomEvaluationStatus(t, ere2)

	// Evaluations for this profile should not show up in the
	// results.
	ruleType3 := createRandomRuleType(t, proj.ID)
	profile3 := createRandomProfile(t, proj.ID, []string{"label3"})
	fmt.Println(profile3)
	riID3 := createRandomRuleInstance(
		t,
		proj.ID,
		profile3.ID,
		ruleType3.ID,
	)
	ere3 := createRandomEvaluationRuleEntity(t, riID3, repo1.ID)
	es3 := createRandomEvaluationStatus(t, ere3)

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
				Entitynames: []string{"mindersec/minder"},
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
				Notentitynames: []string{"mindersec/minder"},
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
			},
		},
		{
			name: "entity name filter multi include",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Entitynames: []string{fullRepoName(repo1), "mindersec/minder"},
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
			},
		},
		{
			name: "entity name filter multi exclude",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Notentitynames: []string{fullRepoName(repo1), "mindersec/minder"},
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
				Notprofilenames: []string{"mindersec/minder"},
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
			},
		},
		{
			name: "profile name filter multi include",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Profilenames: []string{profile1.Name, "mindersec/minder"},
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
			},
		},
		{
			name: "profile name filter multi exclude",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Notprofilenames: []string{profile1.Name, "mindersec/minder"},
				Projectid:       proj.ID,
				Size:            5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 0)
			},
		},

		// profile labels filter
		{
			name: "profile labels filter missing",
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
				require.Empty(t, row.ProfileLabels)
			},
		},
		{
			name: "profile labels filter include",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Labels:    []string{"nonexisting"},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 0)
			},
		},
		{
			name: "profile labels filter include match label2",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Labels:    []string{"label2"},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 1)
				row := rows[0]
				require.Equal(t, es2, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, profile2.Labels, row.ProfileLabels)
			},
		},
		{
			name: "profile labels filter include match label3",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Labels:    []string{"label3"},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 1)
				row := rows[0]
				require.Equal(t, es3, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, profile3.Labels, row.ProfileLabels)
			},
		},
		{
			name: "profile labels filter match *",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Labels:    []string{"*"},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 3)

				row := rows[0]
				require.Equal(t, es3, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, profile3.Labels, row.ProfileLabels)

				row = rows[1]
				require.Equal(t, es2, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, profile2.Labels, row.ProfileLabels)

				row = rows[2]
				require.Equal(t, es1, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, profile1.Labels, row.ProfileLabels)
			},
		},
		{
			name: "profile labels filter exclude label2",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Notlabels: []string{"label2"},
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
				require.Equal(t, profile1.Labels, row.ProfileLabels)
			},
		},
		{
			name: "profile labels filter exclude label3",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Notlabels: []string{"label3"},
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
				require.Equal(t, profile1.Labels, row.ProfileLabels)
			},
		},
		{
			name: "profile labels filter include * exclude label2",
			params: ListEvaluationHistoryParams{
				Next: sql.NullTime{
					Time:  time.UnixMicro(999999999999999999).UTC(),
					Valid: true,
				},
				Labels:    []string{"*"},
				Notlabels: []string{"label3"},
				Projectid: proj.ID,
				Size:      5,
			},
			checkf: func(t *testing.T, rows []ListEvaluationHistoryRow) {
				t.Helper()
				require.Len(t, rows, 2)

				row := rows[0]
				require.Equal(t, es2, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, profile2.Labels, row.ProfileLabels)

				row = rows[1]
				require.Equal(t, es1, row.EvaluationID)
				require.Equal(t, EntitiesRepository, row.EntityType)
				require.Equal(t, repo1.ID, row.EntityID)
				require.Equal(t, profile1.Labels, row.ProfileLabels)
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

	repos := make([]EntityRepository, 0)
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

func TestGetEvaluationHistory(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, proj.ID)

	repos := make([]EntityRepository, 0)
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

	absent, err := uuid.NewRandom()
	require.NoError(t, err)

	tests := []struct {
		name   string
		params GetEvaluationHistoryParams
		checkf func(*testing.T, GetEvaluationHistoryRow)
		norows bool
		error  bool
	}{
		{
			name: "present",
			params: GetEvaluationHistoryParams{
				EvaluationID: ess[0],
				ProjectID:    proj.ID,
			},
			checkf: func(t *testing.T, row GetEvaluationHistoryRow) {
				t.Helper()
				require.Equal(t, ess[0], row.EvaluationID)
			},
		},
		{
			name: "absent",
			params: GetEvaluationHistoryParams{
				EvaluationID: absent,
				ProjectID:    proj.ID,
			},
			norows: true,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			row, err := testQueries.GetEvaluationHistory(
				context.Background(),
				tt.params,
			)

			if tt.error {
				require.Error(t, err)
				require.Nil(t, row)
				return
			}

			if tt.norows {
				require.Error(t, err)
				require.True(t, errors.Is(err, sql.ErrNoRows))
				return
			}

			require.NoError(t, err)
			tt.checkf(t, row)
		})
	}
}

func fullRepoName(r EntityRepository) string {
	return r.Name
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
			RuleID:           ruleID,
			EntityType:       EntitiesRepository,
			EntityInstanceID: entityID,
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
			Checkpoint:   json.RawMessage(`{}`),
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, esID)

	return esID
}
