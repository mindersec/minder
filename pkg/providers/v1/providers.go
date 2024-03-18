// Copyright 2023 Stacklok, Inc
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

// Package v1 for providers provides the public interfaces for the providers
// implemented by minder. The providers are the sources of the data
// that is used by the rules.
package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-git/go-git/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/go-github/v56/github"

	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// V1 is the version of the providers interface
const (
	V1 = "v1"
)

// Provider is the general interface for all providers
type Provider interface {
}

// Git is the interface for git providers
type Git interface {
	Provider

	// Clone clones a git repository
	Clone(ctx context.Context, url string, branch string) (*git.Repository, error)
}

// REST is the interface for interacting with an REST API.
type REST interface {
	Provider

	// GetBaseURL returns the base URL for the REST API.
	GetBaseURL() string

	// NewRequest creates an HTTP request.
	NewRequest(method, url string, body any) (*http.Request, error)

	// Do executes an HTTP request.
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

// RepoLister is the interface for listing repositories
type RepoLister interface {
	Provider

	ListUserRepositories(context.Context, string) ([]*minderv1.Repository, error)
	ListOrganizationRepsitories(context.Context, string) ([]*minderv1.Repository, error)
}

// GitHub is the interface for interacting with the GitHub REST API
// Add methods here for interacting with the GitHub Rest API
type GitHub interface {
	Provider
	RepoLister
	REST
	Git

	GetCredential() GitHubCredential
	GetRepository(context.Context, string, string) (*github.Repository, error)
	ListAllRepositories(context.Context, bool, string) ([]*github.Repository, error)
	GetBranchProtection(context.Context, string, string, string) (*github.Protection, error)
	UpdateBranchProtection(context.Context, string, string, string, *github.ProtectionRequest) error
	ListPackagesByRepository(context.Context, bool, string, string, int64, int, int) ([]*github.Package, error)
	GetPackageByName(context.Context, bool, string, string, string) (*github.Package, error)
	GetPackageVersions(context.Context, bool, string, string, string) ([]*github.PackageVersion, error)
	GetPackageVersionByTag(context.Context, bool, string, string, string, string) (*github.PackageVersion, error)
	GetPackageVersionById(context.Context, bool, string, string, string, int64) (*github.PackageVersion, error)
	GetPullRequest(context.Context, string, string, int) (*github.PullRequest, error)
	CreateReview(context.Context, string, string, int, *github.PullRequestReviewRequest) (*github.PullRequestReview, error)
	UpdateReview(context.Context, string, string, int, int64, string) (*github.PullRequestReview, error)
	ListReviews(context.Context, string, string, int, *github.ListOptions) ([]*github.PullRequestReview, error)
	DismissReview(context.Context, string, string, int, int64,
		*github.PullRequestReviewDismissalRequest) (*github.PullRequestReview, error)
	SetCommitStatus(context.Context, string, string, string, *github.RepoStatus) (*github.RepoStatus, error)
	ListFiles(ctx context.Context, owner string, repo string, prNumber int,
		perPage int, pageNumber int) ([]*github.CommitFile, *github.Response, error)
	GetOwner() string
	ListHooks(ctx context.Context, owner, repo string) ([]*github.Hook, error)
	DeleteHook(ctx context.Context, owner, repo string, id int64) (*github.Response, error)
	CreateHook(ctx context.Context, owner, repo string, hook *github.Hook) (*github.Hook, error)
	CreateSecurityAdvisory(ctx context.Context, owner, repo, severity, summary, description string,
		v []*github.AdvisoryVulnerability) (string, error)
	CloseSecurityAdvisory(ctx context.Context, owner, repo, id string) error
	CreatePullRequest(ctx context.Context, owner, repo, title, body, head, base string) (*github.PullRequest, error)
	ListPullRequests(ctx context.Context, owner, repo string, opt *github.PullRequestListOptions) ([]*github.PullRequest, error)
	GetUserId(ctx context.Context) (int64, error)
	GetName(ctx context.Context) (string, error)
	GetLogin(ctx context.Context) (string, error)
	GetPrimaryEmail(ctx context.Context) (string, error)
	CreateIssueComment(ctx context.Context, owner, repo string, number int, comment string) (*github.IssueComment, error)
	ListIssueComments(ctx context.Context, owner, repo string, number int,
		opts *github.IssueListCommentsOptions,
	) ([]*github.IssueComment, error)
	UpdateIssueComment(ctx context.Context, owner, repo string, number int64, comment string) error
	AddAuthToPushOptions(ctx context.Context, options *git.PushOptions) error
}

// ParseAndValidate parses the given provider configuration and validates it.
func ParseAndValidate(rawConfig json.RawMessage, to any) error {
	if err := json.Unmarshal(rawConfig, to); err != nil {
		return fmt.Errorf("error parsing v1 provider config: %w", err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(to); err != nil {
		return fmt.Errorf("error validating v1 provider config: %w", err)
	}

	return nil
}
