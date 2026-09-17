// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package engine

// Commit contains commit-level data that can be evaluated by policies.
type Commit struct {
	SHA     string
	Message string
}

// CommitEvaluation captures the result of evaluating one policy.
type CommitEvaluation struct {
	Policy string
	Passed bool
	Reason string
}

// CommitResult captures all evaluations for a single commit.
type CommitResult struct {
	SHA         string
	Evaluations []CommitEvaluation
}

// CommitPolicy defines the minimal contract for demo-oriented commit-level policies.
// Concrete implementations should live outside internal/engine.
type CommitPolicy interface {
	Name() string
	Evaluate(commit Commit) CommitEvaluation
}

// CommitEvaluator evaluates commits against a list of commit-level policies.
type CommitEvaluator struct {
	policies []CommitPolicy
}

// NewCommitEvaluator creates a commit evaluator with the provided policies.
func NewCommitEvaluator(policies ...CommitPolicy) *CommitEvaluator {
	return &CommitEvaluator{policies: policies}
}

// Evaluate evaluates the commit against all configured policies.
func (e *CommitEvaluator) Evaluate(commit Commit) []CommitEvaluation {
	if len(e.policies) == 0 {
		return []CommitEvaluation{}
	}

	results := make([]CommitEvaluation, 0, len(e.policies))
	for _, policy := range e.policies {
		res := policy.Evaluate(commit)
		if res.Policy == "" {
			res.Policy = policy.Name()
		}
		results = append(results, res)
	}

	return results
}

// EvaluateAll evaluates all commits in a PR, preserving commit order.
func (e *CommitEvaluator) EvaluateAll(commits []Commit) []CommitResult {
	results := make([]CommitResult, 0, len(commits))

	for _, commit := range commits {
		results = append(results, CommitResult{
			SHA:         commit.SHA,
			Evaluations: e.Evaluate(commit),
		})
	}

	return results
}

// HasPolicyFailures returns true if any policy evaluation fails.
func HasPolicyFailures(results []CommitResult) bool {
	for _, commitResult := range results {
		for _, eval := range commitResult.Evaluations {
			if !eval.Passed {
				return true
			}
		}
	}

	return false
}

// HasFailures returns true if any policy evaluation fails.
// Deprecated: use HasPolicyFailures for clearer semantics.
func HasFailures(results []CommitResult) bool {
	return HasPolicyFailures(results)
}
