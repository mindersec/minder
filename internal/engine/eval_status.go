// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package engine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/mediator/internal/db"
	evalerrors "github.com/stacklok/mediator/internal/engine/errors"
	"github.com/stacklok/mediator/internal/engine/interfaces"
	"github.com/stacklok/mediator/internal/entities"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// EvalStatusParams is a helper struct to pass parameters to createOrUpdateEvalStatus
// to avoid confusion with the parameters order. Since at the moment all our entities are bound to
// a repo and most profiles are expecting a repo, the repoID parameter is mandatory. For entities
// other than artifacts, the artifactID should be 0 which is translated to NULL in the database.
type EvalStatusParams struct {
	profileID        uuid.UUID
	repoID           uuid.UUID
	artifactID       uuid.NullUUID
	entityType       db.Entities
	ruleTypeID       uuid.UUID
	ActionsOnOff     map[string]interfaces.ActionOpt
	EvalStatusFromDb db.ListRuleEvaluationsByProfileIdRow
	EvalErr          error
	ActionsErr       evalerrors.ActionsError
}

func (e *Executor) createEvalStatusParams(
	ctx context.Context,
	inf *EntityInfoWrapper,
	profile *pb.Profile,
	rule *pb.Profile_Rule,
) (*EvalStatusParams, error) {
	// Get Profile UUID
	profileID, err := uuid.Parse(*profile.Id)
	if err != nil {
		return nil, fmt.Errorf("error parsing profile ID: %w", err)
	}

	params := &EvalStatusParams{
		profileID:  profileID,
		repoID:     uuid.MustParse(inf.OwnershipData[RepositoryIDEventKey]),
		entityType: entities.EntityTypeToDB(inf.Type),
	}

	artifactID, ok := inf.OwnershipData[ArtifactIDEventKey]
	if ok {
		params.artifactID = uuid.NullUUID{
			UUID:  uuid.MustParse(artifactID),
			Valid: true,
		}
	}

	// Prepare params for fetching the current rule evaluation from the database
	entityType := db.NullEntities{
		Entities: params.entityType,
		Valid:    true}
	entityID := uuid.NullUUID{}
	if params.entityType == db.EntitiesArtifact {
		entityID = params.artifactID
	} else if params.entityType == db.EntitiesRepository {
		entityID = uuid.NullUUID{
			UUID:  params.repoID,
			Valid: true,
		}
	}
	ruleName := sql.NullString{
		String: rule.Type,
		Valid:  true,
	}

	// Get the current rule evaluation from the database
	evalStatus, err := e.querier.GetRuleEvaluationByProfileIdAndRuleType(ctx,
		params.profileID,
		entityType,
		entityID,
		ruleName,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting rule evaluation status from db: %w", err)
	}

	// Save the current rule evaluation status to the evalParams
	params.EvalStatusFromDb = evalStatus
	return params, nil
}

func (e *Executor) createOrUpdateEvalStatus(
	ctx context.Context,
	evalParams *EvalStatusParams,
) error {
	logger := zerolog.Ctx(ctx)
	// Make sure evalParams is not nil
	if evalParams == nil {
		return fmt.Errorf("createEvalStatusParams cannot be nil")
	}

	// Check if we should skip silently
	if errors.Is(evalParams.EvalErr, evalerrors.ErrEvaluationSkipSilently) {
		logger.Debug().Msgf("silent skip of rule %s for profile %s for entity %s in repo %s",
			evalParams.ruleTypeID, evalParams.profileID, evalParams.entityType, evalParams.repoID)
		return nil
	}

	// Upsert evaluation
	id, err := e.querier.UpsertRuleEvaluations(ctx, db.UpsertRuleEvaluationsParams{
		ProfileID: evalParams.profileID,
		RepositoryID: uuid.NullUUID{
			UUID:  evalParams.repoID,
			Valid: true,
		},
		ArtifactID: evalParams.artifactID,
		Entity:     evalParams.entityType,
		RuleTypeID: evalParams.ruleTypeID,
	})

	if err != nil {
		logger.Error().Msgf("error upserting rule eval, profile %s, entity %s, repo %s: %s",
			evalParams.profileID, evalParams.entityType, evalParams.repoID, err)
		return err
	}
	// Upsert evaluation details
	_, err = e.querier.UpsertRuleDetailsEval(ctx, db.UpsertRuleDetailsEvalParams{
		RuleEvalID: id,
		Status:     errorAsEvalStatus(evalParams.EvalErr),
		Details:    errorAsEvalDetails(evalParams.EvalErr),
	})

	if err != nil {
		logger.Error().Msgf("error upserting rule eval details, profile %s, entity %s, repo %s: %s",
			evalParams.profileID, evalParams.entityType, evalParams.repoID, err)
		return err
	}
	// Upsert remediation details
	_, err = e.querier.UpsertRuleDetailsRemediate(ctx, db.UpsertRuleDetailsRemediateParams{
		RuleEvalID: id,
		Status:     errorAsRemediationStatus(evalParams.ActionsErr.RemediateErr),
		Details:    errorAsActionDetails(evalParams.ActionsErr.RemediateErr),
	})
	if err != nil {
		logger.Error().Msgf("error upserting rule remediation details, profile %s, entity %s, repo %s: %s",
			evalParams.profileID, evalParams.entityType, evalParams.repoID, err)
	}
	// Upsert alert details
	_, err = e.querier.UpsertRuleDetailsAlert(ctx, db.UpsertRuleDetailsAlertParams{
		RuleEvalID: id,
		Status:     errorAsAlertStatus(evalParams.ActionsErr.AlertErr),
		Details:    errorAsActionDetails(evalParams.ActionsErr.AlertErr),
	})
	if err != nil {
		logger.Error().Msgf("error upserting rule alert details, profile %s, entity %s, repo %s: %s",
			evalParams.profileID, evalParams.entityType, evalParams.repoID, err)
	}
	return err
}

func errorAsEvalStatus(err error) db.EvalStatusTypes {
	if errors.Is(err, evalerrors.ErrEvaluationFailed) {
		return db.EvalStatusTypesFailure
	} else if errors.Is(err, evalerrors.ErrEvaluationSkipped) {
		return db.EvalStatusTypesSkipped
	} else if err != nil {
		return db.EvalStatusTypesError
	}
	return db.EvalStatusTypesSuccess
}

func errorAsEvalDetails(err error) string {
	if err != nil {
		return err.Error()
	}

	return ""
}

func errorAsRemediationStatus(err error) db.RemediationStatusTypes {
	if err == nil {
		return db.RemediationStatusTypesSuccess
	}

	switch err != nil {
	case errors.Is(err, evalerrors.ErrActionFailed):
		return db.RemediationStatusTypesFailure
	case errors.Is(err, evalerrors.ErrActionSkipped):
		return db.RemediationStatusTypesSkipped
	case errors.Is(err, evalerrors.ErrActionNotAvailable):
		return db.RemediationStatusTypesNotAvailable
	}
	return db.RemediationStatusTypesError
}

func errorAsAlertStatus(err error) db.AlertStatusTypes {
	if err == nil {
		return db.AlertStatusTypesOn
	}

	switch err != nil {
	case errors.Is(err, evalerrors.ErrActionFailed):
		return db.AlertStatusTypesError
	case errors.Is(err, evalerrors.ErrActionSkipped):
		return db.AlertStatusTypesSkipped
	case errors.Is(err, evalerrors.ErrActionNotAvailable):
		return db.AlertStatusTypesNotAvailable
	}
	return db.AlertStatusTypesError
}

func errorAsActionDetails(err error) string {
	if evalerrors.IsActionFatalError(err) {
		return err.Error()
	}

	return ""
}
