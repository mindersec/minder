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
// Package rule provides the CLI subcommand for managing rules

// Package actions provide necessary interfaces and implementations for
// processing actions, such as remediation and alerts.
package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/sqlc-dev/pqtype"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/actions/alert"
	"github.com/stacklok/minder/internal/engine/actions/remediate"
	"github.com/stacklok/minder/internal/engine/actions/remediate/pull_request"
	enginerr "github.com/stacklok/minder/internal/engine/errors"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// RuleActionsEngine is the engine responsible for processing all actions i.e., remediation and alerts
type RuleActionsEngine struct {
	actions      map[engif.ActionType]engif.Action
	actionsOnOff map[engif.ActionType]engif.ActionOpt
}

// NewRuleActions creates a new rule actions engine
func NewRuleActions(p *minderv1.Profile, rt *minderv1.RuleType, pbuild *providers.ProviderBuilder,
) (*RuleActionsEngine, error) {
	// Create the remediation engine
	remEngine, err := remediate.NewRuleRemediator(rt, pbuild)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule remediator: %w", err)
	}

	// Create the alert engine
	alertEngine, err := alert.NewRuleAlert(rt, pbuild)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule alerter: %w", err)
	}

	return &RuleActionsEngine{
		actions: map[engif.ActionType]engif.Action{
			remEngine.Class():   remEngine,
			alertEngine.Class(): alertEngine,
		},
		// The on/off state of the actions is an integral part of the action engine
		// and should be set upon creation.
		actionsOnOff: map[engif.ActionType]engif.ActionOpt{
			remEngine.Class():   remEngine.GetOnOffState(p),
			alertEngine.Class(): alertEngine.GetOnOffState(p),
		},
	}, nil
}

// GetOnOffState returns the on/off state of the actions
func (rae *RuleActionsEngine) GetOnOffState() map[engif.ActionType]engif.ActionOpt {
	return rae.actionsOnOff
}

// DoActions processes all actions i.e., remediation and alerts
func (rae *RuleActionsEngine) DoActions(
	ctx context.Context,
	ent protoreflect.ProtoMessage,
	params engif.ActionsParams,
) enginerr.ActionsError {
	// Get logger
	logger := zerolog.Ctx(ctx)

	// Default to skipping all actions
	result := getDefaultResult(ctx)
	skipRemediate := true
	skipAlert := true

	// Verify the remediate action engine is available and get its status - on/off/dry-run
	remediateEngine, ok := rae.actions[remediate.ActionType]
	if !ok {
		logger.Error().Str("action_type", string(remediate.ActionType)).Msg("not found")
		result.RemediateErr = fmt.Errorf("%s:%w", remediate.ActionType, enginerr.ErrActionNotAvailable)
	} else {
		skipRemediate = rae.isSkippable(ctx, remediate.ActionType, params.GetEvalErr())
	}

	// Verify the alert action engine is available and get its status - on/off/dry-run
	_, ok = rae.actions[alert.ActionType]
	if !ok {
		logger.Error().Str("action_type", string(alert.ActionType)).Msg("not found")
		result.AlertErr = fmt.Errorf("%s:%w", alert.ActionType, enginerr.ErrActionNotAvailable)
	} else {
		skipAlert = rae.isSkippable(ctx, alert.ActionType, params.GetEvalErr())
	}

	// Try remediating
	if !skipRemediate {
		// Decide if we should remediate
		cmd := shouldRemediate(params.GetEvalStatusFromDb(), params.GetEvalErr())
		// Run remediation
		result.RemediateMeta, result.RemediateErr = rae.processAction(ctx, remediate.ActionType, cmd, ent, params,
			getMeta(params.GetEvalStatusFromDb().RemMetadata))
	}

	// Try alerting
	if !skipAlert {
		// Decide if we should alert
		cmd := shouldAlert(params.GetEvalStatusFromDb(), params.GetEvalErr(), result.RemediateErr, remediateEngine.Type())
		// Run alerting
		result.AlertMeta, result.AlertErr = rae.processAction(ctx, alert.ActionType, cmd, ent, params,
			getMeta(params.GetEvalStatusFromDb().AlertMetadata))
	}
	return result
}

// processAction runs the action engine for the given action type, and also sanity checks the result of the action
func (rae *RuleActionsEngine) processAction(
	ctx context.Context,
	actionType engif.ActionType,
	cmd engif.ActionCmd,
	ent protoreflect.ProtoMessage,
	params engif.ActionsParams,
	metadata *json.RawMessage,
) (json.RawMessage, error) {
	// Get action engine
	action := rae.actions[actionType]
	// Return the result of the action
	logger := zerolog.Ctx(ctx)
	logger.Debug().
		Str("action", string(actionType)).
		Str("cmd", string(cmd)).
		Msg("invoking action")
	return action.Do(ctx, cmd, rae.actionsOnOff[actionType], ent, params, metadata)
}

// shouldRemediate returns the action command for remediation taking into account previous evaluations
func shouldRemediate(prevEvalFromDb *db.ListRuleEvaluationsByProfileIdRow, evalErr error) engif.ActionCmd {
	// Get current evaluation status
	newEval := enginerr.ErrorAsEvalStatus(evalErr)

	// Get previous evaluation status
	prevEval := db.EvalStatusTypesPending
	if prevEvalFromDb.EvalStatus.Valid {
		prevEval = prevEvalFromDb.EvalStatus.EvalStatusTypes
	}

	// Get previous Remediation status
	prevRemediation := db.RemediationStatusTypesSkipped
	if prevEvalFromDb.AlertStatus.Valid {
		prevRemediation = prevEvalFromDb.RemStatus.RemediationStatusTypes
	}

	// Start evaluation scenarios

	// Case 1 - Do nothing if the evaluation status has not changed
	if newEval == prevEval && prevRemediation != db.RemediationStatusTypesError {
		return engif.ActionCmdDoNothing
	}

	// Proceed with use cases where the evaluation changed
	// Case 2 - Evaluation changed from something else to PASSING -> Remediation should be OFF
	if db.EvalStatusTypesSuccess == newEval {
		// The Remediation should be OFF (if it wasn't already)
		if db.RemediationStatusTypesSkipped != prevRemediation {
			return engif.ActionCmdOff
		}
		// We should do nothing if remediation was already skipped
		return engif.ActionCmdDoNothing
	}

	// Case 3 - Evaluation has changed from something else to FAILED -> Remediation should be ON
	if db.EvalStatusTypesFailure == newEval {
		// The Remediation should be turned ON (if it wasn't already)
		if db.RemediationStatusTypesPending != prevRemediation && db.RemediationStatusTypesSuccess != prevRemediation {
			return engif.ActionCmdOn
		}
		// We should do nothing if the Remediation is already pending or successful
		return engif.ActionCmdDoNothing
	}
	// Default to do nothing
	return engif.ActionCmdDoNothing
}

// shouldAlert returns the action command for alerting taking into account previous evaluations
func shouldAlert(
	prevEvalFromDb *db.ListRuleEvaluationsByProfileIdRow,
	evalErr error,
	remErr error,
	remType string,
) engif.ActionCmd {
	// Get current evaluation status
	newEval := enginerr.ErrorAsEvalStatus(evalErr)

	// Get previous evaluation status
	prevEval := db.EvalStatusTypesPending
	if prevEvalFromDb.EvalStatus.Valid {
		prevEval = prevEvalFromDb.EvalStatus.EvalStatusTypes
	}

	// Get previous Alert status
	prevAlert := db.AlertStatusTypesSkipped
	if prevEvalFromDb.AlertStatus.Valid {
		prevAlert = prevEvalFromDb.AlertStatus.AlertStatusTypes
	}

	// Start evaluation scenarios

	// Case 1 - Successful remediation of a type that is not PR is considered instant.
	if remType != pull_request.RemediateType && remErr == nil {
		// If this is the case either skip alerting or turn it off if it was on
		if prevAlert != db.AlertStatusTypesOff {
			return engif.ActionCmdOff
		}
		return engif.ActionCmdDoNothing
	}

	// Case 2 - Do nothing if the evaluation status has not changed
	if newEval == prevEval && prevAlert != db.AlertStatusTypesError {
		return engif.ActionCmdDoNothing
	}

	// Proceed with use cases where the evaluation changed

	// Case 3 - Evaluation changed from something else to PASSING -> Alarm should be OFF
	if db.EvalStatusTypesSuccess == newEval {
		// The Alert should be turned OFF (if it wasn't already)
		if db.AlertStatusTypesOff != prevAlert {
			return engif.ActionCmdOff
		}
		// We should do nothing if the Alert is already OFF
		return engif.ActionCmdDoNothing
	}

	// Case 4 - Evaluation has changed from something else to FAILED -> Alarm should be ON
	if db.EvalStatusTypesFailure == newEval {
		// The Alert should be turned ON (if it wasn't already)
		if db.AlertStatusTypesOn != prevAlert {
			return engif.ActionCmdOn
		}
		// We should do nothing if the Alert is already ON
		return engif.ActionCmdDoNothing
	}
	// Default to do nothing
	return engif.ActionCmdDoNothing
}

// isSkippable returns true if the action should be skipped
func (rae *RuleActionsEngine) isSkippable(ctx context.Context, actionType engif.ActionType, evalErr error) bool {
	var skipRemediation bool

	logger := zerolog.Ctx(ctx)

	// Get the profile option set for this action type
	actionOnOff, ok := rae.actionsOnOff[actionType]
	if !ok {
		// If the action is not found, definitely skip it
		return true
	}
	// Check the action option
	switch actionOnOff {
	case engif.ActionOptOff:
		// Action is off, skip
		return true
	case engif.ActionOptUnknown:
		// Action is unknown, skip
		logger.Info().Msg("unknown action option, check your profile definition")
		return true
	case engif.ActionOptDryRun, engif.ActionOptOn:
		// Action is on or dry-run, do not skip yet. Check the evaluation error
		skipRemediation =
			// rule evaluation was skipped, skip action too
			errors.Is(evalErr, enginerr.ErrEvaluationSkipped) ||
				// rule evaluation was skipped silently, skip action
				errors.Is(evalErr, enginerr.ErrEvaluationSkipSilently) ||
				// skip if the error is not a failure and the action type is remediate
				(!errors.Is(evalErr, enginerr.ErrEvaluationFailed) && actionType == remediate.ActionType) ||
				// rule evaluation had no error, skip action if actionType IS NOT alert
				(evalErr == nil && actionType != alert.ActionType)
	}
	// Everything else, do not skip
	return skipRemediation
}

// getMeta returns the json.RawMessage from the database type, empty if not valid
func getMeta(rawMsg pqtype.NullRawMessage) *json.RawMessage {
	if rawMsg.Valid {
		return &rawMsg.RawMessage
	}
	return nil
}

// getDefaultResult returns the default result for the action engine
func getDefaultResult(ctx context.Context) enginerr.ActionsError {
	// Get logger
	logger := zerolog.Ctx(ctx)

	// Even though meta is an empty json struct by default, there's no risk of overwriting
	// any existing meta entry since we don't upsert in case of conflict while skipping
	m, err := json.Marshal(&map[string]any{})
	if err != nil {
		logger.Error().Err(err).Msg("error marshaling empty json.RawMessage")
	}
	return enginerr.ActionsError{
		RemediateErr:  enginerr.ErrActionSkipped,
		AlertErr:      enginerr.ErrActionSkipped,
		RemediateMeta: m,
		AlertMeta:     m,
	}
}
