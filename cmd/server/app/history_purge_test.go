// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"errors"
	"testing"
	"time"
	"unsafe"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/db"
	dbf "github.com/mindersec/minder/internal/db/fixtures"
)

// This test ensures that the size of these records is kept upder
// control, as the current deletion logic loads all history records
// beyond a fixed time interval in memory. It is not mandatory to keep
// the record as low as possible, but allocated resources must be
// modified accordingly.
func TestRecordSize(t *testing.T) {
	t.Parallel()
	size := unsafe.Sizeof(
		db.ListEvaluationHistoryStaleRecordsRow{
			ID:             uuid.Nil,
			EvaluationTime: time.Now(),
			EntityType:     db.EntitiesArtifact,
			EntityID:       uuid.Nil,
			RuleID:         uuid.Nil,
		},
	)

	require.Equal(t, uintptr(88), size)
}

func TestPurgeLoop(t *testing.T) {
	t.Parallel()

	threshold := time.Now().AddDate(0, 0, -30)

	tests := []struct {
		name      string
		dbSetup   dbf.DBMockBuilder
		threshold time.Time
		dryRun    bool
		size      uint
		err       bool
	}{
		{
			name: "happy path",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistoryStaleRecords(
					nil,
					db.ListEvaluationHistoryStaleRecordsParams{
						Threshold: threshold,
						Size:      int32(4000000),
					},
					db.ListEvaluationHistoryStaleRecordsRow{
						EvaluationTime: time.Now(),
						ID:             uuid1,
						RuleID:         ruleID1,
						EntityType:     db.EntitiesArtifact,
						EntityID:       entityID1,
					},
				),
				withTransactionStuff(),
				withDeleteEvaluationHistoryByIDs(
					nil,
					[]uuid.UUID{
						uuid1,
					},
				),
			),
			threshold: threshold,
			size:      5,
		},
		{
			name: "dry run",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistoryStaleRecords(
					nil,
					db.ListEvaluationHistoryStaleRecordsParams{
						Threshold: threshold,
						Size:      int32(4000000),
					},
					db.ListEvaluationHistoryStaleRecordsRow{
						EvaluationTime: time.Now(),
						ID:             uuid1,
						RuleID:         ruleID1,
						EntityType:     db.EntitiesArtifact,
						EntityID:       entityID1,
					},
				),
			),
			threshold: threshold,
			dryRun:    true,
			size:      5,
		},
		{
			name: "batches",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistoryStaleRecords(
					nil,
					db.ListEvaluationHistoryStaleRecordsParams{
						Threshold: threshold,
						Size:      int32(4000000),
					},
					db.ListEvaluationHistoryStaleRecordsRow{
						EvaluationTime: time.Now(),
						ID:             uuid1,
						RuleID:         ruleID1,
						EntityType:     db.EntitiesArtifact,
						EntityID:       entityID1,
					},
					db.ListEvaluationHistoryStaleRecordsRow{
						EvaluationTime: time.Now(),
						ID:             uuid2,
						RuleID:         ruleID2,
						EntityType:     db.EntitiesArtifact,
						EntityID:       entityID2,
					},
					db.ListEvaluationHistoryStaleRecordsRow{
						EvaluationTime: time.Now(),
						ID:             uuid3,
						RuleID:         ruleID3,
						EntityType:     db.EntitiesArtifact,
						EntityID:       entityID3,
					},
				),
				withTransactionStuff(),
				withDeleteEvaluationHistoryByIDs(
					nil,
					[]uuid.UUID{
						uuid1,
						uuid2,
					},
					[]uuid.UUID{
						uuid3,
					},
				),
			),
			threshold: threshold,
			size:      2,
		},
		{
			name: "no records",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistoryStaleRecords(
					nil,
					db.ListEvaluationHistoryStaleRecordsParams{
						Threshold: threshold,
						Size:      int32(4000000),
					},
				),
			),
			threshold: threshold,
			size:      5,
		},
		{
			name: "read error",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistoryStaleRecords(
					errors.New("boom"),
					db.ListEvaluationHistoryStaleRecordsParams{
						Threshold: threshold,
						Size:      int32(4000000),
					},
				),
			),
			threshold: threshold,
			size:      5,
			err:       true,
		},
		{
			name: "write error",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistoryStaleRecords(
					nil,
					db.ListEvaluationHistoryStaleRecordsParams{
						Threshold: threshold,
						Size:      int32(4000000),
					},
					db.ListEvaluationHistoryStaleRecordsRow{
						EvaluationTime: time.Now(),
						ID:             uuid1,
						RuleID:         ruleID1,
						EntityType:     db.EntitiesArtifact,
						EntityID:       entityID1,
					},
				),
				withTransactionStuff(),
				withDeleteEvaluationHistoryByIDs(
					errors.New("boom"),
					[]uuid.UUID{
						uuid1,
					},
				),
			),
			threshold: threshold,
			size:      5,
			err:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var store db.Store
			if tt.dbSetup != nil {
				store = tt.dbSetup(ctrl)
			}

			err := purgeLoop(ctx, store, tt.threshold, tt.size, tt.dryRun, t.Logf)
			if tt.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestDeleteEvaluationHistory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dbSetup dbf.DBMockBuilder
		records []db.ListEvaluationHistoryStaleRecordsRow
		size    uint
		err     bool
	}{
		{
			name: "more records",
			dbSetup: dbf.NewDBMock(
				withTransactionStuff(),
				withDeleteEvaluationHistoryByIDs(
					nil,
					[]uuid.UUID{
						uuid1,
						uuid2,
						uuid.Nil,
						uuid.Nil,
						uuid.Nil,
					},
					[]uuid.UUID{
						uuid.Nil,
						uuid.Nil,
						uuid.Nil,
					},
				),
			),
			records: []db.ListEvaluationHistoryStaleRecordsRow{
				makeHistoryRow(
					uuid1,
					evaluatedAt1,
					entityType,
					entityID1,
					ruleID1,
				),
				makeHistoryRow(
					uuid2,
					evaluatedAt2,
					entityType,
					entityID2,
					ruleID2,
				),
				makeHistoryRow(
					uuid.Nil,
					evaluatedAt1,
					entityType,
					entityID1,
					ruleID1,
				),
				makeHistoryRow(
					uuid.Nil,
					evaluatedAt1,
					entityType,
					entityID1,
					ruleID1,
				),
				makeHistoryRow(
					uuid.Nil,
					evaluatedAt1,
					entityType,
					entityID1,
					ruleID1,
				),
				makeHistoryRow(
					uuid.Nil,
					evaluatedAt1,
					entityType,
					entityID1,
					ruleID1,
				),
				makeHistoryRow(
					uuid.Nil,
					evaluatedAt1,
					entityType,
					entityID1,
					ruleID1,
				),
				makeHistoryRow(
					uuid.Nil,
					evaluatedAt1,
					entityType,
					entityID1,
					ruleID1,
				),
			},
			size: 5,
		},
		{
			name: "fewer records",
			dbSetup: dbf.NewDBMock(
				withTransactionStuff(),
				withDeleteEvaluationHistoryByIDs(
					nil,
					[]uuid.UUID{
						uuid1,
						uuid2,
					},
				),
			),
			records: []db.ListEvaluationHistoryStaleRecordsRow{
				makeHistoryRow(
					uuid1,
					evaluatedAt1,
					entityType,
					entityID1,
					ruleID1,
				),
				makeHistoryRow(
					uuid2,
					evaluatedAt1,
					entityType,
					entityID1,
					ruleID1,
				),
			},
			size: 5,
		},
		{
			name: "transaction stops",
			dbSetup: dbf.NewDBMock(
				withTransactionStuff(),
				withDeleteEvaluationHistoryByIDs(
					errors.New("boom"),
					[]uuid.UUID{
						uuid1,
						uuid2,
					},
				),
			),
			records: []db.ListEvaluationHistoryStaleRecordsRow{
				makeHistoryRow(
					uuid1,
					evaluatedAt1,
					entityType,
					entityID1,
					ruleID1,
				),
				makeHistoryRow(
					uuid2,
					evaluatedAt1,
					entityType,
					entityID1,
					ruleID1,
				),
			},
			size: 5,
			err:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var store db.Store
			if tt.dbSetup != nil {
				store = tt.dbSetup(ctrl)
			}

			_, err := deleteEvaluationHistory(ctx, store, tt.records, tt.size)
			if tt.err {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

var (
	uuid1        = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	uuid2        = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	uuid3        = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	entityID1    = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	entityID2    = uuid.MustParse("00000000-0000-0000-0000-000000000022")
	entityID3    = uuid.MustParse("00000000-0000-0000-0000-000000000033")
	ruleID1      = uuid.MustParse("00000000-0000-0000-0000-000000000111")
	ruleID2      = uuid.MustParse("00000000-0000-0000-0000-000000000222")
	ruleID3      = uuid.MustParse("00000000-0000-0000-0000-000000000333")
	evaluatedAt1 = time.Now()
	evaluatedAt2 = evaluatedAt1.Add(-1 * time.Hour)
	entityType   = db.EntitiesRepository
)

//nolint:unparam
func makeHistoryRow(
	id uuid.UUID,
	evaluatedAt time.Time,
	entityType db.Entities,
	entityID uuid.UUID,
	ruleID uuid.UUID,
) db.ListEvaluationHistoryStaleRecordsRow {
	return db.ListEvaluationHistoryStaleRecordsRow{
		ID:             id,
		EvaluationTime: evaluatedAt,
		EntityType:     entityType,
		EntityID:       entityID,
		RuleID:         ruleID,
	}
}

func withTransactionStuff() func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			BeginTransaction().
			AnyTimes()
		mock.EXPECT().
			GetQuerierWithTransaction(gomock.Any()).
			Return(mock).
			AnyTimes()
		mock.EXPECT().
			Commit(gomock.Any()).
			AnyTimes()
		mock.EXPECT().
			Rollback(gomock.Any()).
			AnyTimes()
	}
}

func withListEvaluationHistoryStaleRecords(
	err error,
	params db.ListEvaluationHistoryStaleRecordsParams,
	records ...db.ListEvaluationHistoryStaleRecordsRow,
) func(dbf.DBMock) {
	if err != nil {
		return func(mock dbf.DBMock) {
			mock.EXPECT().
				ListEvaluationHistoryStaleRecords(gomock.Any(), params).
				Return(nil, err)
		}
	}
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			ListEvaluationHistoryStaleRecords(gomock.Any(), params).
			Return(records, err)
	}
}

func withDeleteEvaluationHistoryByIDs(
	err error,
	params ...[]uuid.UUID,
) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		if params != nil {
			calls := []any{}
			for _, ps := range params {
				call := mock.EXPECT().
					DeleteEvaluationHistoryByIDs(gomock.Any(), ps).
					Return(int64(len(ps)), err)
				calls = append(calls, call)
			}
			gomock.InOrder(calls...)
			return
		}
		mock.EXPECT().
			DeleteEvaluationHistoryByIDs(gomock.Any(), gomock.Any()).
			Return(int64(0), err)
	}
}
