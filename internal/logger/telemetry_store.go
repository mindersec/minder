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

	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/engine/interfaces"
)

type key int

const (
	// telemetryContextKey is a key used to store the current telemetry record's context
	telemetryContextKey key = iota
)

// RuleEvalData reports
type RuleEvalData struct {
	RuleName string `json:"name"`
	// Action is an ActionOpt string
	Action interfaces.ActionOpt `json:"action"`

	// TODO: how to store results of evaluation?  We probably want to cover:
	// - rule skipped
	// - rule eval failed
	// - rule eval passed, no action needed
	// - rule eval passed, took <store, GHSA, PR, remediate, etc> action

	// TODO: do we want to store params?
}

// TelemetryStore is a struct that can be used to store telemetry data in the context.
type TelemetryStore struct {
	// Project records the project that the request was associated with.
	Project string `json:"project"`

	// The resource processed by the request, for example, a repository or a profile.
	Resource string `json:"resource"`

	// Data from RPCs

	// Hashed (SHA256) `sub` from the JWT.  This should be hard to reverse (pseudonymized),
	// but allows correlation between requests.
	LoginHash string `json:"login_sha"`

	// Data from event processing; may be empty (for example, for RPCs)

	// Rules evaluated during processing
	Evals []RuleEvalData `json:"rules"`
}

// AddRuleEval is a convenience method to add a rule evaluation result to the telemetry store.
func (ts *TelemetryStore) AddRuleEval(ruleName string, action interfaces.ActionOpt) {
	if ts == nil {
		return
	}
	ts.Evals = append(ts.Evals, RuleEvalData{
		RuleName: ruleName, Action: action,
	})
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
	if ts.Project != "" {
		e.Str("project", ts.Project)
	}
	if ts.Resource != "" {
		e.Str("resource", ts.Resource)
	}
	if ts.LoginHash != "" {
		e.Str("login_sha", ts.LoginHash)
	}
	if len(ts.Evals) > 0 {
		e.Any("rules", ts.Evals)
	}
	e.Bool("telemetry", true)
	// Note: we explicitly don't call e.Send() here so that Send() occurs in the
	// same scope as the event is created.
	return e
}
