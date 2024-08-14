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

package rego

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/rego"

	engerrors "github.com/stacklok/minder/internal/engine/errors"
)

// EvaluationType is the type of evaluation to perform
type EvaluationType string

const (
	// DenyByDefaultEvaluationType is the deny-by-default evaluation type
	// It uses the rego query "data.minder.allow" to determine if the
	// object is allowed.
	DenyByDefaultEvaluationType EvaluationType = "deny-by-default"
	// ConstraintsEvaluationType is the constraints evaluation type
	// It uses the rego query "data.minder.violations[results]" to determine
	// if the object violates any constraints. If there are any violations,
	// the object is denied. Denials may contain a message specified through
	// the "msg" key.
	ConstraintsEvaluationType EvaluationType = "constraints"
)

func (e EvaluationType) String() string {
	return string(e)
}

// ConstraintsViolationsFormat is the format to output violations in
type ConstraintsViolationsFormat string

const (
	// ConstraintsViolationsOutputText specifies that the violations should be printed as human-readable text
	ConstraintsViolationsOutputText ConstraintsViolationsFormat = "text"
	// ConstraintsViolationsOutputJSON specifies that violations should be output as JSON
	ConstraintsViolationsOutputJSON ConstraintsViolationsFormat = "json"
)

func (c ConstraintsViolationsFormat) String() string {
	return string(c)
}

type resultEvaluator interface {
	getQuery() func(r *rego.Rego)
	parseResult(rs rego.ResultSet) error
}

type denyByDefaultEvaluator struct {
}

func (*denyByDefaultEvaluator) getQuery() func(r *rego.Rego) {
	return rego.Query(RegoQueryPrefix)
}

func (*denyByDefaultEvaluator) parseResult(rs rego.ResultSet) error {
	if len(rs) == 0 {
		return engerrors.NewErrEvaluationFailed("no results")
	}

	res := rs[0]

	if len(res.Expressions) == 0 {
		return engerrors.NewErrEvaluationFailed("no expressions")
	}

	// get first expression
	exprRaw := res.Expressions[0]
	exprVal := exprRaw.Value
	expr, ok := exprVal.(map[string]any)
	if !ok {
		return engerrors.NewErrEvaluationFailed("unable to get result expression")
	}

	// check if skipped
	skipped, ok := expr["skip"]
	if ok {
		skippedBool, ok := skipped.(bool)
		// if skipped is true, return skipped error
		if ok && skippedBool {
			return engerrors.NewErrEvaluationSkipped("rule not applicable")
		}
	}

	// check if allowed
	allowed, ok := expr["allow"]
	if !ok {
		return engerrors.NewErrEvaluationFailed("unable to get allow result")
	}

	allowedBool, ok := allowed.(bool)
	if !ok {
		return engerrors.NewErrEvaluationFailed("allow result is not a bool")
	}

	if allowedBool {
		return nil
	}

	return engerrors.NewErrEvaluationFailed("denied")
}

type constraintsEvaluator struct {
	format ConstraintsViolationsFormat
}

func (*constraintsEvaluator) getQuery() func(r *rego.Rego) {
	return rego.Query(fmt.Sprintf("%s.violations[details]", RegoQueryPrefix))
}

func (c *constraintsEvaluator) parseResult(rs rego.ResultSet) error {
	if len(rs) == 0 {
		// There were no violations
		return nil
	}

	// Gather violations into one
	resBuilder := c.resultsBuilder(rs)
	if resBuilder == nil {
		return fmt.Errorf("invalid format: %s", c.format)
	}
	for _, r := range rs {
		v, err := resultToViolation(r)
		if err != nil {
			return fmt.Errorf("unexpected error in rego violation: %w", err)
		}

		err = resBuilder.addResult(v)
		if err != nil {
			return fmt.Errorf("cannot add result: %w", err)
		}
	}

	return resBuilder.formatResults()
}

func (c *constraintsEvaluator) resultsBuilder(rs rego.ResultSet) resultBuilder {
	switch c.format {
	case ConstraintsViolationsOutputText:
		return newStringResultBuilder(rs)
	case ConstraintsViolationsOutputJSON:
		return newJSONResultBuilder(rs)
	default:
		return nil
	}
}

func resultToViolation(r rego.Result) (any, error) {
	det := r.Bindings["details"]
	if det == nil {
		return nil, fmt.Errorf("missing details in result")
	}

	detmap, ok := det.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("details is not a map")
	}

	msg, ok := detmap["msg"]
	if !ok {
		return nil, fmt.Errorf("missing msg in details")
	}

	return msg, nil
}

type resultBuilder interface {
	addResult(msg any) error
	formatResults() error
}

type stringResultBuilder struct {
	results []string
}

func newStringResultBuilder(rs rego.ResultSet) *stringResultBuilder {
	return &stringResultBuilder{
		results: make([]string, 0, len(rs)),
	}
}

func (srb *stringResultBuilder) addResult(msg any) error {
	msgstr, ok := msg.(string)
	if !ok {
		return fmt.Errorf("msg is not a string")
	}
	srb.results = append(srb.results, msgstr)
	return nil
}

func (srb *stringResultBuilder) formatResults() error {
	return engerrors.NewErrEvaluationFailed("Evaluation failures: \n - %s", strings.Join(srb.results, "\n - "))
}

type jsonResultBuilder struct {
	results []map[string]interface{}
}

func newJSONResultBuilder(rs rego.ResultSet) *jsonResultBuilder {
	return &jsonResultBuilder{
		results: make([]map[string]interface{}, 0, len(rs)),
	}
}

func (jrb *jsonResultBuilder) addResult(msg any) error {
	var result map[string]interface{}

	msgstr, ok := msg.(string)
	if !ok {
		return fmt.Errorf("msg is not a string")
	}

	err := json.NewDecoder(strings.NewReader(msgstr)).Decode(&result)
	if err != nil {
		// fallback
		result = map[string]interface{}{
			"msg": msgstr,
		}
	}

	jrb.results = append(jrb.results, result)
	return nil
}

func (jrb *jsonResultBuilder) formatResults() error {
	jsonArray, err := json.Marshal(jrb.results)
	if err != nil {
		return fmt.Errorf("failed to marshal violations: %w", err)
	}

	return engerrors.NewErrEvaluationFailed("%s", string(jsonArray))
}
