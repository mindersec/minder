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
