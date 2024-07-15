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

package history

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/stacklok/minder/internal/db"
	dbf "github.com/stacklok/minder/internal/db/fixtures"
)

func TestStoreEvaluationStatus(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name          string
		EntityType    db.Entities
		DBSetup       dbf.DBMockBuilder
		ExpectedError string
	}{
		{
			Name:          "StoreEvaluationStatus rejects invalid entity type",
			EntityType:    "I'm a little teapot",
			ExpectedError: "unknown entity",
		},
		{
			Name:          "StoreEvaluationStatus returns error when unable to query previous status",
			EntityType:    db.EntitiesArtifact,
			DBSetup:       dbf.NewDBMock(withGetLatestEval(emptyLatestResult, errTest)),
			ExpectedError: "error while querying DB",
		},
		{
			Name:       "StoreEvaluationStatus returns error when unable to create new rule/entity",
			EntityType: db.EntitiesPullRequest,
			DBSetup: dbf.NewDBMock(
				withGetLatestEval(emptyLatestResult, sql.ErrNoRows),
				withInsertEvaluationRuleEntity(uuid.Nil, errTest),
			),
			ExpectedError: "error while creating new rule/entity in database",
		},
		{
			Name:       "StoreEvaluationStatus returns error when unable to create new evaluation status",
			EntityType: db.EntitiesRepository,
			DBSetup: dbf.NewDBMock(
				withGetLatestEval(emptyLatestResult, sql.ErrNoRows),
				withInsertEvaluationRuleEntity(ruleEntityID, nil),
				withInsertEvaluationStatus(uuid.Nil, errTest),
			),
			ExpectedError: "error while creating new evaluation status for rule/entity",
		},
		{
			Name:       "StoreEvaluationStatus returns error when unable to set latest status",
			EntityType: db.EntitiesRepository,
			DBSetup: dbf.NewDBMock(
				withGetLatestEval(emptyLatestResult, sql.ErrNoRows),
				withInsertEvaluationRuleEntity(ruleEntityID, nil),
				withInsertEvaluationStatus(evaluationID, nil),
				withUpsertLatestEvaluationStatus(errTest),
			),
			ExpectedError: "error while creating new evaluation status for rule/entity",
		},
		{
			Name:       "StoreEvaluationStatus creates new status for new rule/entity",
			EntityType: db.EntitiesRepository,
			DBSetup: dbf.NewDBMock(
				withGetLatestEval(emptyLatestResult, sql.ErrNoRows),
				withInsertEvaluationRuleEntity(ruleEntityID, nil),
				withInsertEvaluationStatus(evaluationID, nil),
				withUpsertLatestEvaluationStatus(nil),
			),
		},
		{
			Name:       "StoreEvaluationStatus creates new status for state change",
			EntityType: db.EntitiesRepository,
			DBSetup: dbf.NewDBMock(
				withGetLatestEval(existingState, nil),
				withInsertEvaluationStatus(evaluationID, nil),
				withUpsertLatestEvaluationStatus(nil),
			),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var store db.Store
			if scenario.DBSetup != nil {
				store = scenario.DBSetup(ctrl)
			}

			service := NewEvaluationHistoryService()
			id, err := service.StoreEvaluationStatus(ctx, store, ruleID, scenario.EntityType, entityID, errTest)
			if scenario.ExpectedError == "" {
				require.Equal(t, evaluationID, id)
				require.NoError(t, err)
			} else {
				require.Equal(t, uuid.Nil, id)
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}

func TestListEvaluationHistory(t *testing.T) {
	t.Parallel()

	uuid1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	uuid2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	uuid3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	epoch := time.UnixMicro(0).UTC()
	evaluatedAt1 := time.Now()
	evaluatedAt2 := evaluatedAt1.Add(-1 * time.Second)
	evaluatedAt3 := evaluatedAt1.Add(-2 * time.Second)
	entityType := []byte("repository")

	remediation := db.NullRemediationStatusTypes{
		RemediationStatusTypes: db.RemediationStatusTypesSuccess,
		Valid:                  true,
	}
	alert := db.NullAlertStatusTypes{
		AlertStatusTypes: db.AlertStatusTypesOn,
		Valid:            true,
	}

	tests := []struct {
		name    string
		dbSetup dbf.DBMockBuilder
		cursor  *ListEvaluationCursor
		size    uint64
		filter  ListEvaluationFilter
		checkf  func(*testing.T, *ListEvaluationHistoryResult)
		err     bool
	}{
		{
			name: "records",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(nil, nil,
					makeHistoryRow(
						uuid1,
						evaluatedAt1,
						entityType,
						remediation,
						alert,
					),
					makeHistoryRow(
						uuid2,
						evaluatedAt2,
						entityType,
						remediation,
						alert,
					),
					makeHistoryRow(
						uuid3,
						evaluatedAt3,
						entityType,
						remediation,
						alert,
					),
				),
			),
			checkf: func(t *testing.T, rows *ListEvaluationHistoryResult) {
				t.Helper()

				require.NotNil(t, rows)
				require.Len(t, rows.Data, 3)

				// database order is maintained
				item1 := rows.Data[0]
				require.Equal(t, uuid1, item1.EvaluationID)
				require.Equal(t, evaluatedAt1, item1.EvaluatedAt)
				require.Equal(t, uuid1, item1.EntityID)

				item2 := rows.Data[1]
				require.Equal(t, uuid2, item2.EvaluationID)
				require.Equal(t, evaluatedAt2, item2.EvaluatedAt)
				require.Equal(t, uuid2, item2.EntityID)

				item3 := rows.Data[2]
				require.Equal(t, uuid3, item3.EvaluationID)
				require.Equal(t, evaluatedAt3, item3.EvaluatedAt)
				require.Equal(t, uuid3, item3.EntityID)
			},
		},

		// cursor
		{
			name: "cursor next",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size: 0,
						Next: sql.NullTime{
							Time:  epoch.Add(1 * time.Hour),
							Valid: true,
						},
					},
					nil,
				),
			),
			cursor: &ListEvaluationCursor{
				Time:      epoch.Add(1 * time.Hour),
				Direction: Next,
			},
		},
		{
			name: "cursor prev",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size: 0,
						Prev: sql.NullTime{
							Time:  epoch.Add(1 * time.Hour),
							Valid: true,
						},
					},
					nil,
				),
			),
			cursor: &ListEvaluationCursor{
				Time:      epoch.Add(1 * time.Hour),
				Direction: Prev,
			},
		},
		{
			name: "cursor size",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size: 50,
					},
					nil,
				),
			),
			size: 50,
		},

		// filter entity types
		{
			name: "included entity types",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Entitytypes: []db.Entities{
							db.EntitiesRepository,
							db.EntitiesBuildEnvironment,
							db.EntitiesArtifact,
							db.EntitiesPullRequest,
						},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				includedEntityTypes: []string{
					"repository",
					"build_environment",
					"artifact",
					"pull_request",
				},
			},
		},
		{
			name: "included entity types bad string",
			filter: &listEvaluationFilter{
				includedEntityTypes: []string{"foo"},
			},
			err: true,
		},
		{
			name: "excluded entity types",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Notentitytypes: []db.Entities{
							db.EntitiesRepository,
						},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				excludedEntityTypes: []string{"repository"},
			},
		},
		{
			name: "excluded entity types bad string",
			filter: &listEvaluationFilter{
				excludedEntityTypes: []string{"foo"},
			},
			err: true,
		},

		// filter entity names
		{
			name: "included entity names",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size:        0,
						Entitynames: []string{"foo"},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				includedEntityNames: []string{"foo"},
			},
		},
		{
			name: "excluded entity names",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size:           0,
						Notentitynames: []string{"foo"},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				excludedEntityNames: []string{"foo"},
			},
		},

		// filter profile names
		{
			name: "included profile names",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size:         0,
						Profilenames: []string{"foo"},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				includedProfileNames: []string{"foo"},
			},
		},
		{
			name: "excluded profile names",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size:            0,
						Notprofilenames: []string{"foo"},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				excludedProfileNames: []string{"foo"},
			},
		},

		// filter remediations
		{
			name: "included remediations",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size: 0,
						Remediations: []db.RemediationStatusTypes{
							db.RemediationStatusTypesSuccess,
							db.RemediationStatusTypesFailure,
							db.RemediationStatusTypesError,
							db.RemediationStatusTypesSkipped,
							db.RemediationStatusTypesNotAvailable,
							db.RemediationStatusTypesPending,
						},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				includedRemediations: []string{
					"success",
					"failure",
					"error",
					"skipped",
					"not_available",
					"pending",
				},
			},
		},
		{
			name: "included remediations bad string",
			filter: &listEvaluationFilter{
				includedRemediations: []string{"foo"},
			},
			err: true,
		},
		{
			name: "excluded remediations",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size: 0,
						Notremediations: []db.RemediationStatusTypes{
							db.RemediationStatusTypesSuccess,
						},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				excludedRemediations: []string{"success"},
			},
		},
		{
			name: "excluded remediations bad string",
			filter: &listEvaluationFilter{
				excludedRemediations: []string{"foo"},
			},
			err: true,
		},

		// filter alerts
		{
			name: "included alerts",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size: 0,
						Alerts: []db.AlertStatusTypes{
							db.AlertStatusTypesOn,
							db.AlertStatusTypesOff,
							db.AlertStatusTypesError,
							db.AlertStatusTypesSkipped,
							db.AlertStatusTypesNotAvailable,
						},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				includedAlerts: []string{
					"on",
					"off",
					"error",
					"skipped",
					"not_available",
				},
			},
		},
		{
			name: "included alerts bad string",
			filter: &listEvaluationFilter{
				includedAlerts: []string{"foo"},
			},
			err: true,
		},
		{
			name: "excluded alerts",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size: 0,
						Notalerts: []db.AlertStatusTypes{
							db.AlertStatusTypesOn,
						},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				excludedAlerts: []string{"on"},
			},
		},
		{
			name: "excluded alerts bad string",
			filter: &listEvaluationFilter{
				excludedAlerts: []string{"foo"},
			},
			err: true,
		},

		// filter statuses
		{
			name: "included statuses",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size: 0,
						Statuses: []db.EvalStatusTypes{
							db.EvalStatusTypesSuccess,
							db.EvalStatusTypesFailure,
							db.EvalStatusTypesError,
							db.EvalStatusTypesSkipped,
							db.EvalStatusTypesPending,
						},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				includedStatuses: []string{
					"success",
					"failure",
					"error",
					"skipped",
					"pending",
				},
			},
		},
		{
			name: "included statuses bad string",
			filter: &listEvaluationFilter{
				includedStatuses: []string{"foo"},
			},
			err: true,
		},
		{
			name: "excluded statuses",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size: 0,
						Notstatuses: []db.EvalStatusTypes{
							db.EvalStatusTypesSuccess,
						},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				excludedStatuses: []string{"success"},
			},
		},
		{
			name: "excluded statuses bad string",
			filter: &listEvaluationFilter{
				excludedStatuses: []string{"foo"},
			},
			err: true,
		},

		// filter on time range
		{
			name: "time range from to",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size: 0,
						Fromts: sql.NullTime{
							Time:  epoch,
							Valid: true,
						},
						Tots: sql.NullTime{
							Time:  epoch,
							Valid: true,
						},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				from: &epoch,
				to:   &epoch,
			},
		},
		{
			name: "time range from",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size: 0,
						Fromts: sql.NullTime{
							Time:  epoch,
							Valid: true,
						},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				from: &epoch,
			},
		},
		{
			name: "time range from",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(
					&db.ListEvaluationHistoryParams{
						Size: 0,
						Tots: sql.NullTime{
							Time:  epoch,
							Valid: true,
						},
					},
					nil,
				),
			),
			filter: &listEvaluationFilter{
				to: &epoch,
			},
		},

		// errors
		{
			name: "db failure",
			dbSetup: dbf.NewDBMock(
				withListEvaluationHistory(nil, errors.New("whoops")),
			),
			err: true,
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

			service := NewEvaluationHistoryService()
			res, err := service.ListEvaluationHistory(ctx, store, tt.cursor, tt.size, tt.filter)
			if tt.err {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.checkf != nil {
				tt.checkf(t, res)
			}
		})
	}
}

func makeHistoryRow(
	id uuid.UUID,
	evaluatedAt time.Time,
	entityType interface{},
	status db.NullRemediationStatusTypes,
	alert db.NullAlertStatusTypes,
) db.ListEvaluationHistoryRow {
	return db.ListEvaluationHistoryRow{
		EvaluationID: id,
		EvaluatedAt:  evaluatedAt,
		EntityType:   entityType,
		EntityID:     id,
		RepoOwner: sql.NullString{
			Valid:  true,
			String: "stacklok",
		},
		RepoName: sql.NullString{
			Valid:  true,
			String: "minder",
		},
		PrNumber: sql.NullInt64{
			Valid: true,
			Int64: 12345,
		},
		ArtifactName: sql.NullString{
			Valid:  true,
			String: "artifact1",
		},
		// EntityName:        "repo1",
		RuleType:          "rule_type",
		RuleName:          "rule_name",
		ProfileName:       "profile_name",
		EvaluationStatus:  db.EvalStatusTypesSuccess,
		EvaluationDetails: "",
		RemediationStatus: status,
		RemediationDetails: sql.NullString{
			String: "",
			Valid:  true,
		},
		AlertStatus: alert,
		AlertDetails: sql.NullString{
			String: "",
			Valid:  true,
		},
	}
}

var (
	ruleID       = uuid.New()
	entityID     = uuid.New()
	ruleEntityID = uuid.New()
	evaluationID = uuid.New()

	emptyLatestResult = db.EvaluationStatus{}
	existingState     = db.EvaluationStatus{
		ID:           evaluationID,
		RuleEntityID: ruleEntityID,
		Status:       db.EvalStatusTypesError,
		Details:      errTest.Error(),
	}
	errTest = errors.New("oh no")
)

func withGetLatestEval(result db.EvaluationStatus, err error) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			GetLatestEvalStateForRuleEntity(gomock.Any(), gomock.Any()).
			Return(result, err)
	}
}

func withInsertEvaluationRuleEntity(id uuid.UUID, err error) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			InsertEvaluationRuleEntity(gomock.Any(), gomock.Any()).
			Return(id, err)
	}
}

func withInsertEvaluationStatus(id uuid.UUID, err error) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			InsertEvaluationStatus(gomock.Any(), gomock.Any()).
			Return(id, err)
	}
}

func withUpsertLatestEvaluationStatus(err error) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			UpsertLatestEvaluationStatus(gomock.Any(), gomock.Any()).
			Return(err)
	}
}

func withListEvaluationHistory(
	params *db.ListEvaluationHistoryParams,
	err error,
	records ...db.ListEvaluationHistoryRow,
) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		if params != nil {
			mock.EXPECT().
				ListEvaluationHistory(gomock.Any(), *params).
				Return(records, err)
			return
		}
		mock.EXPECT().
			ListEvaluationHistory(gomock.Any(), gomock.Any()).
			Return(records, err)

	}
}
