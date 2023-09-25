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
// implemented by mediator. The providers are the sources of the data
// that is used by the rules.
package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-git/go-git/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/go-github/v53/github"
)

// V1 is the version of the providers interface
const (
	V1 = "v1"
)

// Provider is the general interface for all providers
type Provider interface {
	// GetToken returns the token for the provider
	GetToken() string
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

	// NewRequest creates an HTTP request.
	NewRequest(method, url string, body io.Reader) (*http.Request, error)

	// Do executes an HTTP request.
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
}

// RESTConfig is the struct that contains the configuration for the HTTP client
type RESTConfig struct {
	BaseURL string `json:"base_url" yaml:"base_url" mapstructure:"base_url" validate:"required"`
}

// GitHub is the interface for interacting with the GitHub REST API
// Add methods here for interacting with the GitHub Rest API
type GitHub interface {
	Provider
	REST

	GetAuthenticatedUser(context.Context) (*github.User, error)
	GetRepository(context.Context, string, string) (*github.Repository, error)
	ListAllRepositories(context.Context, bool, string) ([]*github.Repository, error)
	GetBranchProtection(context.Context, string, string, string) (*github.Protection, error)
	ListAllPackages(context.Context, bool, string, string, int, int) ([]*github.Package, error)
	ListPackagesByRepository(context.Context, bool, string, string, int64, int, int) ([]*github.Package, error)
	GetPackageByName(context.Context, bool, string, string, string) (*github.Package, error)
	GetPackageVersions(context.Context, bool, string, string, string) ([]*github.PackageVersion, error)
	GetPackageVersionByTag(context.Context, bool, string, string, string, string) (*github.PackageVersion, error)
	GetPackageVersionById(context.Context, bool, string, string, string, int64) (*github.PackageVersion, error)
	GetPullRequest(context.Context, string, string, int) (*github.PullRequest, error)
	CreateReview(context.Context, string, string, int, *github.PullRequestReviewRequest) (*github.PullRequestReview, error)
	ListReviews(context.Context, string, string, int, *github.ListOptions) ([]*github.PullRequestReview, error)
	DismissReview(context.Context, string, string, int, int64,
		*github.PullRequestReviewDismissalRequest) (*github.PullRequestReview, error)
	SetCommitStatus(context.Context, string, string, string, *github.RepoStatus) (*github.RepoStatus, error)
	ListFiles(ctx context.Context, owner string, repo string, prNumber int,
		perPage int, pageNumber int) ([]*github.CommitFile, *github.Response, error)
	GetOwner() string
}

// GitHubConfig is the struct that contains the configuration for the GitHub client
// Endpoint: is the GitHub API endpoint
// If using the public GitHub API, Endpoint can be left blank
// disable revive linting for this struct as there is nothing wrong with the
// naming convention
type GitHubConfig struct {
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`
}

// ParseAndValidate parses the given provider configuration and validates it.
func ParseAndValidate(rawConfig json.RawMessage, to any) error {
	if err := json.Unmarshal(rawConfig, to); err != nil {
		return fmt.Errorf("error parsing http v1 provider config: %w", err)
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(to); err != nil {
		return fmt.Errorf("error validating http v1 provider config: %w", err)
	}

	return nil
}
