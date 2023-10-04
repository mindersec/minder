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
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/stacklok/mediator/internal/db"
	evalerrors "github.com/stacklok/mediator/internal/engine/errors"
)

// createOrUpdateEvalStatusParams is a helper struct to pass parameters to createOrUpdateEvalStatus
// to avoid confusion with the parameters order. Since at the moment all our entities are bound to
// a repo and most policies are expecting a repo, the repoID parameter is mandatory. For entities
// other than artifacts, the artifactID should be 0 which is translated to NULL in the database.
type createOrUpdateEvalStatusParams struct {
	policyID       uuid.UUID
	repoID         uuid.UUID
	artifactID     *uuid.UUID
	ruleTypeEntity db.Entities
	ruleTypeID     uuid.UUID
	evalErr        error
	remediateErr   error
}

func (e *Executor) createOrUpdateEvalStatus(
	ctx context.Context,
	params *createOrUpdateEvalStatusParams,
) error {
	if params == nil {
		return fmt.Errorf("createOrUpdateEvalStatusParams cannot be nil")
	}

	if errors.Is(params.evalErr, evalerrors.ErrEvaluationSkipSilently) {
		log.Printf("silent skip of rule %s for policy %s for entity %s in repo %s",
			params.ruleTypeID, params.policyID, params.ruleTypeEntity, params.repoID)
		return nil
	}

	var sqlArtifactID uuid.NullUUID
	if params.artifactID != nil {
		sqlArtifactID = uuid.NullUUID{
			UUID:  *params.artifactID,
			Valid: true,
		}
	}

	err := e.querier.UpsertRuleEvaluationStatus(ctx, db.UpsertRuleEvaluationStatusParams{
		PolicyID: params.policyID,
		RepositoryID: uuid.NullUUID{
			UUID:  params.repoID,
			Valid: true,
		},
		ArtifactID:             sqlArtifactID,
		Entity:                 params.ruleTypeEntity,
		RuleTypeID:             params.ruleTypeID,
		EvalStatus:             errorAsEvalStatus(params.evalErr),
		EvalDetails:            errorAsEvalDetails(params.evalErr),
		RemediationStatus:      errorAsRemediationStatus(params.remediateErr),
		RemediationDetails:     errorAsRemediationDetails(params.remediateErr),
		RemediationLastUpdated: setRemediationLastUpdated(params.remediateErr),
	})
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
	case errors.Is(err, evalerrors.ErrRemediateFailed):
		return db.RemediationStatusTypesFailure
	case errors.Is(err, evalerrors.ErrRemediationSkipped):
		return db.RemediationStatusTypesSkipped
	case errors.Is(err, evalerrors.ErrRemediationNotAvailable):
		return db.RemediationStatusTypesNotAvailable
	}

	return db.RemediationStatusTypesError
}

func errorAsRemediationDetails(err error) string {
	if evalerrors.IsRemediateFatalError(err) {
		return err.Error()
	}

	return ""
}

func setRemediationLastUpdated(err error) sql.NullTime {
	ret := sql.NullTime{}
	if evalerrors.IsRemediateInformativeError(err) {
		// just return a NullString
		return ret
	}

	ret.Valid = true
	ret.Time = time.Now()

	return ret
}
