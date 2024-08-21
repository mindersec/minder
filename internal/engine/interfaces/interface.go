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

// Package interfaces provides necessary interfaces and implementations for
// implementing engine plugins
package interfaces

import (
	"context"
	"encoding/json"

	billy "github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/storage"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/minder/internal/db"
	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/entities/checkpoints"
	"github.com/stacklok/minder/internal/profiles/models"
)

// Ingester is the interface for a rule type ingester
type Ingester interface {
	// Ingest does the actual data ingestion for a rule type
	Ingest(ctx context.Context, ent protoreflect.ProtoMessage, params map[string]any) (*Result, error)
	// GetType returns the type of the ingester
	GetType() string
	// GetConfig returns the config for the ingester
	GetConfig() protoreflect.ProtoMessage
}

// Evaluator is the interface for a rule type evaluator
type Evaluator interface {
	Eval(ctx context.Context, profile map[string]any, res *Result) error
}

// Result is the result of an ingester
type Result struct {
	// Object is the object that was ingested. Normally comes from an external
	// system like an HTTP server.
	Object any
	// Fs is the filesystem that was created as a result of the ingestion. This
	// is normally used by the evaluator to do rule evaluation. The filesystem
	// may be a git repo, or a memory filesystem.
	Fs billy.Filesystem
	// Storer is the git storer that was created as a result of the ingestion.
	// FIXME: It might be cleaner to either wrap both Fs and Storer in a struct
	// or pass out the git.Repository structure instead of the storer.
	Storer storage.Storer

	// Checkpoint is the checkpoint at which the ingestion was done. This is
	// used to persist the state of the entity at ingestion time.
	Checkpoint *checkpoints.CheckpointEnvelopeV1
}

// GetCheckpoint returns the checkpoint of the result
func (r *Result) GetCheckpoint() *checkpoints.CheckpointEnvelopeV1 {
	if r == nil {
		return nil
	}

	return r.Checkpoint
}

// ActionType represents the type of action, i.e., remediate, alert, etc.
type ActionType string

// Action is the interface for a rule type action
type Action interface {
	Class() ActionType
	Type() string
	GetOnOffState(models.ActionOpt) models.ActionOpt
	Do(ctx context.Context, cmd ActionCmd, setting models.ActionOpt, entity protoreflect.ProtoMessage,
		params ActionsParams, metadata *json.RawMessage) (json.RawMessage, error)
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
	Result           *Result
	Profile          *models.ProfileAggregate
	Rule             *models.RuleInstance
	RepoID           uuid.NullUUID
	ArtifactID       uuid.NullUUID
	PullRequestID    uuid.NullUUID
	ProjectID        uuid.UUID
	ReleaseID        uuid.UUID
	PipelineRunID    uuid.UUID
	TaskRunID        uuid.UUID
	BuildID          uuid.UUID
	EntityType       db.Entities
	EntityID         uuid.UUID
	EvalStatusFromDb *db.ListRuleEvaluationsByProfileIdRow
	evalErr          error
	actionsOnOff     map[ActionType]models.ActionOpt
	actionsErr       evalerrors.ActionsError
	ExecutionID      uuid.UUID
}

// Ensure EvalStatusParams implements the necessary interfaces
var _ ActionsParams = (*EvalStatusParams)(nil)
var _ EvalParamsReader = (*EvalStatusParams)(nil)
var _ EvalParamsReadWriter = (*EvalStatusParams)(nil)

// GetEvalErr returns the evaluation error
func (e *EvalStatusParams) GetEvalErr() error {
	return e.evalErr
}

// SetEvalErr sets the evaluation error
func (e *EvalStatusParams) SetEvalErr(err error) {
	e.evalErr = err
}

// GetActionsOnOff returns the actions' on/off state
func (e *EvalStatusParams) GetActionsOnOff() map[ActionType]models.ActionOpt {
	return e.actionsOnOff
}

// SetActionsOnOff sets the actions' on/off state
func (e *EvalStatusParams) SetActionsOnOff(actionsOnOff map[ActionType]models.ActionOpt) {
	e.actionsOnOff = actionsOnOff
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
func (e *EvalStatusParams) SetIngestResult(res *Result) {
	e.Result = res
}

// GetIngestResult returns the result of the ingestion, if any
func (e *EvalStatusParams) GetIngestResult() *Result {
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
		Logger()
	if e.RepoID.Valid {
		outl = outl.With().Str("repository_id", e.RepoID.UUID.String()).Logger()
	}

	if e.ArtifactID.Valid {
		outl = outl.With().Str("artifact_id", e.ArtifactID.UUID.String()).Logger()
	}
	if e.PullRequestID.Valid {
		outl = outl.With().Str("pull_request_id", e.PullRequestID.UUID.String()).Logger()
	}

	return outl
}

// EvalParamsReader is the interface used for a rule type evaluator
type EvalParamsReader interface {
	GetRule() *models.RuleInstance
	GetIngestResult() *Result
}

// EvalParamsReadWriter is the interface used for a rule type engine, allows setting the ingestion result
type EvalParamsReadWriter interface {
	EvalParamsReader
	SetIngestResult(*Result)
}

// ActionsParams is the interface used for processing a rule type action
type ActionsParams interface {
	EvalParamsReader
	GetActionsOnOff() map[ActionType]models.ActionOpt
	GetActionsErr() evalerrors.ActionsError
	GetEvalErr() error
	GetEvalStatusFromDb() *db.ListRuleEvaluationsByProfileIdRow
	GetProfile() *models.ProfileAggregate
}
