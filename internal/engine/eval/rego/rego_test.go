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

package rego_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	engerrors "github.com/stacklok/mediator/internal/engine/errors"
	"github.com/stacklok/mediator/internal/engine/eval/rego"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// Evaluates a simple query against a simple policy
// In this case, the policy is a simple "allow" rule.
// The given policy map is empty since all the policy
// needed in ths test case is contained in the rego
// definition.
func TestEvaluatorDenyByDefaultEvalSimple(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package mediator

default allow = false

allow {
	input.ingested.data == "foo"
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	// Matches
	err = e.Eval(context.Background(), emptyPol, map[string]any{
		"data": "foo",
	})
	require.NoError(t, err, "could not evaluate")

	// Doesn't match
	err = e.Eval(context.Background(), emptyPol, map[string]any{
		"data": "bar",
	})
	require.ErrorIs(t, err, engerrors.ErrEvaluationFailed, "should have failed the evaluation")
}

func TestEvaluatorDenyByConstraintsEvalSimple(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package mediator

violations[{"msg": msg}] {
	input.ingested.data != "foo"
	msg := "data did not contain foo"
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	emptyPol := map[string]any{}

	// Matches
	err = e.Eval(context.Background(), emptyPol, map[string]any{
		"data": "foo",
	})
	require.NoError(t, err, "could not evaluate")

	// Doesn't match
	err = e.Eval(context.Background(), emptyPol, map[string]any{
		"data": "bar",
	})
	require.ErrorIs(t, err, engerrors.ErrEvaluationFailed, "should have failed the evaluation")
	require.ErrorContains(t, err, "data did not contain foo", "should have failed the evaluation")
}

// Evaluates a simple query against a simple policy
// In this case, the policy is a simple "allow" rule.
// The given policy map has a value for the "data" key
// which is used in the rego definition. The ingested
// data has to match the policy data.
func TestDenyByDefaultEvaluationWithPolicy(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.DenyByDefaultEvaluationType.String(),
			Def: `
package mediator

default allow = false

allow {
	input.policy.data == input.ingested.data
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	pol := map[string]any{
		"data": "foo",
	}

	// Matches
	err = e.Eval(context.Background(), pol, map[string]any{
		"data": "foo",
	})
	require.NoError(t, err, "could not evaluate")

	// Doesn't match
	err = e.Eval(context.Background(), pol, map[string]any{
		"data": "bar",
	})
	require.ErrorIs(t, err, engerrors.ErrEvaluationFailed, "should have failed the evaluation")
}

func TestConstrainedEvaluationWithPolicy(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package mediator

violations[{"msg": msg}] {
	input.policy.data != input.ingested.data
	msg := sprintf("data did not match policy: %s", [input.policy.data])
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	pol := map[string]any{
		"data": "foo",
	}

	// Matches
	err = e.Eval(context.Background(), pol, map[string]any{
		"data": "foo",
	})
	require.NoError(t, err, "could not evaluate")

	// Doesn't match
	err = e.Eval(context.Background(), pol, map[string]any{
		"data": "bar",
	})
	require.ErrorIs(t, err, engerrors.ErrEvaluationFailed, "should have failed the evaluation")
	assert.ErrorContains(t, err, "data did not match policy: foo", "should have failed the evaluation")
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

		_, err := rego.NewRegoEvaluator(&pb.RuleType_Definition_Eval_Rego{})
		require.Error(t, err, "should have failed to create evaluator")
	})

	t.Run("invalid type", func(t *testing.T) {
		t.Parallel()

		_, err := rego.NewRegoEvaluator(
			&pb.RuleType_Definition_Eval_Rego{
				Type: "invalid",
			},
		)
		require.Error(t, err, "should have failed to create evaluator")
	})
}

// This test case reflects the scenario where the user provided
// a rego policy definition that has a syntax error.
func TestCantEvaluateWithInvalidPolicy(t *testing.T) {
	t.Parallel()

	e, err := rego.NewRegoEvaluator(
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package mediator

violations[{"msg": msg}] {`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	err = e.Eval(context.Background(), map[string]any{}, map[string]any{})
	assert.Error(t, err, "should have failed to evaluate")
}

func TestCantEvaluateWithCompilerError(t *testing.T) {
	t.Parallel()

	// This policy is using a variable that is restricted
	// in OPA's strict mode.
	e, err := rego.NewRegoEvaluator(
		&pb.RuleType_Definition_Eval_Rego{
			Type: rego.ConstraintsEvaluationType.String(),
			Def: `
package mediator

violations[{"msg": msg}] {
	input := 12345
	msg := "data did not contain foo"
}`,
		},
	)
	require.NoError(t, err, "could not create evaluator")

	err = e.Eval(context.Background(), map[string]any{}, 12345)
	assert.Error(t, err, "should have failed to evaluate")
}
