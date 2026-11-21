// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// nonUTCTimeZone is a session time zone with a non-zero, non-whole-hour
// offset and no daylight saving time, so any accidental conversion between
// UTC and session-local wall time shows up as a 5h30m shift.
const nonUTCTimeZone = "Asia/Kolkata"

// queriesInSessionTimeZone returns Queries bound to a single connection whose
// session time zone is set to tz. The time zone is reset before the
// connection goes back to the shared pool.
func queriesInSessionTimeZone(t *testing.T, tz string) *Queries {
	t.Helper()

	ctx := context.Background()
	conn, err := testDB.Conn(ctx)
	require.NoError(t, err)

	_, err = conn.ExecContext(ctx, "SELECT set_config('TimeZone', $1, false)", tz)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := conn.ExecContext(context.Background(), "RESET TIME ZONE")
		require.NoError(t, err)
		require.NoError(t, conn.Close())
	})

	return New(conn)
}

func evaluationTime(t *testing.T, esID uuid.UUID) time.Time {
	t.Helper()

	var evalTime time.Time
	err := testDB.QueryRowContext(context.Background(),
		"SELECT evaluation_time FROM evaluation_statuses WHERE id = $1", esID,
	).Scan(&evalTime)
	require.NoError(t, err)
	return evalTime
}

// createLatestEvaluation records one evaluation for a fresh repository and
// marks it as the latest one, returning the repository, the evaluation status
// ID, and the evaluation time stored by the database.
func createLatestEvaluation(t *testing.T) (uuid.UUID, EntityRepository, uuid.UUID, time.Time) {
	t.Helper()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, proj.ID)
	repo := createRandomRepository(t, proj.ID, prov)
	ruleType := createRandomRuleType(t, proj.ID)
	profile := createRandomProfile(t, proj.ID, []string{})
	riID := createRandomRuleInstance(t, proj.ID, profile.ID, ruleType.ID)
	ereID := createRandomEvaluationRuleEntity(t, riID, repo.ID)
	esID := createRandomEvaluationStatus(t, ereID)

	err := testQueries.UpsertLatestEvaluationStatus(context.Background(), UpsertLatestEvaluationStatusParams{
		RuleEntityID:        ereID,
		EvaluationHistoryID: esID,
		ProfileID:           profile.ID,
	})
	require.NoError(t, err)

	return proj.ID, repo, esID, evaluationTime(t, esID)
}

// TestOldestRuleEvaluationsIgnoreSessionTimeZone checks that the oldest
// evaluation time used by Reminder is the stored instant, regardless of the
// database session time zone.
func TestOldestRuleEvaluationsIgnoreSessionTimeZone(t *testing.T) {
	t.Parallel()

	_, repo, _, evalTime := createLatestEvaluation(t)
	q := queriesInSessionTimeZone(t, nonUTCTimeZone)

	byEntity, err := q.ListOldestRuleEvaluationsByEntityID(context.Background(), []uuid.UUID{repo.ID})
	require.NoError(t, err)
	require.Len(t, byEntity, 1)
	require.Truef(t, evalTime.Equal(byEntity[0].OldestLastUpdated),
		"ListOldestRuleEvaluationsByEntityID: want %s, got %s", evalTime.UTC(), byEntity[0].OldestLastUpdated.UTC())

	byRepo, err := q.ListOldestRuleEvaluationsByRepositoryId(context.Background(), []uuid.UUID{repo.ID})
	require.NoError(t, err)
	require.Len(t, byRepo, 1)
	require.Truef(t, evalTime.Equal(byRepo[0].OldestLastUpdated),
		"ListOldestRuleEvaluationsByRepositoryId: want %s, got %s", evalTime.UTC(), byRepo[0].OldestLastUpdated.UTC())
}

// TestListEvaluationHistoryIgnoresSessionTimeZone checks that the cursor and
// time range parameters of ListEvaluationHistory are compared as instants,
// regardless of the database session time zone.
func TestListEvaluationHistoryIgnoresSessionTimeZone(t *testing.T) {
	t.Parallel()

	projID, _, esID, evalTime := createLatestEvaluation(t)
	q := queriesInSessionTimeZone(t, nonUTCTimeZone)

	before := sql.NullTime{Time: evalTime.Add(-time.Minute).UTC(), Valid: true}
	after := sql.NullTime{Time: evalTime.Add(time.Minute).UTC(), Valid: true}

	tests := []struct {
		name   string
		params ListEvaluationHistoryParams
	}{
		{
			name:   "next cursor just after the evaluation",
			params: ListEvaluationHistoryParams{Next: after},
		},
		{
			name:   "prev cursor just before the evaluation",
			params: ListEvaluationHistoryParams{Prev: before},
		},
		{
			name:   "time range around the evaluation",
			params: ListEvaluationHistoryParams{Fromts: before, Tots: after},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			params := tt.params
			params.Projectid = projID
			params.Size = 5

			rows, err := q.ListEvaluationHistory(context.Background(), params)
			require.NoError(t, err)
			require.Len(t, rows, 1)
			require.Equal(t, esID, rows[0].EvaluationID)
		})
	}
}
