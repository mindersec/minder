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

// EvaluationHistoryService contains methods to access the eval history log
type EvaluationHistoryService interface {
	StoreEvaluationStatus(
		ctx context.Context,
		qtx db.Querier,
		ruleID uuid.UUID,
		entityType db.Entities,
		entityID uuid.UUID,
		evalError error,
	) error
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
) error {
	var ruleEntityID, evaluationID uuid.UUID
	status := evalerrors.ErrorAsEvalStatus(evalError)
	details := evalerrors.ErrorAsEvalDetails(evalError)

	params, err := paramsFromEntity(ruleID, entityID, entityType)
	if err != nil {
		return err
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
				return fmt.Errorf("error while creating new rule/entity in database: %w", err)
			}
		} else {
			return fmt.Errorf("error while querying DB: %w", err)
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
		if err = e.createNewStatus(ctx, qtx, ruleEntityID, status, details); err != nil {
			return fmt.Errorf("error while creating new evaluation status for rule/entity %s: %w", ruleEntityID, err)
		}
	} else {
		if err = e.updateExistingStatus(ctx, qtx, entityID, latestRecord.EvaluationTimes); err != nil {
			return fmt.Errorf("error while updating existing evaluation status for rule/entity %s: %w", ruleEntityID, err)
		}
	}

	return nil
}

func (_ *evaluationHistoryService) createNewStatus(
	ctx context.Context,
	qtx db.Querier,
	ruleEntityID uuid.UUID,
	status db.EvalStatusTypes,
	details string,
) error {
	newEvaluationID, err := qtx.InsertEvaluationStatus(ctx,
		db.InsertEvaluationStatusParams{
			RuleEntityID: ruleEntityID,
			Status:       status,
			Details:      details,
		},
	)
	if err != nil {
		return err
	}

	// mark this as the latest status for this rule/entity
	return qtx.UpsertLatestEvaluationStatus(ctx,
		db.UpsertLatestEvaluationStatusParams{
			RuleEntityID:        ruleEntityID,
			EvaluationHistoryID: newEvaluationID,
		},
	)
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
