// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"net/http"

	"github.com/google/go-github/v63/github"

	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

// Ensure that TestKit implements the GitHub interfaces used by the diff
// ingester and the vulncheck evaluator.
var (
	_ interfaces.GitHubListAndClone  = &TestKit{}
	_ interfaces.GitHubIssuePRClient = &TestKit{}
	_ interfaces.SelfAwareness       = &TestKit{}
)

// githubClient returns a GitHub client whose requests are served by the
// TestKit's HTTP handler, so tests can mock GitHub API responses the same way
// as any other REST call.
func (tk *TestKit) githubClient() *github.Client {
	return github.NewClient(&http.Client{Transport: tk})
}

// ListFiles lists the files changed in a pull request
func (tk *TestKit) ListFiles(
	ctx context.Context, owner, repo string, prNumber int, perPage, page int,
) ([]*github.CommitFile, *github.Response, error) {
	return tk.githubClient().PullRequests.ListFiles(ctx, owner, repo, prNumber,
		&github.ListOptions{PerPage: perPage, Page: page})
}

// ListReviews lists the reviews on a pull request
func (tk *TestKit) ListReviews(
	ctx context.Context, owner, repo string, number int, opts *github.ListOptions,
) ([]*github.PullRequestReview, error) {
	reviews, _, err := tk.githubClient().PullRequests.ListReviews(ctx, owner, repo, number, opts)
	return reviews, err
}

// CreateReview creates a review on a pull request
func (tk *TestKit) CreateReview(
	ctx context.Context, owner, repo string, number int, review *github.PullRequestReviewRequest,
) (*github.PullRequestReview, error) {
	res, _, err := tk.githubClient().PullRequests.CreateReview(ctx, owner, repo, number, review)
	return res, err
}

// DismissReview dismisses a review on a pull request
func (tk *TestKit) DismissReview(
	ctx context.Context, owner, repo string, number int, reviewID int64,
	req *github.PullRequestReviewDismissalRequest,
) (*github.PullRequestReview, error) {
	res, _, err := tk.githubClient().PullRequests.DismissReview(ctx, owner, repo, number, reviewID, req)
	return res, err
}

// SetCommitStatus sets the status of a commit
func (tk *TestKit) SetCommitStatus(
	ctx context.Context, owner, repo string, sha string, status *github.RepoStatus,
) (*github.RepoStatus, error) {
	res, _, err := tk.githubClient().Repositories.CreateStatus(ctx, owner, repo, sha, status)
	return res, err
}

// ListIssueComments lists the comments on an issue or pull request
func (tk *TestKit) ListIssueComments(
	ctx context.Context, owner, repo string, number int, opts *github.IssueListCommentsOptions,
) ([]*github.IssueComment, error) {
	comments, _, err := tk.githubClient().Issues.ListComments(ctx, owner, repo, number, opts)
	return comments, err
}

// CreateIssueComment creates a comment on an issue or pull request
func (tk *TestKit) CreateIssueComment(
	ctx context.Context, owner, repo string, number int, comment string,
) (*github.IssueComment, error) {
	res, _, err := tk.githubClient().Issues.CreateComment(ctx, owner, repo, number,
		&github.IssueComment{Body: &comment})
	return res, err
}

// UpdateIssueComment updates a comment on an issue or pull request
func (tk *TestKit) UpdateIssueComment(ctx context.Context, owner, repo string, id int64, comment string) error {
	_, _, err := tk.githubClient().Issues.EditComment(ctx, owner, repo, id,
		&github.IssueComment{Body: &comment})
	return err
}

// GetUserId returns the ID of the authenticated user
func (tk *TestKit) GetUserId(ctx context.Context) (int64, error) {
	user, _, err := tk.githubClient().Users.Get(ctx, "")
	return user.GetID(), err
}
