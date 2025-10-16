// Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package interfaces

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v63/github"
)

// Provider is a slice of the github.com/mindersec/minder/pkg/providers/v1.Provider
// interface which contains only the methods needed for engine evaluation. (currently none)
type Provider interface {
}

// GitProvider is a subset of the Provider interface that is used for git ingestion for rules.
type GitProvider interface {
	// Clone clones a git repository.  This provides a full git Repository
	// which can be used to create new commits, etc.
	Clone(ctx context.Context, url string, branch string) (*git.Repository, error)

	// FSAtRef returns the filesystem at the given ref for the git repository,
	// along with the resolved hash of the ref.
	//	FSAtRef(ctx context.Context, url string, ref string) (billy.Filesystem, plumbing.Hash, error)
}

// RESTProvider is a subset of the Provider interface used for REST API ingestion.
type RESTProvider interface {
	GetBaseURL() string
	NewRequest(method, url string, body any) (*http.Request, error)
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

// GitHubIssuePRClient is a subset of the Provider interface that is used for managing
// issue and PR comments (which are partially, but not fully interchangeable).
type GitHubIssuePRClient interface {
	ListReviews(ctx context.Context, owner, repo string, number int, opts *github.ListOptions) (
		[]*github.PullRequestReview, error)
	CreateReview(ctx context.Context, owner, repo string, number int, review *github.PullRequestReviewRequest) (
		*github.PullRequestReview, error)
	DismissReview(ctx context.Context, owner, repo string, number int, reviewID int64,
		req *github.PullRequestReviewDismissalRequest) (
		*github.PullRequestReview, error)
	SetCommitStatus(ctx context.Context, owner, repo string, sha string, status *github.RepoStatus) (*github.RepoStatus, error)
	ListIssueComments(ctx context.Context, owner, repo string, number int, opts *github.IssueListCommentsOptions) (
		[]*github.IssueComment, error)
	CreateIssueComment(ctx context.Context, owner, repo string, number int, comment string) (*github.IssueComment, error)
	UpdateIssueComment(ctx context.Context, owner, repo string, id int64, comment string) error
}

// SelfAwareness is needed in the PAT token authentication flow to switch between comments
// and pull request reviews, since you can't review your own pull requests.
type SelfAwareness interface {
	// GetUserId returns the ID of the authenticated user.
	GetUserId(ctx context.Context) (int64, error)
}

// GitHubListAndClone is an interface that defines the methods needed to list files
// in a GitHub pull request
type GitHubListAndClone interface {
	ListFiles(ctx context.Context, owner, repo string, prNumber int, perPage, page int) (
		[]*github.CommitFile, *github.Response, error)
	Clone(ctx context.Context, repoURL, ref string) (*git.Repository, error)
}

// As is a type-cast function for Providers
func As[T any](provider Provider) (T, error) {
	result, ok := provider.(T)
	if !ok {
		return result, errors.New("provider type cast failed")
	}
	return result, nil
}
