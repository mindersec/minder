// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

package rego

import (
	"errors"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/rego"

	engerrors "github.com/stacklok/mediator/internal/engine/errors"
)

// EvaluationType is the type of evaluation to perform
type EvaluationType string

const (
	// DenyByDefaultEvaluationType is the deny-by-default evaluation type
	// It uses the rego query "data.mediator.allow" to determine if the
	// object is allowed.
	DenyByDefaultEvaluationType EvaluationType = "deny-by-default"
	// ConstraintsEvaluationType is the constraints evaluation type
	// It uses the rego query "data.mediator.violations[results]" to determine
	// if the object violates any constraints. If there are any violations,
	// the object is denied. Denials may contain a message specified through
	// the "msg" key.
	ConstraintsEvaluationType EvaluationType = "constraints"
)

func (e EvaluationType) String() string {
	return string(e)
}

type resultEvaluator interface {
	getQuery() func(r *rego.Rego)
	parseResult(rs rego.ResultSet) error
}

type denyByDefaultEvaluator struct {
}

func (*denyByDefaultEvaluator) getQuery() func(r *rego.Rego) {
	return rego.Query("data.mediator.allow")
}

func (*denyByDefaultEvaluator) parseResult(rs rego.ResultSet) error {
	if len(rs) == 0 {
		return engerrors.NewErrEvaluationFailed("Evaluation failed: no results")
	}

	if rs.Allowed() {
		return nil
	}

	return engerrors.NewErrEvaluationFailed("denied")
}

type constraintsEvaluator struct {
}

func (*constraintsEvaluator) getQuery() func(r *rego.Rego) {
	return rego.Query("data.mediator.violations[details]")
}

func (*constraintsEvaluator) parseResult(rs rego.ResultSet) error {
	if len(rs) == 0 {
		// There were no violations
		return nil
	}

	// Gather violations into one
	violations := make([]string, 0, len(rs))
	for _, r := range rs {
		v := resultToViolation(r)
		if errors.Is(v, engerrors.ErrEvaluationFailed) {
			violations = append(violations, v.Error())
		} else {
			return fmt.Errorf("unexpected error in rego violation: %w", v)
		}
	}

	return engerrors.NewErrEvaluationFailed("Evaluation failures: \n - %s", strings.Join(violations, "\n - "))
}

func resultToViolation(r rego.Result) error {
	det := r.Bindings["details"]
	if det == nil {
		return fmt.Errorf("missing details in result")
	}

	detmap, ok := det.(map[string]interface{})
	if !ok {
		return fmt.Errorf("details is not a map")
	}

	msg, ok := detmap["msg"]
	if !ok {
		return fmt.Errorf("missing msg in details")
	}

	msgstr, ok := msg.(string)
	if !ok {
		return fmt.Errorf("msg is not a string")
	}

	return engerrors.NewErrEvaluationFailed(msgstr)
}
