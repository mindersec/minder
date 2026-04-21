// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type alwaysPassPolicy struct{}
type alwaysFailPolicy struct{}

func (alwaysPassPolicy) Name() string {
	return "always_pass"
}

func (p alwaysPassPolicy) Evaluate(_ Commit) CommitEvaluation {
	return CommitEvaluation{
		Policy: p.Name(),
		Passed: true,
		Reason: "policy passed",
	}
}

func (alwaysFailPolicy) Name() string {
	return "always_fail"
}

func (p alwaysFailPolicy) Evaluate(_ Commit) CommitEvaluation {
	return CommitEvaluation{
		Policy: p.Name(),
		Passed: false,
		Reason: "policy failed",
	}
}

func TestCommitEvaluatorEvaluateWithoutPolicies(t *testing.T) {
	t.Parallel()

	evaluator := NewCommitEvaluator()

	results := evaluator.Evaluate(Commit{Message: "feat: add evaluator"})
	require.NotNil(t, results)
	require.Empty(t, results)
}

func TestCommitEvaluatorEvaluateWithPolicy(t *testing.T) {
	t.Parallel()

	evaluator := NewCommitEvaluator(alwaysPassPolicy{})

	results := evaluator.Evaluate(Commit{Message: "anything"})
	require.Len(t, results, 1)
	require.Equal(t, "always_pass", results[0].Policy)
	require.True(t, results[0].Passed)
	require.Equal(t, "policy passed", results[0].Reason)
}

func TestConventionalCommitPolicy(t *testing.T) {
	t.Parallel()

	policy := ConventionalCommitPolicy{}

	t.Run("passes on valid message", func(t *testing.T) {
		t.Parallel()

		result := policy.Evaluate(Commit{Message: "feat(engine): add commit evaluator"})
		require.True(t, result.Passed)
		require.Equal(t, "conventional_commit", result.Policy)
	})

	t.Run("passes with slash scope", func(t *testing.T) {
		t.Parallel()

		result := policy.Evaluate(Commit{Message: "feat(api/v1): add endpoint"})
		require.True(t, result.Passed)
		require.Equal(t, "conventional_commit", result.Policy)
	})

	t.Run("passes with uppercase scope", func(t *testing.T) {
		t.Parallel()

		result := policy.Evaluate(Commit{Message: "fix(API): handle timeout"})
		require.True(t, result.Passed)
		require.Equal(t, "conventional_commit", result.Policy)
	})

	t.Run("fails on invalid message", func(t *testing.T) {
		t.Parallel()

		result := policy.Evaluate(Commit{Message: "Add commit evaluator"})
		require.False(t, result.Passed)
		require.Equal(t, "conventional_commit", result.Policy)
		require.Contains(t, result.Reason, "commit message")
	})
}

func TestCommitEvaluatorEvaluateAll(t *testing.T) {
	t.Parallel()

	evaluator := NewCommitEvaluator(alwaysPassPolicy{})

	results := evaluator.EvaluateAll([]Commit{
		{SHA: "sha1", Message: "feat: one"},
		{SHA: "sha2", Message: "fix: two"},
	})

	require.Len(t, results, 2)
	require.Equal(t, "sha1", results[0].SHA)
	require.Equal(t, "sha2", results[1].SHA)
	require.Len(t, results[0].Evaluations, 1)
	require.True(t, results[0].Evaluations[0].Passed)
}

func TestEvaluateAllEmpty(t *testing.T) {
	t.Parallel()

	evaluator := NewCommitEvaluator(alwaysPassPolicy{})
	results := evaluator.EvaluateAll(nil)
	require.Empty(t, results)
}

func TestHasPolicyFailures(t *testing.T) {
	t.Parallel()

	allPass := []CommitResult{
		{SHA: "sha1", Evaluations: []CommitEvaluation{{Policy: "p1", Passed: true}}},
		{SHA: "sha2", Evaluations: []CommitEvaluation{{Policy: "p1", Passed: true}}},
	}
	require.False(t, HasPolicyFailures(allPass))
	require.False(t, HasFailures(allPass))

	hasFail := []CommitResult{
		{SHA: "sha1", Evaluations: []CommitEvaluation{{Policy: "p1", Passed: true}}},
		{SHA: "sha2", Evaluations: []CommitEvaluation{{Policy: "p1", Passed: false}}},
	}
	require.True(t, HasPolicyFailures(hasFail))
	require.True(t, HasFailures(hasFail))

	evaluator := NewCommitEvaluator(alwaysPassPolicy{}, alwaysFailPolicy{})
	require.True(t, HasPolicyFailures(evaluator.EvaluateAll([]Commit{{SHA: "sha3", Message: "any"}})))
}
