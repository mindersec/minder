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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/entities"
	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/profiles/models"
)

func (e *executor) createEvalStatusParams(
	ctx context.Context,
	inf *entities.EntityInfoWrapper,
	profile *models.ProfileAggregate,
	rule *models.RuleInstance,
) (*engif.EvalStatusParams, error) {
	eID, err := inf.GetID()
	if err != nil {
		return nil, fmt.Errorf("Error getting ID from entity info wrapper")
	}

	params := &engif.EvalStatusParams{
		Rule:        rule,
		Profile:     profile,
		EntityType:  entities.EntityTypeToDB(inf.Type),
		EntityID:    eID,
		ProjectID:   inf.ProjectID,
		ExecutionID: *inf.ExecutionID, // Execution ID is required in the executor.
	}

	// TODO: once we replace the existing profile state types with the new
	// evaluation history tables, this can go away.
	ruleTypeName, err := e.querier.GetRuleTypeNameByID(ctx, rule.RuleTypeID)
	if err != nil {
		return nil, fmt.Errorf("error while retrieving rule type name: %w", err)
	}

	nullableRuleTypeName := sql.NullString{
		String: ruleTypeName,
		Valid:  true,
	}

	ruleName := sql.NullString{
		String: rule.Name,
		Valid:  true,
	}

	// Get the current rule evaluation from the database
	evalStatus, err := e.querier.GetRuleEvaluationByProfileIdAndRuleType(ctx,
		params.Profile.ID,
		ruleName,
		eID,
		nullableRuleTypeName,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting rule evaluation status from db: %w", err)
	}

	// Save the current rule evaluation status to the evalParams
	params.EvalStatusFromDb = evalStatus

	return params, nil
}

// createOrUpdateEvalStatus takes care of recording the rule evaluation results.
// This function inserts into the database:
//
//   - The rule evaluation parameters (profile, repo, artifact, entity, etc).
//   - The rule evaluation status and details.
//   - The remediation status and details.
//   - The alert status and details.
//
// If the error in the evaluation status resolves to an errors.ErrEvaluationSkipSilently,
// no details are stored or logged.
func (e *executor) createOrUpdateEvalStatus(
	ctx context.Context,
	params *engif.EvalStatusParams,
) error {
	logger := params.DecorateLogger(zerolog.Ctx(ctx).With().Logger())
	// Make sure evalParams is not nil
	if params == nil {
		return fmt.Errorf("createEvalStatusParams cannot be nil")
	}

	// Check if we should skip silently
	if errors.Is(params.GetEvalErr(), evalerrors.ErrEvaluationSkipSilently) {
		logger.Info().Msg("rule evaluation skipped silently - skip updating the database")
		return nil
	}

	status := evalerrors.ErrorAsEvalStatus(params.GetEvalErr())
	e.metrics.CountEvalStatus(ctx, status, params.EntityType)

	remediationStatus := evalerrors.ErrorAsRemediationStatus(params.GetActionsErr().RemediateErr)
	e.metrics.CountRemediationStatus(ctx, remediationStatus)

	alertStatus := evalerrors.ErrorAsAlertStatus(params.GetActionsErr().AlertErr)
	e.metrics.CountAlertStatus(ctx, alertStatus)

	chckpoint := params.GetIngestResult().GetCheckpoint()
	chkpjs, err := chckpoint.ToJSONorDefault(json.RawMessage(`{}`))
	if err != nil {
		logger.Err(err).Msg("error marshalling checkpoint")
	}

	// Log result in the evaluation history tables
	err = e.querier.WithTransactionErr(func(qtx db.ExtendQuerier) error {
		evalID, err := e.historyService.StoreEvaluationStatus(
			ctx,
			qtx,
			params.Rule.ID,
			params.Profile.ID,
			params.EntityType,
			params.EntityID,
			params.GetEvalErr(),
			chkpjs,
		)
		if err != nil {
			return err
		}

		// These could be added into the history service, but since there
		// is ongoing discussion about decoupling alerting and remediation
		// from evaluation, I am leaving them here to make them easy to
		// move elsewhere.
		err = qtx.InsertRemediationEvent(ctx, db.InsertRemediationEventParams{
			EvaluationID: evalID,
			Status:       remediationStatus,
			Details:      errorAsActionDetails(params.GetActionsErr().RemediateErr),
			Metadata:     params.GetActionsErr().RemediateMeta,
		})
		if err != nil {
			return err
		}

		return qtx.InsertAlertEvent(ctx, db.InsertAlertEventParams{
			EvaluationID: evalID,
			Status:       alertStatus,
			Details:      errorAsActionDetails(params.GetActionsErr().AlertErr),
			Metadata:     params.GetActionsErr().AlertMeta,
		})
	})
	if err != nil {
		logger.Err(err).Msg("error logging evaluation status")
		return err
	}

	return err
}

func errorAsActionDetails(err error) string {
	if evalerrors.IsActionFatalError(err) {
		return err.Error()
	}

	return ""
}
