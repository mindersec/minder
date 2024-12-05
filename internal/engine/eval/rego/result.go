// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rego

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/open-policy-agent/opa/rego"
	"google.golang.org/protobuf/reflect/protoreflect"

	engerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval/templates"
	pbinternal "github.com/mindersec/minder/internal/proto"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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
	getQuery() func(*rego.Rego)
	parseResult(rego.ResultSet, protoreflect.ProtoMessage) error
}

type denyByDefaultEvaluator struct {
}

func (*denyByDefaultEvaluator) getQuery() func(r *rego.Rego) {
	return rego.Query(RegoQueryPrefix)
}

func (*denyByDefaultEvaluator) parseResult(rs rego.ResultSet, entity protoreflect.ProtoMessage) error {
	// This usually happens when the provided Rego code is empty
	if len(rs) == 0 {
		return engerrors.NewDetailedErrEvaluationFailed(
			templates.RegoDenyByDefaultTemplate,
			map[string]any{
				"message": "no results",
			},
			"no results",
		)
	}

	res := rs[0]

	// This usually happens when the provided Rego code is empty
	if len(res.Expressions) == 0 {
		return engerrors.NewDetailedErrEvaluationFailed(
			templates.RegoDenyByDefaultTemplate,
			map[string]any{
				"message": "no expressions",
			},
			"no expressions",
		)
	}

	// get first expression
	exprRaw := res.Expressions[0]
	exprVal := exprRaw.Value
	expr, ok := exprVal.(map[string]any)
	if !ok {
		return engerrors.NewDetailedErrEvaluationFailed(
			templates.RegoDenyByDefaultTemplate,
			map[string]any{
				"message": "unable to get result expression",
			},
			"unable to get result expression",
		)
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
		return engerrors.NewDetailedErrEvaluationFailed(
			templates.RegoDenyByDefaultTemplate,
			map[string]any{
				"message": "unable to get allow result",
			},
			"unable to get allow result",
		)
	}

	allowedBool, ok := allowed.(bool)
	if !ok {
		return engerrors.NewDetailedErrEvaluationFailed(
			templates.RegoDenyByDefaultTemplate,
			map[string]any{
				"message": "allow result is not a bool",
			},
			"allow result is not a bool",
		)
	}

	if allowedBool {
		return nil
	}

	// check if custom message was provided
	var message string
	msg, ok := expr["message"]
	if ok {
		if message, ok = msg.(string); !ok {
			message = "denied"
		}
	}
	if message == "" {
		message = "denied"
	}

	entityName := getEntityName(entity)
	return engerrors.NewDetailedErrEvaluationFailed(
		templates.RegoDenyByDefaultTemplate,
		map[string]any{
			"message":    message,
			"entityName": entityName,
		},
		"denied",
	)
}

type constraintsEvaluator struct {
	format ConstraintsViolationsFormat
}

func (*constraintsEvaluator) getQuery() func(r *rego.Rego) {
	return rego.Query(fmt.Sprintf("%s.violations[details]", RegoQueryPrefix))
}

func (c *constraintsEvaluator) parseResult(rs rego.ResultSet, _ protoreflect.ProtoMessage) error {
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
	return engerrors.NewDetailedErrEvaluationFailed(
		templates.RegoConstraints,
		map[string]any{
			"violations": srb.results,
		},
		"Evaluation failures: \n - %s",
		strings.Join(srb.results, "\n - "),
	)
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
