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

package rego_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	engerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/eval/rego"
	engif "github.com/stacklok/minder/internal/engine/interfaces"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

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
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	// Matches
	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: map[string]any{
			"data": "foo",
		},
	})
	require.NoError(t, err, "could not evaluate")

	// Doesn't match
	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: map[string]any{
			"data": "bar",
		},
	})
	require.ErrorIs(t, err, engerrors.ErrEvaluationFailed, "should have failed the evaluation")
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
	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: map[string]any{
			"data": "bar",
		},
	})
	require.ErrorIs(t, err, engerrors.ErrEvaluationSkipped, "should have been skipped")
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
	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: map[string]any{
			"data": "foo",
		},
	})
	require.NoError(t, err, "could not evaluate")

	// Doesn't match
	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: map[string]any{
			"data": "bar",
		},
	})
	require.ErrorIs(t, err, engerrors.ErrEvaluationFailed, "should have failed the evaluation")
	require.ErrorContains(t, err, "data did not contain foo", "should have failed the evaluation")
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

	err = e.Eval(context.Background(), emptyPol, &engif.Result{
		Object: map[string]any{
			"data":  "foo",
			"datum": "bar",
		},
	})
	require.ErrorIs(t, err, engerrors.ErrEvaluationFailed, "should have failed the evaluation")
	require.ErrorContains(t, err, "- evaluation failure: data should not contain foo\n")
	require.ErrorContains(t, err, "- evaluation failure: datum should not contain bar")
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
	err = e.Eval(context.Background(), pol, &engif.Result{
		Object: map[string]any{
			"data": "foo",
		},
	})
	require.NoError(t, err, "could not evaluate")

	// Doesn't match
	err = e.Eval(context.Background(), pol, &engif.Result{
		Object: map[string]any{
			"data": "bar",
		},
	})
	require.ErrorIs(t, err, engerrors.ErrEvaluationFailed, "should have failed the evaluation")
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
	err = e.Eval(context.Background(), pol, &engif.Result{
		Object: map[string]any{
			"data": "foo",
		},
	})
	require.NoError(t, err, "could not evaluate")

	// Doesn't match
	err = e.Eval(context.Background(), pol, &engif.Result{
		Object: map[string]any{
			"data": "bar",
		},
	})
	require.ErrorIs(t, err, engerrors.ErrEvaluationFailed, "should have failed the evaluation")
	assert.ErrorContains(t, err, "data did not match profile: foo", "should have failed the evaluation")
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

	err = e.Eval(context.Background(), map[string]any{},
		&engif.Result{Object: map[string]any{}})
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

	err = e.Eval(context.Background(), map[string]any{},
		&engif.Result{Object: map[string]any{}})
	assert.Error(t, err, "should have failed to evaluate")
}
