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
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/engine/actions/alert"
	"github.com/stacklok/mediator/internal/engine/actions/remediate"
	enginerr "github.com/stacklok/mediator/internal/engine/errors"
	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	"github.com/stacklok/mediator/internal/providers"
	mediatorv1 "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// RuleActionsEngine is the engine responsible for processing all actions i.e., remediation and alerts
type RuleActionsEngine struct {
	actions      map[engif.ActionType]engif.Action
	actionsOnOff map[engif.ActionType]engif.ActionOpt
}

// NewRuleActions creates a new rule actions engine
func NewRuleActions(p *mediatorv1.Profile, rt *mediatorv1.RuleType, pbuild *providers.ProviderBuilder,
) (*RuleActionsEngine, error) {
	// Create the remediation engine
	remEngine, err := remediate.NewRuleRemediator(rt, pbuild)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule remediator: %w", err)
	}

	// Create the alert engine
	alertEngine, err := alert.NewRuleAlert(rt, pbuild)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule remediator: %w", err)
	}

	return &RuleActionsEngine{
		actions: map[engif.ActionType]engif.Action{
			remEngine.Type():   remEngine,
			alertEngine.Type(): alertEngine,
		},
		actionsOnOff: map[engif.ActionType]engif.ActionOpt{
			remEngine.Type():   remEngine.GetOnOffState(p),
			alertEngine.Type(): alertEngine.GetOnOffState(p),
		},
	}, nil
}

// DoActions processes all actions i.e., remediation and alerts
func (rae *RuleActionsEngine) DoActions(
	ctx context.Context,
	ent protoreflect.ProtoMessage,
	ruleDef map[string]any,
	ruleParams map[string]any,
	evalErr error,
	dbEvalStatus db.ListRuleEvaluationsByProfileIdRow,
) enginerr.ActionsError {
	// Get logger
	logger := zerolog.Ctx(ctx)

	// Default to return that we skipped all actions
	err := enginerr.ActionsError{
		RemediateErr: enginerr.ErrActionSkipped,
		AlertErr:     enginerr.ErrActionSkipped,
	}
	skipRemediate := true
	skipAlert := true

	// Load remediate action engine
	remediateEngine, ok := rae.actions[remediate.ActionType]
	if !ok {
		logger.Debug().Msg(fmt.Sprintf("action engine not found: %s", remediate.ActionType))
		err.RemediateErr = fmt.Errorf("%s:%w", remediate.ActionType, enginerr.ErrActionNotAvailable)
	} else {
		skipRemediate = rae.isSkippable(ctx, remediate.ActionType, evalErr)
	}

	// Load alert action engine
	alertEngine, ok := rae.actions[alert.ActionType]
	if !ok {
		logger.Debug().Msg(fmt.Sprintf("action engine not found: %s", alert.ActionType))
		err.AlertErr = fmt.Errorf("%s:%w", alert.ActionType, enginerr.ErrActionNotAvailable)
	} else {
		skipAlert = rae.isSkippable(ctx, alert.ActionType, evalErr)
	}

	// Exit early if both should be skipped
	if skipRemediate && skipAlert {
		// err is still set to skip all actions
		return err
	}

	// Try remediating
	if !skipRemediate {
		res := shouldRemediate(dbEvalStatus, evalErr)
		err.RemediateErr = remediateEngine.Do(ctx, res, rae.actionsOnOff[remediate.ActionType], ent, ruleDef, ruleParams)
	}

	// Try alerting
	if !skipAlert {
		res := shouldAlert(dbEvalStatus, evalErr, err.RemediateErr)
		err.AlertErr = alertEngine.Do(ctx, res, rae.actionsOnOff[alert.ActionType], ent, ruleDef, ruleParams)
	}

	return err
}

// shouldRemediate returns the action command for remediation taking into account previous evaluations
func shouldRemediate(prevEvalFromDb db.ListRuleEvaluationsByProfileIdRow, evalErr error) engif.ActionCmd {
	_ = prevEvalFromDb
	_ = evalErr
	return engif.ActionCmdOn
}

// shouldAlert returns the action command for alerting taking into account previous evaluations
func shouldAlert(prevEvalFromDb db.ListRuleEvaluationsByProfileIdRow, evalErr error, remErr error) engif.ActionCmd {
	// Start simple without taking into account the remediation status
	_ = remErr
	newEval := enginerr.ErrorAsEvalStatus(evalErr)
	prevAlert := db.AlertStatusTypesSkipped
	if prevEvalFromDb.AlertStatus.Valid {
		prevAlert = prevEvalFromDb.AlertStatus.AlertStatusTypes
	}

	// Start evaluation scenarios
	// Case 1 - Evaluation has PASSED, but the Alert was ON.
	if db.EvalStatusTypesSuccess == newEval && db.AlertStatusTypesOn == prevAlert {
		// We should turn it OFF.
		return engif.ActionCmdOff
	}

	// Case 2 - Evaluation has FAILED, but the alert was OFF.
	if db.EvalStatusTypesFailure == newEval && db.AlertStatusTypesOff == prevAlert {
		// We should turn it ON.
		return engif.ActionCmdOn
	}

	// Do nothing in all other cases
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
		logger.Debug().Msg("unknown action option, check your profile definition")
		return true
	case engif.ActionOptDryRun, engif.ActionOptOn:
		// Action is on or dry-run, do not skip yet. Check the evaluation error
		skipRemediation =
			// rule evaluation was skipped, skip action too
			errors.Is(evalErr, enginerr.ErrEvaluationSkipped) ||
				// rule evaluation was skipped silently, skip action
				errors.Is(evalErr, enginerr.ErrEvaluationSkipSilently) ||
				// TODO: (radoslav) Discuss this with Jakub and decide if we want to skip remediation too
				// rule evaluation had no error, skip action if actionType IS NOT alert
				(evalErr == nil && actionType != alert.ActionType)
	}
	// Everything else, do not skip
	return skipRemediation

}
