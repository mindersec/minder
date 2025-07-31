// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rego_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	engerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval/rego"
	"github.com/mindersec/minder/internal/engine/options"
	"github.com/mindersec/minder/internal/util/ptr"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
	v1mockds "github.com/mindersec/minder/pkg/datasources/v1/mock"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

func TestEvalNoDef(t *testing.T) {
	t.Parallel()

	_, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
		},
	)

	assert.ErrorContains(t, err, "could not parse rego config:")
}

func TestEvalDefEmpty(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def:  ` `,
		},
	)

	require.NoError(t, err, "expected successful creation of evaluator")

	emptyPol := map[string]any{}
	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{},
	})
	assert.ErrorContains(t, err, "rego_parse_error: empty module")
}

func TestEvalDefWrongPackage(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def:  "package badday\n\nallow := true",
		},
	)

	require.NoError(t, err, "expected successful creation of evaluator")

	emptyPol := map[string]any{}
	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{},
	})
	assert.ErrorContains(t, err, "evaluation failure: no results from Rego eval")
}

func TestEvalDefNoOutput(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def:  "package minder\n#Does not define allow",
		},
	)

	require.NoError(t, err, "expected successful creation of evaluator")

	emptyPol := map[string]any{}
	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{},
	})
	assert.ErrorContains(t, err, "evaluation failure: unable to get allow result")
}

// Evaluates a simple query against a simple profile
// In this case, the profile is a simple "allow" rule.
// The given profile map is empty since all the profile
// needed in ths test case is contained in the rego
// definition.
func TestEvaluatorDenyByDefaultEvalSimple(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	input.ingested.data == "foo"
}
	
message = "By the pricking of my thumbs..."
`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	// Matches
	res, err := e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data": "foo",
		},
	})
	assert.NoError(t, err, "could not evaluate")
	assert.Equal(t, &interfaces.EvaluationResult{}, res)

	// Doesn't match
	res, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data": "bar",
		},
	})
	assert.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "should have failed the evaluation")
	assert.Equal(t, &interfaces.EvaluationResult{
		Output: "By the pricking of my thumbs...",
	}, res)
}

func TestEvaluatorDenyByDefaultEvalWrongType(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

allow := "false"`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}
	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{},
	})
	assert.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "should have failed the evaluation")
	assert.ErrorContains(t, err, "evaluation failure: allow result is not a bool")
}

func TestEvaluatorDenyByDefaultSkip(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	input.ingested.data == "foo"
}

skip {
	true
}
`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	// Doesn't match
	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data": "bar",
		},
	})
	assert.ErrorIs(t, err, interfaces.ErrEvaluationSkipped, "should have been skipped, %+v", err)
}

func TestEvaluatorDenyByDefaultSkipWrongType(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

allow := true

skip := "true"
`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}
	res, err := e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{},
	})
	assert.ErrorContains(t, err, "evaluation failure: skip result is not a bool")
	assert.Nil(t, res)
}

func TestEvaluatorDenyByDefaultJSONOutput(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type:            rego.DenyByDefaultEvaluationType.String(),
			ViolationFormat: ptr.Ptr(rego.OutputJSON.String()),
			Def: `
package minder

allow := false

message := "always fail"
output := [1, 2, 3]
`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}
	res, err := e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{},
	})
	assert.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "expected evaulation failed, got %+v", err)
	assert.ErrorContains(t, err, "evaluation failure: denied")
	assert.Equal(t, err.(*engerrors.EvaluationError).Details(), "always fail")
	assert.Equal(t, &interfaces.EvaluationResult{
		Output: []any{json.Number("1"), json.Number("2"), json.Number("3")},
	}, res)
}

func TestEvaluatorDenyByDefaultJSONOutputFromMessage(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			// ViolationFormat is not used for deny_by_default
			ViolationFormat: ptr.Ptr(rego.OutputJSON.String()),
			Def: `
package minder

allow := false

message := "[1, 2, 3]"
`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}
	res, err := e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{},
	})
	assert.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "expected evaulation failed, got %+v", err)
	assert.Equal(t, &interfaces.EvaluationResult{
		// Output is just a string when promoted from message.
		Output: "[1, 2, 3]",
	}, res)
}

func TestEvaluatorDenyByConstraintsEvalSimple(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package minder

violations[{"msg": msg}] {
	input.ingested.data != "foo"
	msg := "data did not contain foo"
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	// Matches
	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data": "foo",
		},
	})
	require.NoError(t, err)

	// Doesn't match
	res, err := e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data": "bar",
		},
	})
	assert.ErrorIs(t, err, interfaces.ErrEvaluationFailed)
	assert.ErrorContains(t, err, "data did not contain foo")
	assert.Equal(t, []any{"data did not contain foo"}, res.Output)
}

func TestEvaluatorConstraintWithOutput(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package minder

messages = ["one", "two"]
violations[{"msg": msg}] {
    msg := messages[_]
}

output := {"messages_len": 2, "messages": messages}
`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	res, err := e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{"data": "foo"},
	})

	assert.ErrorIs(t, err, interfaces.ErrEvaluationFailed)
	assert.ErrorContains(t, err, "evaluation failure: Evaluation failures: \n - one\n - two")
	assert.Equal(t,
		map[string]any{"messages_len": json.Number("2"), "messages": []any{"one", "two"}},
		res.Output)
}

func TestEvaluatorDenyByConstraintsEvalMultiple(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package minder

violations[{"msg": msg}] {
	input.ingested.data == "foo"
	msg := "data should not contain foo"
}

violations[{"msg": msg}] {
	input.ingested.datum == "bar"
	msg := "datum should not contain bar"
}
`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	res, err := e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data":  "foo",
			"datum": "bar",
		},
	})
	assert.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "should have failed the evaluation")
	assert.ErrorContains(t, err, "- data should not contain foo\n")
	assert.ErrorContains(t, err, "- datum should not contain bar")
	assert.Equal(t, []any{"data should not contain foo", "datum should not contain bar"}, res.Output)
}

func TestEvaluatorConstraintWrongType(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package minder

violations := {"msg": "I forgot the list"}
`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{"data": "foo"},
	})

	assert.ErrorIs(t, err, interfaces.ErrEvaluationFailed)
	assert.ErrorContains(t, err, "evaluation failure: unable to get violations array, found map[string]interface {}")
}

func TestEvaluatorConstraintWrongTypeInArray(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package minder

violations[msg] {
  msg := "I forgot the list"
}
`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{"data": "foo"},
	})

	assert.ErrorIs(t, err, interfaces.ErrEvaluationFailed)
	assert.ErrorContains(t, err, "wrong type for violation: string")
}

func TestEvaluatorConstraintWrongKeyInMap(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package minder

violations[{"problem": msg}] {
  msg := "this is the wrong key!"
}
`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{"data": "foo"},
	})

	assert.ErrorIs(t, err, interfaces.ErrEvaluationFailed)
	assert.ErrorContains(t, err, "missing msg in details")
}

func TestEvaluatorConstraintWrongTypeInObject(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package minder

violations[{"msg": msg}] {
  msg := {"detail": "this is the wrong key!"}
}
`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{"data": "foo"},
	})

	assert.ErrorIs(t, err, interfaces.ErrEvaluationFailed)
	assert.ErrorContains(t, err, "msg is not a string")
}

// Evaluates a simple query against a simple profile
// In this case, the profile is a simple "allow" rule.
// The given profile map has a value for the "data" key
// which is used in the rego definition. The ingested
// data has to match the profile data.
func TestDenyByDefaultEvaluationWithProfile(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	input.profile.data == input.ingested.data
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	pol := map[string]any{
		"data": "foo",
	}

	// Matches
	_, err = e.Eval(context.Background(), pol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data": "foo",
		},
	})
	require.NoError(t, err, "could not evaluate")

	// Doesn't match
	_, err = e.Eval(context.Background(), pol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data": "bar",
		},
	})
	require.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "should have failed the evaluation")
}

func TestConstrainedEvaluationWithProfile(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package minder

violations[{"msg": msg}] {
	input.profile.data != input.ingested.data
	msg := sprintf("data did not match profile: %s", [input.profile.data])
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	pol := map[string]any{
		"data": "foo",
	}

	// Matches
	_, err = e.Eval(context.Background(), pol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data": "foo",
		},
	})
	require.NoError(t, err, "could not evaluate")

	// Doesn't match
	res, err := e.Eval(context.Background(), pol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data": "bar",
		},
	})
	require.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "should have failed the evaluation")
	assert.ErrorContains(t, err, "data did not match profile: foo", "should have failed the evaluation")
	assert.Equal(t, []any{"data did not match profile: foo"}, res.Output)
}

const (
	jsonPolicyDef = `
package minder

violations[{"msg": msg}] {
  expected_set := {x | x := input.profile.data[_]}
  input_set := {x | x := input.ingested.data[_]}

  intersection := expected_set & input_set
  not count(intersection) == count(input.ingested.data)

  difference := [x | x := input.ingested.data[_]; not intersection[x]]
  
  msg = format_message(difference, input.output_format)
}

format_message(difference, format) = msg {
    format == "json"
	json_body := {"actions_not_allowed": difference}
    msg := json.marshal(json_body)
}

format_message(difference, format) = msg {
    not format == "json"
	msg := sprintf("extra actions found in workflows but not allowed in the profile: %v", [difference])
}
`
)

func TestConstraintsJSONOutput(t *testing.T) {
	t.Parallel()

	violationFormat := rego.OutputJSON.String()
	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type:            rego.ConstraintsEvaluationType.String(),
			ViolationFormat: &violationFormat,
			Def:             jsonPolicyDef,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	pol := map[string]any{
		"data": []string{"foo", "bar"},
	}

	evalResult, err := e.Eval(context.Background(), pol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data": []string{"foo", "bar", "baz"},
		},
	})
	require.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "should have failed the evaluation")

	// check that the error payload msg is JSON in the expected format
	errmsg := engerrors.ErrorAsEvalDetails(err)
	var errDetails []struct {
		ActionsNotAllowed []string `json:"actions_not_allowed"`
	}
	err = json.Unmarshal([]byte(errmsg), &errDetails)
	require.NoError(t, err, "could not unmarshal error JSON")
	assert.Len(t, errDetails, 1, "should have one result")
	assert.Contains(t, errDetails[0].ActionsNotAllowed, "baz", "should have baz in the result")

	// check that result is a list with a single element
	outputList, ok := evalResult.Output.([]any)
	require.True(t, ok, "evaluation result output should be a list, got %+T", evalResult.Output)

	assert.Len(t, outputList, 1, "evaluation result should have one element")

	assert.Contains(t, outputList[0], "actions_not_allowed", "evaluation result should contain actions_not_allowed key")
	assert.Contains(t, outputList[0].(map[string]any)["actions_not_allowed"], "baz", "evaluation result should contain baz")
}

func TestConstraintsJSONFalback(t *testing.T) {
	t.Parallel()

	violationFormat := rego.OutputJSON.String()
	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type:            rego.ConstraintsEvaluationType.String(),
			ViolationFormat: &violationFormat,
			Def: `
package minder

violations[{"msg": msg}] {
	input.profile.data != input.ingested.data
	msg := sprintf("data did not match profile: %s", [input.profile.data])
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	pol := map[string]any{
		"data": "foo",
	}

	evalResult, err := e.Eval(context.Background(), pol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data": "bar",
		},
	})
	require.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "should have failed the evaluation")

	// check that the error payload msg is JSON in the expected format
	errmsg := engerrors.ErrorAsEvalDetails(err)
	var errDetails []struct {
		Msg string `json:"msg"`
	}
	err = json.Unmarshal([]byte(errmsg), &errDetails)
	require.NoError(t, err, "could not unmarshal error JSON")
	assert.Len(t, errDetails, 1, "error details should have one element")

	// check that result is a list with a single element
	assert.Len(t, evalResult.Output, 1, "evaluation result should have one element")
	assert.Equal(t, []any{map[string]any{"msg": "data did not match profile: foo"}}, evalResult.Output)
}

func TestOutputTypePassedIntoRule(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def:  jsonPolicyDef,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	pol := map[string]any{
		"data": []string{"one", "two"},
	}

	res, err := e.Eval(context.Background(), pol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data": []string{"two", "three"},
		},
	})
	require.Error(t, err, "should have failed the evaluation")
	require.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "should have failed the evaluation")

	errmsg := engerrors.ErrorAsEvalDetails(err)
	assert.Contains(t, errmsg, "extra actions found in workflows but not allowed in the profile", "should have the expected error message")
	assert.Contains(t, errmsg, "three", "should have the expected content")
	assert.Equal(t, []any{`extra actions found in workflows but not allowed in the profile: ["three"]`}, res.Output)
}

func TestCantCreateEvaluatorWithInvalidConfig(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		_, err := rego.NewRegoEvaluator(nil)
		require.Error(t, err, "should have failed to create evaluator")
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		_, err := rego.NewRegoEvaluator(&minderv1.RuleType_Definition_Eval_Rego{})
		require.Error(t, err, "should have failed to create evaluator")
	})

	t.Run("invalid type", func(t *testing.T) {
		t.Parallel()

		_, err := rego.NewRegoEvaluator(
			&minderv1.RuleType_Definition_Eval_Rego{
				Type: "invalid",
			},
		)
		require.Error(t, err, "should have failed to create evaluator")
	})
}

// This test case reflects the scenario where the user provided
// a rego profile definition that has a syntax error.
func TestCantEvaluateWithInvalidProfile(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package minder

violations[{"msg": msg}] {`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	_, err = e.Eval(context.Background(), map[string]any{}, nil,
		&interfaces.Ingested{Object: map[string]any{}})
	assert.Error(t, err, "should have failed to evaluate")
}

func TestCantEvaluateWithCompilerError(t *testing.T) {
	t.Parallel()

	// This profile is using a variable that is restricted
	// in OPA's strict mode.
	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package minder

violations[{"msg": msg}] {
	input := 12345
	msg := "data did not contain foo"
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	_, err = e.Eval(context.Background(), map[string]any{}, nil,
		&interfaces.Ingested{Object: map[string]any{}})
	assert.Error(t, err, "should have failed to evaluate")
	assert.ErrorContains(t, err, "rego_compile_error: variables must not shadow input (use a different variable name)")
}

func TestCustomDatasourceRegister(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	fds := v1mockds.NewMockDataSource(ctrl)
	fdsf := v1mockds.NewMockDataSourceFuncDef(ctrl)

	fds.EXPECT().GetFuncs().Return(map[v1datasources.DataSourceFuncKey]v1datasources.DataSourceFuncDef{
		"source": fdsf,
	}).AnyTimes()

	fdsf.EXPECT().ValidateArgs(gomock.Any()).Return(nil).AnyTimes()

	fdsr := v1datasources.NewDataSourceRegistry()

	err := fdsr.RegisterDataSource("fake", fds)
	require.NoError(t, err, "could not register data source")

	e, err := rego.NewRegoEvaluator(
		&minderv1.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package minder

default allow = false

allow {
	minder.datasource.fake.source({"datasourcetest": input.ingested.data}) == "foo"
}`,
		},
		options.WithDataSources(fdsr),
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	// Matches
	fdsf.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any()).Return("foo", nil)
	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data": "foo",
		},
	})
	require.NoError(t, err, "could not evaluate")

	// Doesn't match
	fdsf.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any()).Return("bar", nil)
	_, err = e.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{
		Object: map[string]any{
			"data": "bar",
		},
	})
	require.ErrorIs(t, err, interfaces.ErrEvaluationFailed, "should have failed the evaluation")
}
