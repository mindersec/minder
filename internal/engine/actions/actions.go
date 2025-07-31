// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package actions provide necessary interfaces and implementations for
// processing actions, such as remediation and alerts.
package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/actions/alert"
	"github.com/mindersec/minder/internal/engine/actions/remediate"
	"github.com/mindersec/minder/internal/engine/actions/remediate/pull_request"
	enginerr "github.com/mindersec/minder/internal/engine/errors"
	engif "github.com/mindersec/minder/internal/engine/interfaces"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/profiles/models"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// RuleActionsEngine is the engine responsible for processing all actions i.e., remediation and alerts
type RuleActionsEngine struct {
	actions map[engif.ActionType]engif.Action
}

// NewRuleActions creates a new rule actions engine
func NewRuleActions(
	ctx context.Context,
	ruletype *minderv1.RuleType,
	provider provinfv1.Provider,
	actionConfig *models.ActionConfiguration,
) (*RuleActionsEngine, error) {
	// Create the remediation engine
	remEngine, err := remediate.NewRuleRemediator(ruletype, provider, actionConfig.Remediate)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule remediator: %w", err)
	}

	// Create the alert engine
	alertEngine, err := alert.NewRuleAlert(ctx, ruletype, provider, actionConfig.Alert)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule alerter: %w", err)
	}

	return &RuleActionsEngine{
		actions: map[engif.ActionType]engif.Action{
			remEngine.Class():   remEngine,
			alertEngine.Class(): alertEngine,
		},
	}, nil
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
			getRemediationMeta(params.GetEvalStatusFromDb()))
	}

	// Try alerting
	if !skipAlert {
		// Decide if we should alert
		cmd := shouldAlert(params.GetEvalStatusFromDb(), params.GetEvalErr(), result.RemediateErr, remediateEngine.Type())
		// Run alerting
		result.AlertMeta, result.AlertErr = rae.processAction(ctx, alert.ActionType, cmd, ent, params,
			getAlertMeta(params.GetEvalStatusFromDb()))
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
) (jmsg json.RawMessage, finalErr error) {
	defer func() {
		if r := recover(); r != nil {
			zerolog.Ctx(ctx).Error().Interface("recovered", r).
				Bytes("stack", debug.Stack()).
				Msg("panic in action execution")
			finalErr = enginerr.ErrInternal
		}
	}()
	zerolog.Ctx(ctx).Debug().Str("action", string(actionType)).Str("cmd", string(cmd)).Msg("invoking action")
	// Get action engine
	action := rae.actions[actionType]
	// Return the result of the action
	return action.Do(ctx, cmd, ent, params, metadata)
}

// shouldRemediate returns the action command for remediation taking into account previous evaluations
func shouldRemediate(prevEvalFromDb *db.ListRuleEvaluationsByProfileIdRow, evalErr error) engif.ActionCmd {
	// Get current evaluation status
	newEval := enginerr.ErrorAsEvalStatus(evalErr)

	// Get previous Remediation status
	prevRemediation := db.RemediationStatusTypesSkipped
	if prevEvalFromDb != nil {
		prevRemediation = prevEvalFromDb.RemStatus
	}

	// Start evaluation scenarios

	// Case 1 - Do not try to be smart about it by doing nothing if the evaluation status has not changed

	// Proceed with use cases where the evaluation changed
	switch newEval {
	case db.EvalStatusTypesError:
	case db.EvalStatusTypesSuccess:
		// Case 2 - Evaluation changed from something else to ERROR -> Remediation should be OFF
		// Case 3 - Evaluation changed from something else to PASSING -> Remediation should be OFF
		// The Remediation should be OFF (if it wasn't already)
		if db.RemediationStatusTypesSkipped != prevRemediation {
			return engif.ActionCmdOff
		}
		// We should do nothing if remediation was already skipped
		return engif.ActionCmdDoNothing
	case db.EvalStatusTypesFailure:
		// Case 4 - Evaluation has changed from something else to FAILED -> Remediation should be ON
		// We should remediate only if the previous remediation was skipped, so we don't risk endless remediation loops
		if db.RemediationStatusTypesSkipped == prevRemediation {
			return engif.ActionCmdOn
		}
		// Do nothing if the Remediation is something else other than skipped, i.e. pending, success, error, etc.
		return engif.ActionCmdDoNothing
	case db.EvalStatusTypesSkipped:
	case db.EvalStatusTypesPending:
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

	// Get previous Alert status
	prevAlert := db.AlertStatusTypesSkipped
	if prevEvalFromDb != nil {
		prevAlert = prevEvalFromDb.AlertStatus
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

	// Case 2 - Do not try to be smart about it by doing nothing if the evaluation status has not changed

	// Proceed with use cases where the evaluation changed
	switch newEval {
	case db.EvalStatusTypesError:
	case db.EvalStatusTypesFailure:
		// Case 3 - Evaluation changed from something else to ERROR -> Alert should be ON
		// Case 4 - Evaluation has changed from something else to FAILED -> Alert should be ON
		// The Alert should be on (if it wasn't already)
		if db.AlertStatusTypesOn != prevAlert {
			return engif.ActionCmdOn
		}
		// We should do nothing if alert was already turned on
		return engif.ActionCmdDoNothing
	case db.EvalStatusTypesSuccess:
		// Case 5 - Evaluation changed from something else to PASSING -> Alert should be OFF
		// The Alert should be turned OFF (if it wasn't already)
		if db.AlertStatusTypesOff != prevAlert {
			return engif.ActionCmdOff
		}
		// We should do nothing if the Alert is already OFF
		return engif.ActionCmdDoNothing
	case db.EvalStatusTypesSkipped:
	case db.EvalStatusTypesPending:
		return engif.ActionCmdDoNothing
	}

	// Default to do nothing
	return engif.ActionCmdDoNothing
}

// isSkippable returns true if the action should be skipped
func (rae *RuleActionsEngine) isSkippable(ctx context.Context, actionType engif.ActionType, evalErr error) bool {
	var skipAction bool

	logger := zerolog.Ctx(ctx).Info().
		Str("eval_status", string(enginerr.ErrorAsEvalStatus(evalErr))).
		Str("action", string(actionType))

	// Get the profile option set for this action type
	action, ok := rae.actions[actionType]
	if !ok {
		// If the action is not found, definitely skip it
		logger.Msg("action type not found, skipping")
		return true
	}
	// Check the action option
	switch action.GetOnOffState() {
	case models.ActionOptOff:
		// Action is off, skip
		logger.Msg("action is off, skipping")
		return true
	case models.ActionOptUnknown:
		// Action is unknown, skip
		logger.Msg("unknown action option, skipping")
		return true
	case models.ActionOptDryRun, models.ActionOptOn:
		// Action is on or dry-run, do not skip yet. Check the evaluation error
		skipAction =
			// rule evaluation was skipped, skip action too
			errors.Is(evalErr, interfaces.ErrEvaluationSkipped) ||
				// rule evaluation was skipped silently, skip action
				errors.Is(evalErr, enginerr.ErrEvaluationSkipSilently)
	}
	logger.Bool("skip_action", skipAction).Msg("action skip decision")
	// Everything else, do not skip
	return skipAction
}

// getRemediationMeta returns the json.RawMessage from the database type, empty if not valid
func getRemediationMeta(prevEvalFromDb *db.ListRuleEvaluationsByProfileIdRow) *json.RawMessage {
	if prevEvalFromDb != nil {
		return &prevEvalFromDb.RemMetadata
	}
	return nil
}

// getAlertMeta returns the json.RawMessage from the database type, empty if not valid
func getAlertMeta(prevEvalFromDb *db.ListRuleEvaluationsByProfileIdRow) *json.RawMessage {
	if prevEvalFromDb != nil {
		return &prevEvalFromDb.AlertMetadata
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
