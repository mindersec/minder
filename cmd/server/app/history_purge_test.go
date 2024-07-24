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

	"github.com/stacklok/minder/internal/db"
	dbf "github.com/stacklok/minder/internal/db/fixtures"
)

// This test ensures that the size of these records is kept upder
// control, as the current deletion logic loads all history records
// beyond a fixed time interval in memory. It is not mandatory to keep
// the record as low as possible, but allocated resources must be
// modified accordingly.
func TestRecordSize(t *testing.T) {
	t.Parallel()
	size := unsafe.Sizeof(
		db.ListEvaluationHistoryOlderThanRow{
			ID:             uuid.Nil,
			EvaluationTime: time.Now(),
			EntityType:     int32(1),
			EntityID:       uuid.Nil,
			RuleID:         uuid.Nil,
		},
	)

	require.Equal(t, 80, int(size))
}

func TestFilterRecords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		records  []db.ListEvaluationHistoryOlderThanRow
		expected []db.ListEvaluationHistoryOlderThanRow
	}{
		{
			name: "older removed",
			records: []db.ListEvaluationHistoryOlderThanRow{
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
					entityID1,
					ruleID1,
				),
			},
			expected: []db.ListEvaluationHistoryOlderThanRow{
				makeHistoryRow(
					uuid2,
					evaluatedAt2,
					entityType,
					entityID1,
					ruleID1,
				),
			},
		},
		{
			name: "older removed bis",
			records: []db.ListEvaluationHistoryOlderThanRow{
				makeHistoryRow(
					uuid2,
					evaluatedAt2,
					entityType,
					entityID1,
					ruleID1,
				),
				makeHistoryRow(
					uuid1,
					evaluatedAt1,
					entityType,
					entityID1,
					ruleID1,
				),
			},
			expected: []db.ListEvaluationHistoryOlderThanRow{
				makeHistoryRow(
					uuid2,
					evaluatedAt2,
					entityType,
					entityID1,
					ruleID1,
				),
			},
		},
		{
			name: "all new",
			records: []db.ListEvaluationHistoryOlderThanRow{
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
			},
			expected: []db.ListEvaluationHistoryOlderThanRow{},
		},
		{
			name: "entity type",
			records: []db.ListEvaluationHistoryOlderThanRow{
				makeHistoryRow(
					uuid1,
					evaluatedAt1,
					entityType, // different entity type
					entityID1,
					ruleID1,
				),
				makeHistoryRow(
					uuid1,
					evaluatedAt1,
					int32(2), // different entity type
					entityID1,
					ruleID1,
				),
			},
			expected: []db.ListEvaluationHistoryOlderThanRow{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res := filterRecords(tt.records)
			require.Len(t, res, len(tt.expected))
			for i := 0; i < len(tt.expected); i++ {
				require.Equal(t, tt.expected[i], res[i])
			}
		})
	}
}

func TestDeleteEvaluationHistory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dbSetup dbf.DBMockBuilder
		records []db.ListEvaluationHistoryOlderThanRow
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
			records: []db.ListEvaluationHistoryOlderThanRow{
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
			records: []db.ListEvaluationHistoryOlderThanRow{
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
			records: []db.ListEvaluationHistoryOlderThanRow{
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
	entityID1    = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	entityID2    = uuid.MustParse("00000000-0000-0000-0000-000000000022")
	ruleID1      = uuid.MustParse("00000000-0000-0000-0000-000000000111")
	ruleID2      = uuid.MustParse("00000000-0000-0000-0000-000000000222")
	evaluatedAt1 = time.Now()
	evaluatedAt2 = evaluatedAt1.Add(-1 * time.Hour)
	entityType   = int32(1)
)

//nolint:unparam
func makeHistoryRow(
	id uuid.UUID,
	evaluatedAt time.Time,
	entityType int32,
	entityID uuid.UUID,
	ruleID uuid.UUID,
) db.ListEvaluationHistoryOlderThanRow {
	return db.ListEvaluationHistoryOlderThanRow{
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
