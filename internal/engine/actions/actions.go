// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package actions provides the rule actions engine responsible for
// processing remediation and alert actions based on evaluation results.
package actions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/engine/actions/alert"
	"github.com/mindersec/minder/internal/engine/actions/remediate"
	"github.com/mindersec/minder/internal/engine/actions/remediate/pull_request"
	engif "github.com/mindersec/minder/internal/engine/interfaces"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	enginerr "github.com/mindersec/minder/pkg/engine/errors"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/profiles/models"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// RuleActionsEngine is responsible for executing remediation and alert actions
type RuleActionsEngine struct {
	actions map[engif.ActionType]engif.Action
}

// NewRuleActions creates a new RuleActionsEngine
func NewRuleActions(
	ctx context.Context,
	ruletype *minderv1.RuleType,
	provider provinfv1.Provider,
	actionConfig *models.ActionConfiguration,
) (*RuleActionsEngine, error) {

	remEngine, err := remediate.NewRuleRemediator(ruletype, provider, actionConfig.Remediate)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule remediator: %w", err)
	}

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

// DoActions executes remediation and alert actions
func (rae *RuleActionsEngine) DoActions(
	ctx context.Context,
	ent protoreflect.ProtoMessage,
	params engif.ActionsParams,
) enginerr.ActionsError {
	logger := zerolog.Ctx(ctx)

	result := getDefaultResult()
	skipRemediate := true
	skipAlert := true

	remediateEngine, ok := rae.actions[remediate.ActionType]
	if !ok {
		logger.Error().Str("action_type", string(remediate.ActionType)).Msg("not found")
		result.RemediateErr = fmt.Errorf("%s:%w", remediate.ActionType, enginerr.ErrActionNotAvailable)
	} else {
		skipRemediate = rae.isSkippable(ctx, remediate.ActionType, params.GetEvalErr())
	}

	_, ok = rae.actions[alert.ActionType]
	if !ok {
		logger.Error().Str("action_type", string(alert.ActionType)).Msg("not found")
		result.AlertErr = fmt.Errorf("%s:%w", alert.ActionType, enginerr.ErrActionNotAvailable)
	} else {
		skipAlert = rae.isSkippable(ctx, alert.ActionType, params.GetEvalErr())
	}

	var prev *PreviousEval
	if row := params.GetEvalStatusFromDb(); row != nil {
		prev = &PreviousEval{
			RemediationStatus: RemediationStatus(row.RemStatus),
			AlertStatus:       AlertStatus(row.AlertStatus),
			RemediationMeta:   &row.RemMetadata,
			AlertMeta:         &row.AlertMetadata,
		}
	}
	status := mapEvalStatus(params.GetEvalErr())

	var remMeta *json.RawMessage
	var alertMeta *json.RawMessage

	if prev != nil {
		remMeta = getRemediationMeta(prev)
		alertMeta = getAlertMeta(prev)
	}

	if !skipRemediate {
		cmd := shouldRemediate(prev, status)
		result.RemediateMeta, result.RemediateErr = rae.processAction(
			ctx,
			remediate.ActionType,
			cmd,
			ent,
			params,
			remMeta,
		)
	}

	if !skipAlert {
		cmd := shouldAlert(
			prev,
			status,
			result.RemediateErr,
			remediateEngine.Type(),
		)

		result.AlertMeta, result.AlertErr = rae.processAction(
			ctx,
			alert.ActionType,
			cmd,
			ent,
			params,
			alertMeta,
		)
	}

	return result
}

// processAction safely executes an action
func (rae *RuleActionsEngine) processAction(
	ctx context.Context,
	actionType engif.ActionType,
	cmd engif.ActionCmd,
	ent protoreflect.ProtoMessage,
	params engif.ActionsParams,
	metadata *json.RawMessage,
) (res json.RawMessage, err error) {
	defer func() {
		if r := recover(); r != nil {
			zerolog.Ctx(ctx).Error().
				Interface("recovered", r).
				Bytes("stack", debug.Stack()).
				Msg("panic in action execution")

			err = enginerr.ErrInternal
		}
	}()

	action := rae.actions[actionType]
	return action.Do(ctx, cmd, ent, params, metadata)
}

// shouldRemediate determines the remediation action command.
// taking into account the current evaluation result and the
// previous remediation state.
func shouldRemediate(prevEval *PreviousEval, evalStatus EvalStatus) engif.ActionCmd {
	// Default previous remediation status
	prevRemediation := RemediationStatusSkipped
	if prevEval != nil {
		prevRemediation = prevEval.RemediationStatus
	}

	switch evalStatus {
	case EvalStatusError:
		// TODO: Historically, error may have fallen through to success behavior.
		// This change treats error as no-op. Confirm intended behavior.
		return engif.ActionCmdDoNothing

	case EvalStatusSuccess:
		// Case 2 - Evaluation changed from something else to ERROR -> Remediation should be OFF
		// Case 3 - Evaluation changed from something else to PASSING -> Remediation should be OFF
		// If remediation was previously active, turn it off.
		if prevRemediation != RemediationStatusSkipped {
			return engif.ActionCmdOff
		}
		// DoNothing means we keep the current state unchanged, rather than explicitly turning it OFF.
		return engif.ActionCmdDoNothing

	case EvalStatusFailure:
		// Case 4 - Evaluation has changed from something else to FAILED -> Remediation should be ON
		// We only trigger remediation if the previous remediation was skipped.
		// This prevents repeated remediation attempts and avoids endless remediation loops
		// where the system keeps trying to remediate an already-handled failure.
		if prevRemediation == RemediationStatusSkipped {
			return engif.ActionCmdOn
		}
		return engif.ActionCmdDoNothing

	case EvalStatusSkipped, EvalStatusPending:
		return engif.ActionCmdDoNothing
	}

	return engif.ActionCmdDoNothing
}

// shouldAlert returns the action command for alerting,
// based on the current evaluation result, previous alert state,
// and remediation outcome.
func shouldAlert(
	prevEval *PreviousEval,
	evalStatus EvalStatus,
	remErr error,
	remType string,
) engif.ActionCmd {

	// Default previous alert status
	prevAlert := AlertStatusSkipped
	if prevEval != nil {
		prevAlert = prevEval.AlertStatus
	}

	// Case 1 - Successful remediation of a type that is not PR is considered instant.
	if remType != pull_request.RemediateType && remErr == nil {
		// If remediation succeeded, we turn alert OFF.
		// This effectively skips alerting (or turns it off) when remediation resolves the issue immediately.
		if prevAlert != AlertStatusOff {
			return engif.ActionCmdOff
		}
		return engif.ActionCmdDoNothing
	}

	// Case 2 - Do not try to be smart about it by doing nothing if the evaluation status has not changed

	// Proceed with use cases where the evaluation changed

	// The alert logic attempts to interpret transitions between previous and current states,
	// rather than simply mirroring the current evaluation result. This helps avoid noisy or
	// redundant alerts and ensures we react only to meaningful state changes.

	switch evalStatus {

	case EvalStatusError:
		// TODO: Historically, error may have fallen through to success behavior.
		// This change treats error as no-op. Confirm intended behavior.
		return engif.ActionCmdDoNothing

	// Case 3 - Evaluation changed from something else to ERROR -> Alert should be ON
	// Case 4 - Evaluation has changed from something else to FAILED -> Alert should be ON
	case EvalStatusFailure:
		if prevAlert != AlertStatusOn {
			return engif.ActionCmdOn
		}
		return engif.ActionCmdDoNothing
	// Case 5 - Evaluation changed from something else to PASSING -> Alert should be OFF
	case EvalStatusSuccess:
		// Case 5 - Evaluation changed from something else to PASSING -> Alert should be OFF
		if prevAlert != AlertStatusOff {
			return engif.ActionCmdOff
		}
		return engif.ActionCmdDoNothing

	case EvalStatusSkipped, EvalStatusPending:
		return engif.ActionCmdDoNothing
	}

	return engif.ActionCmdDoNothing
}

// isSkippable checks if action should be skipped
func (rae *RuleActionsEngine) isSkippable(ctx context.Context, actionType engif.ActionType, evalErr error) bool {
	action, ok := rae.actions[actionType]
	if !ok {
		zerolog.Ctx(ctx).Info().
			Str("eval_status", string(mapEvalStatus(evalErr))).
			Str("action", string(actionType)).
			Msg("action type not found, skipping")
		return true
	}

	switch action.GetOnOffState() {
	case models.ActionOptOff:
		return true
	case models.ActionOptUnknown:
		return true
	case models.ActionOptDryRun, models.ActionOptOn:
		// Some actions in ON or DRY-RUN mode may still be skipped if the evaluation
		// explicitly returns a "skip" result. This ensures we respect evaluation intent.
		return errors.Is(evalErr, interfaces.ErrEvaluationSkipped) ||
			errors.Is(evalErr, enginerr.ErrEvaluationSkipSilently)
	}

	return false
}

// getRemediationMeta returns remediation metadata from previous evaluation.
func getRemediationMeta(prevEval *PreviousEval) *json.RawMessage {
	if prevEval != nil {
		return prevEval.RemediationMeta
	}
	return nil
}

// getAlertMeta returns alert metadata from previous evaluation.
func getAlertMeta(prevEval *PreviousEval) *json.RawMessage {
	if prevEval != nil {
		return prevEval.AlertMeta
	}
	return nil
}

func getDefaultResult() enginerr.ActionsError {
	m := json.RawMessage("{}")

	return enginerr.ActionsError{
		RemediateErr:  enginerr.ErrActionSkipped,
		AlertErr:      enginerr.ErrActionSkipped,
		RemediateMeta: m,
		AlertMeta:     m,
	}
}

// mapEvalStatus converts evaluation error into engine EvalStatus.
func mapEvalStatus(err error) EvalStatus {
	if err == nil {
		return EvalStatusSuccess
	}

	// skipped cases
	if errors.Is(err, enginerr.ErrEvaluationSkipSilently) ||
		errors.Is(err, interfaces.ErrEvaluationSkipped) {
		return EvalStatusSkipped
	}

	// failure case (CORRECT detection)
	if errors.Is(err, interfaces.ErrEvaluationFailed) {
		return EvalStatusFailure
	}

	// everything else = error
	return EvalStatusError
}
