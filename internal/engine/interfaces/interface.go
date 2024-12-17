// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package interfaces provides necessary interfaces and implementations for
// implementing engine plugins
package interfaces

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/db"
	evalerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/profiles/models"
)

// ActionType represents the type of action, i.e., remediate, alert, etc.
type ActionType string

// Action is the interface for a rule type action
type Action interface {
	Class() ActionType
	Type() string
	GetOnOffState() models.ActionOpt
	Do(ctx context.Context, cmd ActionCmd, entity protoreflect.ProtoMessage,
		params ActionsParams, metadata *json.RawMessage) (json.RawMessage, error)
}

// AggregatingAction is the interface for an action that aggregates multiple
// pieces to form a final action. Normally this will come from the result of a
// `Do` call on an action.
type AggregatingAction interface {
	Flush(item ...any) error
}

// ActionCmd is the type that defines what effect an action should have
type ActionCmd string

const (
	// ActionCmdOff means turn off the action
	ActionCmdOff ActionCmd = "turn_off"
	// ActionCmdOn means turn on the action
	ActionCmdOn ActionCmd = "turn_on"
	// ActionCmdDoNothing means the action should do nothing
	ActionCmdDoNothing ActionCmd = "do_nothing"
)

// EvalStatusParams is a helper struct to pass parameters to createOrUpdateEvalStatus
// to avoid confusion with the parameters' order. Since at the moment, all our entities are bound to
// a repo and most profiles are expecting a repo, the RepoID parameter is mandatory. For entities
// other than artifacts, the ArtifactID should be 0 that is translated to NULL in the database.
type EvalStatusParams struct {
	Result           *interfaces.Result
	Profile          *models.ProfileAggregate
	Rule             *models.RuleInstance
	ProjectID        uuid.UUID
	ReleaseID        uuid.UUID
	PipelineRunID    uuid.UUID
	TaskRunID        uuid.UUID
	BuildID          uuid.UUID
	EntityType       db.Entities
	EntityID         uuid.UUID
	EvalStatusFromDb *db.ListRuleEvaluationsByProfileIdRow
	evalErr          error
	evalResult       *interfaces.EvaluationResult
	actionsErr       evalerrors.ActionsError
	ExecutionID      uuid.UUID
}

// Ensure EvalStatusParams implements the necessary interfaces
var _ EvalParamsReader = (*EvalStatusParams)(nil)
var _ interfaces.ResultSink = (*EvalStatusParams)(nil)

// GetEvalErr returns the evaluation error
func (e *EvalStatusParams) GetEvalErr() error {
	return e.evalErr
}

// SetEvalErr sets the evaluation error
func (e *EvalStatusParams) SetEvalErr(err error) {
	e.evalErr = err
}

// GetEvalResult returns the evaluation result
func (e *EvalStatusParams) GetEvalResult() *interfaces.EvaluationResult {
	return e.evalResult
}

// SetEvalResult sets the evaluation result for use later on in the actions
func (e *EvalStatusParams) SetEvalResult(res *interfaces.EvaluationResult) {
	e.evalResult = res
}

// SetActionsErr sets the actions' error
func (e *EvalStatusParams) SetActionsErr(ctx context.Context, actionErr evalerrors.ActionsError) {
	// Get logger
	logger := zerolog.Ctx(ctx)

	// Make sure we don't try to push a nil json.RawMessage accidentally
	if actionErr.AlertMeta == nil {
		// Default to an empty json struct if the action did not return anything
		m, err := json.Marshal(&map[string]any{})
		if err != nil {
			// This should never happen since we are marshaling an empty struct
			logger.Error().Err(err).Msg("error marshaling empty json.RawMessage")
		}
		actionErr.AlertMeta = m
	}
	if actionErr.RemediateMeta == nil {
		// Default to an empty json struct if the action did not return anything
		m, err := json.Marshal(&map[string]any{})
		if err != nil {
			// This should never happen since we are marshaling an empty struct
			logger.Error().Err(err).Msg("error marshaling empty json.RawMessage")
		}
		actionErr.RemediateMeta = m
	}
	// All okay
	e.actionsErr = actionErr
}

// GetActionsErr returns the actions' error
func (e *EvalStatusParams) GetActionsErr() evalerrors.ActionsError {
	return e.actionsErr
}

// GetRule returns the rule
func (e *EvalStatusParams) GetRule() *models.RuleInstance {
	return e.Rule
}

// GetEvalStatusFromDb returns the evaluation status from the database
// Returns nil if there is no previous state for this rule/entity
func (e *EvalStatusParams) GetEvalStatusFromDb() *db.ListRuleEvaluationsByProfileIdRow {
	return e.EvalStatusFromDb
}

// GetProfile returns the profile
func (e *EvalStatusParams) GetProfile() *models.ProfileAggregate {
	return e.Profile
}

// SetIngestResult sets the result of the ingestion for use later on in the actions
func (e *EvalStatusParams) SetIngestResult(res *interfaces.Result) {
	e.Result = res
}

// GetIngestResult returns the result of the ingestion, if any
func (e *EvalStatusParams) GetIngestResult() *interfaces.Result {
	return e.Result
}

// DecorateLogger decorates the logger with the necessary fields
func (e *EvalStatusParams) DecorateLogger(l zerolog.Logger) zerolog.Logger {
	outl := l.With().
		Str("entity_type", string(e.EntityType)).
		Str("profile_id", e.Profile.ID.String()).
		Str("rule_name", e.GetRule().Name).
		Str("execution_id", e.ExecutionID.String()).
		Str("rule_type_id", e.Rule.RuleTypeID.String()).
		Str("entity_id", e.EntityID.String()).
		Logger()

	return outl
}

// EvalParamsReader is the interface used for a rule type evaluator
type EvalParamsReader interface {
	GetRule() *models.RuleInstance
	GetIngestResult() *interfaces.Result
}

// ActionsParams is the interface used for processing a rule type action
type ActionsParams interface {
	EvalParamsReader
	interfaces.ResultSink
	GetActionsErr() evalerrors.ActionsError
	GetEvalErr() error
	GetEvalResult() *interfaces.EvaluationResult
	GetEvalStatusFromDb() *db.ListRuleEvaluationsByProfileIdRow
	GetProfile() *models.ProfileAggregate
}
