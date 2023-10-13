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
	"fmt"
	"log"

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
	actions map[string]engif.Action
}

// NewRuleActions creates a new rule actions engine
func NewRuleActions(rt *mediatorv1.RuleType, pbuild *providers.ProviderBuilder) (*RuleActionsEngine, error) {
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
		actions: map[string]engif.Action{
			remEngine.Type():   remEngine,
			alertEngine.Type(): alertEngine,
		},
	}, nil
}

// DoActions processes all actions i.e., remediation and alerts
func (rae *RuleActionsEngine) DoActions(
	ctx context.Context,
	ent protoreflect.ProtoMessage,
	ruleDef map[string]any,
	ruleParams map[string]any,
	actionsOnOff map[string]engif.ActionOpt,
	evalErr error,
	dbEvalStatus db.ListRuleEvaluationsByProfileIdRow,
) enginerr.ActionsError {
	// TODO: revisit skipping actions
	err := enginerr.ActionsError{
		RemediateErr: enginerr.ErrActionSkipped,
		AlertErr:     enginerr.ErrActionSkipped,
	}
	skipRemediate := true
	skipAlert := true

	// Load remediate action engine
	remediateEngine, remOK := rae.actions[remediate.ActionType]
	if !remOK {
		log.Printf("action engine not found: %s", remediate.ActionType)
		err.RemediateErr = fmt.Errorf("%s:%w", remediate.ActionType, enginerr.ErrActionNotAvailable)
	} else {
		skipRemediate = remediateEngine.IsSkippable(ctx, actionsOnOff[remediate.ActionType], evalErr)
	}

	// Load alert action engine
	alertEngine, alertOK := rae.actions[alert.ActionType]
	if !alertOK {
		log.Printf("action engine not found: %s", alert.ActionType)
		err.AlertErr = fmt.Errorf("%s:%w", alert.ActionType, enginerr.ErrActionNotAvailable)
	} else {
		skipAlert = alertEngine.IsSkippable(ctx, actionsOnOff[alert.ActionType], evalErr)
	}

	// Exit early if both should be skipped
	if skipRemediate && skipAlert {
		return err
	}

	// Try remediating first
	if !skipRemediate {
		err.RemediateErr = remediateEngine.Do(ctx, actionsOnOff[remediate.ActionType], ent, ruleDef, ruleParams, dbEvalStatus)
	}

	// If remediate failed fatally and alert actions should not be skipped, try alerting
	if enginerr.IsActionFatalError(err.RemediateErr) && !skipAlert {
		err.AlertErr = alertEngine.Do(ctx, actionsOnOff[alert.ActionType], ent, ruleDef, ruleParams, dbEvalStatus)
	}

	return err
}

// GetActionsOnOffStates returns a map of the action states for all actions - on, off, ect.
func (rae *RuleActionsEngine) GetActionsOnOffStates(profile *mediatorv1.Profile) map[string]engif.ActionOpt {
	res := map[string]engif.ActionOpt{}
	for _, action := range rae.actions {
		res[action.Type()] = action.GetOnOffState(profile)
	}
	return res
}
