// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package engine

import "regexp"

var conventionalCommitPattern = regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([A-Za-z0-9._/-]+\))?!?: .+`)

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

// CommitPolicy defines a single commit-level policy.
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

// ConventionalCommitPolicy is an example commit policy that validates commit
// messages against a Conventional Commits-like format.
type ConventionalCommitPolicy struct{}

// Name returns the policy name.
func (ConventionalCommitPolicy) Name() string {
	return "conventional_commit"
}

// Evaluate validates commit message format.
func (p ConventionalCommitPolicy) Evaluate(commit Commit) CommitEvaluation {
	if conventionalCommitPattern.MatchString(commit.Message) {
		return CommitEvaluation{
			Policy: p.Name(),
			Passed: true,
			Reason: "commit message matches conventional commit format",
		}
	}

	return CommitEvaluation{
		Policy: p.Name(),
		Passed: false,
		Reason: "commit message must follow <type>(optional-scope): <description>",
	}
}
