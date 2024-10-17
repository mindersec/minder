// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package pr_actions contains shared code to take on PRs
package pr_actions

// Action specifies what action to take on the PR
type Action string

const (
	// ActionReviewPr does a code review
	ActionReviewPr Action = "review"
	// ActionComment comments on the PR
	ActionComment Action = "comment"
	// ActionCommitStatus sets the commit status on the PR
	ActionCommitStatus Action = "commit_status"
	// ActionProfileOnly only evaluates the PR and return the eval status. Useful for testing.
	ActionProfileOnly Action = "profile_only"
	// ActionSummary puts a summary of the findings into the PR
	ActionSummary Action = "summary"
)
