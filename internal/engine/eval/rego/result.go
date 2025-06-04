// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rego

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/v1/rego"
	"google.golang.org/protobuf/reflect/protoreflect"

	engerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval/templates"
	pbinternal "github.com/mindersec/minder/internal/proto"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
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

// EvalOutputFormat is the format to output violations in
type EvalOutputFormat string

const (
	// OutputText specifies that the violations should be printed as human-readable text
	OutputText EvalOutputFormat = "text"
	// OutputJSON specifies that violations should be output as JSON
	OutputJSON EvalOutputFormat = "json"
)

func (c EvalOutputFormat) String() string {
	return string(c)
}

type resultEvaluator interface {
	parseResult(rego.ResultSet, protoreflect.ProtoMessage) (*interfaces.EvaluationResult, error)
}

type denyByDefaultEvaluator struct {
}

func (*denyByDefaultEvaluator) parseResult(rs rego.ResultSet, entity protoreflect.ProtoMessage,
) (*interfaces.EvaluationResult, error) {
	expr, err := getExports(rs)
	if err != nil {
		return nil, err
	}

	skipped, err := valueFromExpression[bool](expr, "skip")
	// Not found is the same as false (default value)
	if err != nil && !errors.Is(err, errNotFound) {
		return nil, err
	}
	if skipped {
		return nil, engerrors.NewErrEvaluationSkipped("rule not applicable")
	}

	allowed, err := valueFromExpression[bool](expr, "allow")
	if errors.Is(err, errNotFound) {
		return nil, engerrors.NewErrEvaluationFailed("unable to get allow result")
	} else if err != nil {
		return nil, err
	}

	result := &interfaces.EvaluationResult{}

	if allowed {
		return result, nil
	}

	// check if custom message was provided
	message, err := valueFromExpression[string](expr, "message")
	if err != nil && !errors.Is(err, errNotFound) {
		return nil, err
	}
	message = cmp.Or(message, "denied")

	// We don't need the error here; if the output can't be parsed, we
	// *always* fall back to the message.
	result.Output, _ = valueFromExpression[any](expr, "output")
	if result.Output == nil {
		result.Output = message
	}

	entityName := getEntityName(entity)
	return result, engerrors.NewDetailedErrEvaluationFailed(
		templates.RegoDenyByDefaultTemplate,
		map[string]any{
			"message":    message,
			"entityName": entityName,
		},
		"denied",
	)
}

// errNotFound is only used to signal that the key was not found in valueFromExpression
var errNotFound = errors.New("not found")

// valueFromExpression is a helper to fetch a typed value from a JSON object
// if the value is found, it returns a nil error.  If not, it returns either
// errNotFound if the field was not found, or an EvaluationFailed if the
// field was found but was of the wrong type.
func valueFromExpression[T any](object map[string]any, key string) (T, error) {
	var ret T
	value, ok := object[key]
	if !ok {
		return ret, errNotFound
	}

	ret, ok = value.(T)
	if !ok {
		return ret, engerrors.NewErrEvaluationFailed("%s result is not a %T", key, ret)
	}

	return ret, nil
}

type constraintsEvaluator struct {
	format EvalOutputFormat
}

func (c *constraintsEvaluator) parseResult(rs rego.ResultSet, _ protoreflect.ProtoMessage) (*interfaces.EvaluationResult, error) {
	expr, err := getExports(rs)
	if err != nil {
		return nil, err
	}

	skipped, err := valueFromExpression[bool](expr, "skip")
	// Not found is the same as false (default value)
	if err != nil && !errors.Is(err, errNotFound) {
		return nil, err
	}
	if skipped {
		return nil, engerrors.NewErrEvaluationSkipped("rule not applicable")
	}

	violations, ok := expr["violations"].([]any)
	if !ok {
		return nil, engerrors.NewErrEvaluationFailed(
			"unable to get violations array, found %T",
			expr["violations"],
		)
	}

	result := &interfaces.EvaluationResult{}
	if len(violations) == 0 {
		// On success, there's no need to pass along any further data, so early return.
		return result, nil
	}

	resBuilder := c.resultsBuilder(violations)
	for _, v := range violations {
		msg, err := resultToViolation(v)
		if err != nil {
			return nil, engerrors.NewErrEvaluationFailed("%s", err)
		}

		if err := resBuilder.addViolation(msg); err != nil {
			return nil, engerrors.NewErrEvaluationFailed("cannot add result: %s", err)
		}
	}

	// We don't need the error here; if the output can't be parsed, we
	// *always* fall back to the message.
	result.Output, _ = valueFromExpression[any](expr, "output")
	if result.Output == nil {
		result.Output = resBuilder.violationsAsOutput()
	}

	return result, resBuilder.formatResults()
}

func (c *constraintsEvaluator) resultsBuilder(rs []any) resultBuilder {
	switch c.format {
	case OutputText:
		return newStringResultBuilder(rs)
	case OutputJSON:
		return newJSONResultBuilder(rs)
	default:
		return nil
	}
}

func resultToViolation(result any) (string, error) {
	r, ok := result.(map[string]any)
	if !ok {
		return "", fmt.Errorf("wrong type for violation: %T", result)
	}
	msg, ok := r["msg"]
	if !ok {
		return "", fmt.Errorf("missing msg in details")
	}

	msgstr, ok := msg.(string)
	if !ok {
		return "", errors.New("msg is not a string")
	}

	return msgstr, nil
}

type resultBuilder interface {
	addViolation(msg string) error
	formatResults() error
	violationsAsOutput() []any
}

type stringResultBuilder struct {
	results []string
}

func newStringResultBuilder(rs []any) *stringResultBuilder {
	return &stringResultBuilder{
		results: make([]string, 0, len(rs)),
	}
}

func (srb *stringResultBuilder) addViolation(msg string) error {
	srb.results = append(srb.results, msg)
	return nil
}

func (srb *stringResultBuilder) formatResults() error {
	return engerrors.NewDetailedErrEvaluationFailed(
		templates.RegoConstraints,
		map[string]any{
			"violations": srb.results,
		},
		"Evaluation failures: \n - %s",
		strings.Join(srb.results, "\n - "),
	)
}

func (srb *stringResultBuilder) violationsAsOutput() []any {
	res := make([]any, 0, len(srb.results))
	for _, r := range srb.results {
		res = append(res, r)
	}
	return res
}

type jsonResultBuilder struct {
	results []map[string]any
}

func newJSONResultBuilder(rs []any) *jsonResultBuilder {
	return &jsonResultBuilder{
		results: make([]map[string]interface{}, 0, len(rs)),
	}
}

func (jrb *jsonResultBuilder) addViolation(msg string) error {
	var result map[string]interface{}

	if err := json.Unmarshal([]byte(msg), &result); err != nil {
		// fallback
		result = map[string]interface{}{
			"msg": msg,
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

func (jrb *jsonResultBuilder) violationsAsOutput() []any {
	res := make([]any, 0, len(jrb.results))
	for _, r := range jrb.results {
		res = append(res, r)
	}
	return res
}

// getExports expects to be called with a rego module, and extracts a map
// (JSON object) of the results
func getExports(rs rego.ResultSet) (rego.Vars, error) {
	// Since we ask for the module, we only get no results if the module
	// is not defined.
	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return nil, engerrors.NewDetailedErrEvaluationFailed(
			templates.RegoDenyByDefaultTemplate,
			map[string]any{
				"message": "no results from Rego eval",
			},
			"no results from Rego eval",
		)
	}

	// get first expression
	exprVal := rs[0].Expressions[0].Value
	expr, ok := exprVal.(map[string]any)
	if !ok {
		return nil, engerrors.NewDetailedErrEvaluationFailed(
			templates.RegoDenyByDefaultTemplate,
			map[string]any{
				"message": "unable to get result expression",
			},
			"unable to get result expression",
		)
	}

	return expr, nil
}

func getEntityName(entity protoreflect.ProtoMessage) string {
	switch inner := entity.(type) {
	case *pbinternal.PullRequest:
		return fmt.Sprintf("%s/%s#%d",
			inner.RepoOwner,
			inner.RepoName,
			inner.Number,
		)
	case *minderv1.Repository:
		return fmt.Sprintf("%s/%s",
			inner.Owner,
			inner.Name,
		)
	case *minderv1.Artifact:
		return fmt.Sprintf("%s/%s (%s)",
			inner.Owner,
			inner.Name,
			inner.Type,
		)
	default:
		return ""
	}
}
