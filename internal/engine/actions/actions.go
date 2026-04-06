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
// based on rule evaluation results.
type RuleActionsEngine struct {
	actions map[engif.ActionType]engif.Action
}

// NewRuleActions creates a new RuleActionsEngine with configured remediation
// and alert action handlers.
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

// DoActions executes remediation and alert actions based on evaluation results
// and returns the outcome of those actions.
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
			zerolog.Ctx(ctx).Error().
				Interface("recovered", r).
				Bytes("stack", debug.Stack()).
				Msg("panic in action execution")
			finalErr = enginerr.ErrInternal
		}
	}()

	zerolog.Ctx(ctx).Debug().
		Str("action", string(actionType)).
		Str("cmd", string(cmd)).
		Msg("invoking action")

	action := rae.actions[actionType]
	return action.Do(ctx, cmd, ent, params, metadata)
}

// ✅ FIXED (uses adapter + enum comparisons)
func shouldRemediate(prevEvalFromDb *db.ListRuleEvaluationsByProfileIdRow, evalErr error) engif.ActionCmd {

	newEval := dbadapter.ErrorAsEvalStatus(evalErr)

	prevRemediation := db.RemediationStatusTypesSkipped
	if prevEvalFromDb != nil {
		prevRemediation = prevEvalFromDb.RemStatus
	}

	if newEval == db.EvalStatusTypesSuccess {
		if prevRemediation != db.RemediationStatusTypesSkipped {
			return engif.ActionCmdOff
		}
		return engif.ActionCmdDoNothing
	}

	if newEval == db.EvalStatusTypesError {
		return engif.ActionCmdDoNothing
	}

	if newEval == db.EvalStatusTypesFailure {
		if prevRemediation == db.RemediationStatusTypesSkipped {
			return engif.ActionCmdOn
		}
		return engif.ActionCmdDoNothing
	}

	return engif.ActionCmdDoNothing
}

// ✅ FIXED
func shouldAlert(
	prevEvalFromDb *db.ListRuleEvaluationsByProfileIdRow,
	evalErr error,
	remErr error,
	remType string,
) engif.ActionCmd {

	newEval := dbadapter.ErrorAsEvalStatus(evalErr)

	prevAlert := db.AlertStatusTypesSkipped
	if prevEvalFromDb != nil {
		prevAlert = prevEvalFromDb.AlertStatus
	}

	if remType != pull_request.RemediateType && remErr == nil {
		if prevAlert != db.AlertStatusTypesOff {
			return engif.ActionCmdOff
		}
		return engif.ActionCmdDoNothing
	}

	if newEval == db.EvalStatusTypesError {
		return engif.ActionCmdDoNothing
	}

	if newEval == db.EvalStatusTypesFailure {
		if prevAlert != db.AlertStatusTypesOn {
			return engif.ActionCmdOn
		}
		return engif.ActionCmdDoNothing
	}

	if newEval == db.EvalStatusTypesSuccess {
		if prevAlert != db.AlertStatusTypesOff {
			return engif.ActionCmdOff
		}
		return engif.ActionCmdDoNothing
	}

	return engif.ActionCmdDoNothing
}

// ✅ FIXED
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
		logger.Msg("action is off, skipping")
		return true
	case models.ActionOptUnknown:
		logger.Msg("unknown action option, skipping")
		return true
	case models.ActionOptDryRun, models.ActionOptOn:
		return errors.Is(evalErr, interfaces.ErrEvaluationSkipped) ||
			errors.Is(evalErr, enginerr.ErrEvaluationSkipSilently)
	}

	return false
}

func getRemediationMeta(prevEvalFromDb *db.ListRuleEvaluationsByProfileIdRow) *json.RawMessage {
	if prevEvalFromDb != nil {
		return &prevEvalFromDb.RemMetadata
	}
	return nil
}

func getAlertMeta(prevEvalFromDb *db.ListRuleEvaluationsByProfileIdRow) *json.RawMessage {
	if prevEvalFromDb != nil {
		return &prevEvalFromDb.AlertMetadata
	}
	return nil
}

func getDefaultResult(ctx context.Context) enginerr.ActionsError {

	logger := zerolog.Ctx(ctx)

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
