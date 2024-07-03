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

// Package history contains logic for tracking evaluation history
package history

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	evalerrors "github.com/stacklok/minder/internal/engine/errors"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// EvaluationHistoryService contains methods to add/query data in the history table.
type EvaluationHistoryService interface {
	// StoreEvaluationStatus stores the result of this evaluation in the history table.
	// Returns the UUID of the evaluation status.
	StoreEvaluationStatus(
		ctx context.Context,
		qtx db.Querier,
		ruleID uuid.UUID,
		entityType db.Entities,
		entityID uuid.UUID,
		evalError error,
	) (uuid.UUID, error)
	// ListEvaluationHistory returns a list of evaluations stored
	// in the history table.
	ListEvaluationHistory(
		ctx context.Context,
		qtx db.Querier,
		cursor *ListEvaluationCursor,
		size uint64,
		filter ListEvaluationFilter,
	) (*ListEvaluationHistoryResult, error)
}

// NewEvaluationHistoryService creates a new instance of EvaluationHistoryService
func NewEvaluationHistoryService() EvaluationHistoryService {
	return &evaluationHistoryService{}
}

type evaluationHistoryService struct{}

func (e *evaluationHistoryService) StoreEvaluationStatus(
	ctx context.Context,
	qtx db.Querier,
	ruleID uuid.UUID,
	entityType db.Entities,
	entityID uuid.UUID,
	evalError error,
) (uuid.UUID, error) {
	var ruleEntityID, evaluationID uuid.UUID
	status := evalerrors.ErrorAsEvalStatus(evalError)
	details := evalerrors.ErrorAsEvalDetails(evalError)

	params, err := paramsFromEntity(ruleID, entityID, entityType)
	if err != nil {
		return uuid.Nil, err
	}

	// find the latest record for this rule/entity pair
	latestRecord, err := qtx.GetLatestEvalStateForRuleEntity(ctx,
		db.GetLatestEvalStateForRuleEntityParams{
			RuleID:        params.RuleID,
			RepositoryID:  params.RepositoryID,
			PullRequestID: params.PullRequestID,
			ArtifactID:    params.ArtifactID,
		},
	)
	if err != nil {
		// if we find nothing, create a new rule/entity record
		if errors.Is(err, sql.ErrNoRows) {
			ruleEntityID, err = qtx.InsertEvaluationRuleEntity(ctx,
				db.InsertEvaluationRuleEntityParams{
					RuleID:        params.RuleID,
					RepositoryID:  params.RepositoryID,
					PullRequestID: params.PullRequestID,
					ArtifactID:    params.ArtifactID,
				},
			)
			if err != nil {
				return uuid.Nil, fmt.Errorf("error while creating new rule/entity in database: %w", err)
			}
		} else {
			return uuid.Nil, fmt.Errorf("error while querying DB: %w", err)
		}
	} else {
		ruleEntityID = latestRecord.RuleEntityID
		evaluationID = latestRecord.ID
	}

	previousDetails := latestRecord.Details
	previousStatus := latestRecord.Status

	if evaluationID == uuid.Nil || previousDetails != details || previousStatus != status {
		// if there is no prior state for this rule/entity, or the previous state
		// differs from the current one, create a new status record.
		if evaluationID, err = e.createNewStatus(ctx, qtx, ruleEntityID, status, details); err != nil {
			return uuid.Nil, fmt.Errorf("error while creating new evaluation status for rule/entity %s: %w", ruleEntityID, err)
		}
	} else {
		if err = e.updateExistingStatus(ctx, qtx, entityID, latestRecord.EvaluationTimes); err != nil {
			return uuid.Nil, fmt.Errorf("error while updating existing evaluation status for rule/entity %s: %w", ruleEntityID, err)
		}
	}

	return evaluationID, nil
}

func (_ *evaluationHistoryService) createNewStatus(
	ctx context.Context,
	qtx db.Querier,
	ruleEntityID uuid.UUID,
	status db.EvalStatusTypes,
	details string,
) (uuid.UUID, error) {
	newEvaluationID, err := qtx.InsertEvaluationStatus(ctx,
		db.InsertEvaluationStatusParams{
			RuleEntityID: ruleEntityID,
			Status:       status,
			Details:      details,
		},
	)
	if err != nil {
		return uuid.Nil, err
	}

	// mark this as the latest status for this rule/entity
	err = qtx.UpsertLatestEvaluationStatus(ctx,
		db.UpsertLatestEvaluationStatusParams{
			RuleEntityID:        ruleEntityID,
			EvaluationHistoryID: newEvaluationID,
		},
	)
	if err != nil {
		return uuid.Nil, err
	}

	return newEvaluationID, err
}

func (_ *evaluationHistoryService) updateExistingStatus(
	ctx context.Context,
	qtx db.Querier,
	evaluationID uuid.UUID,
	times []time.Time,
) error {
	// if the status is repeated, then just append the current timestamp to it
	times = append(times, time.Now())
	return qtx.UpdateEvaluationTimes(ctx, db.UpdateEvaluationTimesParams{
		EvaluationTimes: times,
		ID:              evaluationID,
	})
}

func paramsFromEntity(
	ruleID uuid.UUID,
	entityID uuid.UUID,
	entityType db.Entities,
) (*ruleEntityParams, error) {
	params := ruleEntityParams{RuleID: ruleID}

	nullableEntityID := uuid.NullUUID{
		UUID:  entityID,
		Valid: true,
	}

	switch entityType {
	case db.EntitiesRepository:
		params.RepositoryID = nullableEntityID
	case db.EntitiesPullRequest:
		params.PullRequestID = nullableEntityID
	case db.EntitiesArtifact:
		params.ArtifactID = nullableEntityID
	case db.EntitiesBuildEnvironment:
	default:
		return nil, fmt.Errorf("unknown entity %s", entityType)
	}
	return &params, nil
}

type ruleEntityParams struct {
	RuleID        uuid.UUID
	RepositoryID  uuid.NullUUID
	ArtifactID    uuid.NullUUID
	PullRequestID uuid.NullUUID
}

func (_ *evaluationHistoryService) ListEvaluationHistory(
	ctx context.Context,
	qtx db.Querier,
	cursor *ListEvaluationCursor,
	size uint64,
	filter ListEvaluationFilter,
) (*ListEvaluationHistoryResult, error) {
	params := db.ListEvaluationHistoryParams{
		Size: int32(size),
	}
	if string(cursor.Direction) == "next" {
		params.Next = sql.NullTime{
			Time:  cursor.Time,
			Valid: true,
		}
	}
	if string(cursor.Direction) == "prev" {
		params.Prev = sql.NullTime{
			Time:  cursor.Time,
			Valid: true,
		}
	}
	if len(filter.IncludedEntityNames()) != 0 {
		params.Entitynames = filter.IncludedEntityNames()
	}
	if len(filter.IncludedProfileNames()) != 0 {
		params.Profilenames = filter.IncludedProfileNames()
	}

	rows, err := qtx.ListEvaluationHistory(ctx, params)
	if err != nil {
		return nil, errors.New("internal error")
	}

	result := &ListEvaluationHistoryResult{
		Data: rows,
	}
	if len(rows) > 0 {
		newest := rows[0]
		oldest := rows[len(rows)-1]

		result.Next = []byte(fmt.Sprintf("+%d", oldest.EvaluatedAt.UnixMicro()))
		result.Prev = []byte(fmt.Sprintf("-%d", newest.EvaluatedAt.UnixMicro()))
	}

	return result, nil
}
