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

	dbadapter "github.com/mindersec/minder/internal/adapters/db"
	"github.com/mindersec/minder/internal/db"
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

	result := getDefaultResult(ctx)
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

	if !skipRemediate {
		cmd := shouldRemediate(params.GetEvalStatusFromDb(), params.GetEvalErr())
		result.RemediateMeta, result.RemediateErr = rae.processAction(
			ctx,
			remediate.ActionType,
			cmd,
			ent,
			params,
			getRemediationMeta(params.GetEvalStatusFromDb()),
		)
	}

	if !skipAlert {
		cmd := shouldAlert(
			params.GetEvalStatusFromDb(),
			params.GetEvalErr(),
			result.RemediateErr,
			remediateEngine.Type(),
		)

		result.AlertMeta, result.AlertErr = rae.processAction(
			ctx,
			alert.ActionType,
			cmd,
			ent,
			params,
			getAlertMeta(params.GetEvalStatusFromDb()),
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
) (json.RawMessage, error) {

	defer func() {
		if r := recover(); r != nil {
			zerolog.Ctx(ctx).Error().
				Interface("recovered", r).
				Bytes("stack", debug.Stack()).
				Msg("panic in action execution")
		}
	}()

	action := rae.actions[actionType]
	return action.Do(ctx, cmd, ent, params, metadata)
}

// shouldRemediate returns the action command for remediation,
// taking into account the current evaluation result and the
// previous remediation state.
func shouldRemediate(prev *db.ListRuleEvaluationsByProfileIdRow, evalErr error) engif.ActionCmd {

	newEval := dbadapter.ErrorAsEvalStatus(evalErr)

	// Default to skipped if no previous evaluation exists
	prevRemediation := db.RemediationStatusTypesSkipped
	if prev != nil {
		prevRemediation = prev.RemStatus
	}

	switch newEval {
	case db.EvalStatusTypesError:
		return engif.ActionCmdDoNothing

	case db.EvalStatusTypesSuccess:
		if prevRemediation != db.RemediationStatusTypesSkipped {
			return engif.ActionCmdOff
		}
		return engif.ActionCmdDoNothing

	case db.EvalStatusTypesFailure:
		if prevRemediation == db.RemediationStatusTypesSkipped {
			return engif.ActionCmdOn
		}
		return engif.ActionCmdDoNothing

	case db.EvalStatusTypesSkipped, db.EvalStatusTypesPending:
		return engif.ActionCmdDoNothing
	}

	return engif.ActionCmdDoNothing
}

// shouldAlert returns the action command for alerting,
// based on the current evaluation result, previous alert state,
// and remediation outcome.
func shouldAlert(
	prev *db.ListRuleEvaluationsByProfileIdRow,
	evalErr error,
	remErr error,
	remType string,
) engif.ActionCmd {

	newEval := dbadapter.ErrorAsEvalStatus(evalErr)

	// Default to skipped if no previous evaluation exists
	prevAlert := db.AlertStatusTypesSkipped
	if prev != nil {
		prevAlert = prev.AlertStatus
	}

	if remType != pull_request.RemediateType && remErr == nil {
		if prevAlert != db.AlertStatusTypesOff {
			return engif.ActionCmdOff
		}
		return engif.ActionCmdDoNothing
	}

	switch newEval {
	case db.EvalStatusTypesError:
		return engif.ActionCmdDoNothing

	case db.EvalStatusTypesFailure:
		if prevAlert != db.AlertStatusTypesOn {
			return engif.ActionCmdOn
		}
		return engif.ActionCmdDoNothing

	case db.EvalStatusTypesSuccess:
		if prevAlert != db.AlertStatusTypesOff {
			return engif.ActionCmdOff
		}
		return engif.ActionCmdDoNothing

	case db.EvalStatusTypesSkipped, db.EvalStatusTypesPending:
		return engif.ActionCmdDoNothing
	}

	return engif.ActionCmdDoNothing
}

// isSkippable checks if action should be skipped
func (rae *RuleActionsEngine) isSkippable(ctx context.Context, actionType engif.ActionType, evalErr error) bool {

	logger := zerolog.Ctx(ctx).Info().
		Str("eval_status", string(dbadapter.ErrorAsEvalStatus(evalErr))).
		Str("action", string(actionType))

	action, ok := rae.actions[actionType]
	if !ok {
		logger.Msg("action type not found, skipping")
		return true
	}

	switch action.GetOnOffState() {
	case models.ActionOptOff:
		return true
	case models.ActionOptUnknown:
		return true
	case models.ActionOptDryRun, models.ActionOptOn:
		return errors.Is(evalErr, interfaces.ErrEvaluationSkipped) ||
			errors.Is(evalErr, enginerr.ErrEvaluationSkipSilently)
	}

	return false
}

// helpers
func getRemediationMeta(prev *db.ListRuleEvaluationsByProfileIdRow) *json.RawMessage {
	if prev != nil {
		return &prev.RemMetadata
	}
	return nil
}

func getAlertMeta(prev *db.ListRuleEvaluationsByProfileIdRow) *json.RawMessage {
	if prev != nil {
		return &prev.AlertMetadata
	}
	return nil
}

func getDefaultResult(ctx context.Context) enginerr.ActionsError {
	logger := zerolog.Ctx(ctx)

	m, err := json.Marshal(&map[string]any{})
	if err != nil {
		logger.Error().Err(err).Msg("error marshaling empty json")
	}

	return enginerr.ActionsError{
		RemediateErr:  enginerr.ErrActionSkipped,
		AlertErr:      enginerr.ErrActionSkipped,
		RemediateMeta: m,
		AlertMeta:     m,
	}
}
