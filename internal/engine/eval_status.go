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
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"

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
	profileID      uuid.UUID
	repoID         uuid.UUID
	artifactID     uuid.NullUUID
	ruleTypeEntity db.Entities
	ruleTypeID     uuid.UUID
	Actions        map[string]interfaces.ActionOpt
	EvalErr        error
	ActionsErr     *evalerrors.ActionsError

	// TODO: implement storing existing db status here
}

func (e *Executor) createEvalStatusParams(
	ctx context.Context,
	inf *EntityInfoWrapper,
	profile *pb.Profile,
) (*EvalStatusParams, error) {
	// Get Profile UUID
	profileID, err := uuid.Parse(*profile.Id)
	if err != nil {
		return nil, fmt.Errorf("error parsing profile ID: %w", err)
	}

	params := &EvalStatusParams{
		profileID:      profileID,
		repoID:         uuid.MustParse(inf.OwnershipData[RepositoryIDEventKey]),
		ruleTypeEntity: entities.EntityTypeToDB(inf.Type),
	}

	artifactID, ok := inf.OwnershipData[ArtifactIDEventKey]
	if ok {
		params.artifactID = uuid.NullUUID{
			UUID:  uuid.MustParse(artifactID),
			Valid: true,
		}
	}

	// TODO: implement storing existing db status here
	_ = ctx
	_ = e
	return params, nil
}

func (e *Executor) createOrUpdateEvalStatus(
	ctx context.Context,
	evalParams *EvalStatusParams,
) error {
	// Make sure evalParams is not nil
	if evalParams == nil {
		return fmt.Errorf("createEvalStatusParams cannot be nil")
	}

	// Check if we should skip silently
	if errors.Is(evalParams.EvalErr, evalerrors.ErrEvaluationSkipSilently) {
		log.Printf("silent skip of rule %s for profile %s for entity %s in repo %s",
			evalParams.ruleTypeID, evalParams.profileID, evalParams.ruleTypeEntity, evalParams.repoID)
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
		Entity:     evalParams.ruleTypeEntity,
		RuleTypeID: evalParams.ruleTypeID,
	})

	if err != nil {
		log.Printf(
			"error upserting rule eval, profile %s, entity %s, repo %s: %s",
			evalParams.profileID,
			evalParams.ruleTypeEntity,
			evalParams.repoID,
			err,
		)
		return err
	}
	// Upsert evaluation details
	_, err = e.querier.UpsertRuleDetailsEval(ctx, db.UpsertRuleDetailsEvalParams{
		RuleEvalID: id,
		Status:     errorAsEvalStatus(evalParams.EvalErr),
		Details:    errorAsEvalDetails(evalParams.EvalErr),
	})

	if err != nil {
		log.Printf(
			"error upserting rule eval details, profile %s, entity %s, repo %s: %s\"",
			evalParams.profileID,
			evalParams.ruleTypeEntity,
			evalParams.repoID,
			err,
		)
		return err
	}
	// Upsert remediation details
	_, err = e.querier.UpsertRuleDetailsRemediate(ctx, db.UpsertRuleDetailsRemediateParams{
		RuleEvalID: id,
		Status:     errorAsRemediationStatus(evalParams.ActionsErr.RemediateErr),
		Details:    errorAsActionDetails(evalParams.ActionsErr.RemediateErr),
	})
	if err != nil {
		log.Printf(
			"error upserting rule remediation details, profile %s, entity %s, repo %s: %s",
			evalParams.profileID,
			evalParams.ruleTypeEntity,
			evalParams.repoID,
			err,
		)
	}
	// Upsert alert details
	_, err = e.querier.UpsertRuleDetailsAlert(ctx, db.UpsertRuleDetailsAlertParams{
		RuleEvalID: id,
		Status:     errorAsAlertStatus(evalParams.ActionsErr.AlertErr),
		Details:    errorAsActionDetails(evalParams.ActionsErr.AlertErr),
	})
	if err != nil {
		log.Printf(
			"error upserting rule alert details, profile %s, entity %s, repo %s: %s",
			evalParams.profileID,
			evalParams.ruleTypeEntity,
			evalParams.repoID,
			err,
		)
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
