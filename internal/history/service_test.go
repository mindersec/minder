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

package history_test

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
	engerr "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/history"
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
			Name:       "StoreEvaluationStatus returns error when unable to update status with timestamp",
			EntityType: db.EntitiesRepository,
			DBSetup: dbf.NewDBMock(
				withGetLatestEval(sameState, nil),
				withUpdateEvaluationTimes(errTest),
			),
			ExpectedError: "error while updating existing evaluation status for rule/entity",
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
				withGetLatestEval(differentState, nil),
				withInsertEvaluationStatus(evaluationID, nil),
				withUpsertLatestEvaluationStatus(nil),
			),
		},
		{
			Name:       "StoreEvaluationStatus creates new status when status is the same, but details differ",
			EntityType: db.EntitiesRepository,
			DBSetup: dbf.NewDBMock(
				withGetLatestEval(differentDetails, nil),
				withInsertEvaluationStatus(evaluationID, nil),
				withUpsertLatestEvaluationStatus(nil),
			),
		},
		{
			Name:       "StoreEvaluationStatus adds timestamp when state does not change",
			EntityType: db.EntitiesRepository,
			DBSetup: dbf.NewDBMock(
				withGetLatestEval(sameState, nil),
				withUpdateEvaluationTimes(nil),
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

			service := history.NewEvaluationHistoryService()
			err := service.StoreEvaluationStatus(ctx, store, ruleID, scenario.EntityType, entityID, errTest)
			if scenario.ExpectedError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}

var (
	ruleID       = uuid.New()
	entityID     = uuid.New()
	ruleEntityID = uuid.New()
	evaluationID = uuid.New()

	emptyLatestResult = db.EvaluationStatus{}
	sameState         = db.EvaluationStatus{
		ID:              evaluationID,
		RuleEntityID:    ruleEntityID,
		Status:          db.EvalStatusTypesError,
		Details:         errTest.Error(),
		EvaluationTimes: []time.Time{time.Now()},
	}
	differentDetails = db.EvaluationStatus{
		ID:              evaluationID,
		RuleEntityID:    ruleEntityID,
		Status:          db.EvalStatusTypesError,
		Details:         "something went wrong",
		EvaluationTimes: []time.Time{time.Now()},
	}
	differentState = db.EvaluationStatus{
		ID:              evaluationID,
		RuleEntityID:    ruleEntityID,
		Status:          db.EvalStatusTypesSkipped,
		Details:         engerr.ErrEvaluationSkipped.Error(),
		EvaluationTimes: []time.Time{time.Now()},
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

func withUpdateEvaluationTimes(err error) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			UpdateEvaluationTimes(gomock.Any(), gomock.Any()).
			Return(err)
	}
}

func withUpsertLatestEvaluationStatus(err error) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			UpsertLatestEvaluationStatus(gomock.Any(), gomock.Any()).
			Return(err)
	}
}
