// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/engine/actions/alert"
	"github.com/stacklok/minder/internal/engine/actions/remediate"
	"github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/interfaces"
)

type key int

const (
	// telemetryContextKey is a key used to store the current telemetry record's context
	telemetryContextKey key = iota
)

// RuleType is a struct describing a rule type for telemetry purposes
type RuleType struct {
	Name string    `json:"name"`
	ID   uuid.UUID `json:"id"`
}

// Profile is a struct describing a Profile for telemetry purposes
type Profile struct {
	Name string    `json:"name"`
	ID   uuid.UUID `json:"id"`
}

// ActionEvalData reports
type ActionEvalData struct {
	// how was the action configured - on, off, ...
	State string `json:"state"`
	// what was the result of the action - success, failure, ...
	Result string `json:"result"`
}

// RuleEvalData reports
type RuleEvalData struct {
	RuleType RuleType `json:"ruletype"`
	Profile  Profile  `json:"profile"`

	EvalResult string                                   `json:"eval_result"`
	Actions    map[interfaces.ActionType]ActionEvalData `json:"actions"`

	// TODO: do we want to store params?
}

// ProjectTombstone can be used to store project metadata in the context of deletion.
type ProjectTombstone struct {
	// Project records the project ID that the request was associated with.
	Project uuid.UUID `json:"project"`

	// ProfileCount records the number of profiles associated with the project.
	ProfileCount int `json:"profile_count"`
	// RepositoriesCount records the number of repositories associated with the project.
	RepositoriesCount int `json:"repositories_count"`
	// Entitlements that the projects has.
	Entitlements []string `json:"entitlements"`
}

// Equals compares two ProjectTombstone structs for equality.
func (pt ProjectTombstone) Equals(other ProjectTombstone) bool {
	return pt.Project == other.Project &&
		pt.ProfileCount == other.ProfileCount &&
		pt.RepositoriesCount == other.RepositoriesCount &&
		slices.Equal(pt.Entitlements, other.Entitlements)
}

// TelemetryStore is a struct that can be used to store telemetry data in the context.
type TelemetryStore struct {
	// Project records the project ID that the request was associated with.
	Project uuid.UUID `json:"project"`

	// Provider records the provider name that the request was associated with.
	Provider string `json:"provider"`

	// ProviderID records the provider ID that the request was associated with.
	ProviderID uuid.UUID `json:"provider_id"`

	// Repository is the repository ID that the request was associated with.
	Repository uuid.UUID `json:"repository"`

	// Artifact is the artifact ID that the request was associated with.
	Artifact uuid.UUID `json:"artifact"`

	// PullRequest is the pull request ID that the request was associated with.
	PullRequest uuid.UUID `json:"pr"`

	// Profile is the profile that the request was associated with.
	Profile Profile `json:"profile"`

	// RuleType is the rule type that the request was associated with.
	RuleType RuleType `json:"ruletype"`

	// Data from RPCs

	// Hashed (SHA256) `sub` from the JWT.  This should be hard to reverse (pseudonymized),
	// but allows correlation between requests.
	LoginHash string `json:"login_sha"`

	// Data from event processing; may be empty (for example, for RPCs)

	// Rules evaluated during processing
	Evals []RuleEvalData `json:"rules"`

	// Metadata about the project tombstone
	ProjectTombstone ProjectTombstone `json:"project_tombstone"`
}

// AddRuleEval is a convenience method to add a rule evaluation result to the telemetry store.
func (ts *TelemetryStore) AddRuleEval(
	evalInfo interfaces.ActionsParams,
	ruleTypeName string,
) {
	if ts == nil {
		return
	}

	red := RuleEvalData{
		RuleType:   RuleType{Name: ruleTypeName, ID: evalInfo.GetRule().RuleTypeID},
		Profile:    Profile{Name: evalInfo.GetProfile().Name, ID: evalInfo.GetProfile().ID},
		EvalResult: errors.EvalErrorAsString(evalInfo.GetEvalErr()),
		Actions: map[interfaces.ActionType]ActionEvalData{
			remediate.ActionType: {
				State:  evalInfo.GetActionsOnOff()[remediate.ActionType].String(),
				Result: errors.RemediationErrorAsString(evalInfo.GetActionsErr().RemediateErr),
			},
			alert.ActionType: {
				State:  evalInfo.GetActionsOnOff()[alert.ActionType].String(),
				Result: errors.AlertErrorAsString(evalInfo.GetActionsErr().AlertErr),
			},
		},
	}

	ts.Evals = append(ts.Evals, red)
}

// BusinessRecord provides the ability to store an observation about the current
// flow of business logic in the context of the current request.  When called in
// in the context of a logged action, it will record and send the marshalled data
// to the logging system.
//
// When called outside a logged context, it will collect and discard the data.
func BusinessRecord(ctx context.Context) *TelemetryStore {
	ts, ok := ctx.Value(telemetryContextKey).(*TelemetryStore)
	if !ok {
		// return a dummy value, to make it easy to chain this call.
		return &TelemetryStore{}
	}
	// Intentionally allowing aliasing here, we want to collect all data for one
	// from different execution points, then write it out at completion.
	return ts
}

// WithTelemetry enriches the current context with a TelemetryStore which will
// collect observations about the current flow of business logic.
func (ts *TelemetryStore) WithTelemetry(ctx context.Context) context.Context {
	if ts == nil {
		return ctx
	}
	return context.WithValue(ctx, telemetryContextKey, ts)
}

// Record adds the collected data to the supplied event record.
func (ts *TelemetryStore) Record(e *zerolog.Event) *zerolog.Event {
	if ts == nil {
		return e
	}
	// We could use reflection here like json.Marshal, but given
	// the small number of fields, we'll just add them explicitly.
	if ts.Project != uuid.Nil {
		e.Str("project", ts.Project.String())
	}
	if ts.Provider != "" {
		e.Str("provider", ts.Provider)
	}
	if ts.ProviderID != uuid.Nil {
		e.Str("provider_id", ts.ProviderID.String())
	}
	if ts.LoginHash != "" {
		e.Str("login_sha", ts.LoginHash)
	}
	if ts.Repository != uuid.Nil {
		e.Str("repository", ts.Repository.String())
	}
	if ts.Artifact != uuid.Nil {
		e.Str("artifact", ts.Artifact.String())
	}
	if ts.PullRequest != uuid.Nil {
		e.Str("pr", ts.PullRequest.String())
	}
	if ts.Profile != (Profile{}) {
		e.Any("profile", ts.Profile)
	}
	if ts.RuleType != (RuleType{}) {
		e.Any("ruletype", ts.RuleType)
	}
	if !ts.ProjectTombstone.Equals(ProjectTombstone{}) {
		e.Any("project_tombstone", ts.ProjectTombstone)
	}
	if len(ts.Evals) > 0 {
		e.Any("rules", ts.Evals)
	}
	e.Str("telemetry", "true")
	// Note: we explicitly don't call e.Send() here so that Send() occurs in the
	// same scope as the event is created.
	return e
}
