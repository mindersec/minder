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
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-github/v63/github"

	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// V1 is the version of the providers interface
const (
	V1 = "v1"
)

// Provider is the general interface for all providers
type Provider interface {
	// CanImplement returns true/false depending on whether the Provider
	// can implement the specified trait
	CanImplement(trait minderv1.ProviderType) bool

	// FetchAllProperties fetches all properties for the given entity
	FetchAllProperties(
		ctx context.Context, getByProps *properties.Properties, entType minderv1.Entity) (*properties.Properties, error)
	// FetchProperty fetches a single property for the given entity
	FetchProperty(
		ctx context.Context, getByProps *properties.Properties, entType minderv1.Entity, key string) (*properties.Property, error)
	// GetEntityName forms an entity name from the given properties
	// The name is used to identify the entity within minder and is how
	// it will be stored in the database.
	GetEntityName(entType minderv1.Entity, props *properties.Properties) (string, error)
}

// Git is the interface for git providers
type Git interface {
	Provider

	// Clone clones a git repository
	Clone(ctx context.Context, url string, branch string) (*git.Repository, error)
}

// REST is the trait interface for interacting with an REST API.
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

	ListAllRepositories(context.Context) ([]*minderv1.Repository, error)
}

var (
	// ArtifactTypeContainerRetentionPeriod represents the retention period for container artifacts
	ArtifactTypeContainerRetentionPeriod = time.Now().AddDate(0, -6, 0)
)

// GetArtifactVersionsFilter is the options to filter GetArtifactVersions
type GetArtifactVersionsFilter interface {
	// IsSkippable determines if an artifact should be skipped
	IsSkippable(createdAt time.Time, tags []string) error
}

// ArtifactProvider is the interface for artifact providers. This will
// contain methods for interacting with artifacts.
type ArtifactProvider interface {
	// GetArtifactVersions returns the versions of the given artifact.
	GetArtifactVersions(ctx context.Context, artifact *minderv1.Artifact,
		filter GetArtifactVersionsFilter) ([]*minderv1.ArtifactVersion, error)
}

// GitHub is the interface for interacting with the GitHub REST API
// Add methods here for interacting with the GitHub Rest API
type GitHub interface {
	Provider
	RepoLister
	REST
	Git
	ImageLister
	ArtifactProvider

	GetCredential() GitHubCredential
	GetRepository(context.Context, string, string) (*github.Repository, error)
	GetBranchProtection(context.Context, string, string, string) (*github.Protection, error)
	UpdateBranchProtection(context.Context, string, string, string, *github.ProtectionRequest) error
	ListPackagesByRepository(context.Context, string, string, int64, int, int) ([]*github.Package, error)
	GetPackageByName(context.Context, string, string, string) (*github.Package, error)
	GetPackageVersionById(context.Context, string, string, string, int64) (*github.PackageVersion, error)
	GetPullRequest(context.Context, string, string, int) (*github.PullRequest, error)
	CreateReview(context.Context, string, string, int, *github.PullRequestReviewRequest) (*github.PullRequestReview, error)
	UpdateReview(context.Context, string, string, int, int64, string) (*github.PullRequestReview, error)
	ListReviews(context.Context, string, string, int, *github.ListOptions) ([]*github.PullRequestReview, error)
	DismissReview(context.Context, string, string, int, int64,
		*github.PullRequestReviewDismissalRequest) (*github.PullRequestReview, error)
	SetCommitStatus(context.Context, string, string, string, *github.RepoStatus) (*github.RepoStatus, error)
	ListFiles(ctx context.Context, owner string, repo string, prNumber int,
		perPage int, pageNumber int) ([]*github.CommitFile, *github.Response, error)
	IsOrg() bool
	ListHooks(ctx context.Context, owner, repo string) ([]*github.Hook, error)
	DeleteHook(ctx context.Context, owner, repo string, id int64) (*github.Response, error)
	EditHook(ctx context.Context, owner, repo string, id int64, hook *github.Hook) (*github.Hook, error)
	CreateHook(ctx context.Context, owner, repo string, hook *github.Hook) (*github.Hook, error)
	CreateSecurityAdvisory(ctx context.Context, owner, repo, severity, summary, description string,
		v []*github.AdvisoryVulnerability) (string, error)
	CloseSecurityAdvisory(ctx context.Context, owner, repo, id string) error
	CreatePullRequest(ctx context.Context, owner, repo, title, body, head, base string) (*github.PullRequest, error)
	ClosePullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error)
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
	StartCheckRun(context.Context, string, string, *github.CreateCheckRunOptions) (*github.CheckRun, error)
	UpdateCheckRun(context.Context, string, string, int64, *github.UpdateCheckRunOptions) (*github.CheckRun, error)
}

// ImageLister is the interface for listing images
type ImageLister interface {
	Provider

	// ListImages lists the images available for the provider
	ListImages(ctx context.Context) ([]string, error)

	// GetNamespaceURL returns the repository URL
	GetNamespaceURL() string
}

// OCI is the interface for interacting with OCI registries
type OCI interface {
	Provider
	ArtifactProvider

	// ListTags lists the tags available for the given container in the given namespace
	// for the OCI provider.
	ListTags(ctx context.Context, name string) ([]string, error)

	// GetDigest returns the digest for the given tag of the given container in the given namespace
	// for the OCI provider.
	GetDigest(ctx context.Context, name, tag string) (string, error)

	// GetReferrer returns the referrer for the given tag of the given container in the given namespace
	// for the OCI provider. It returns the referrer as a golang struct given the OCI spec.
	// TODO - Define the referrer struct
	GetReferrer(ctx context.Context, name, tag, artifactType string) (any, error)

	// GetManifest returns the manifest for the given tag of the given container in the given namespace
	// for the OCI provider. It returns the manifest as a golang struct given the OCI spec.
	// TODO - Define the manifest struct
	GetManifest(ctx context.Context, name, tag string) (*v1.Manifest, error)

	// GetRegistry returns the registry name
	GetRegistry() string

	// GetAuthenticator returns the authenticator for the OCI provider
	GetAuthenticator() (authn.Authenticator, error)
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

// As is a type-cast function for Providers
func As[T Provider](provider Provider) (T, error) {
	result, ok := provider.(T)
	if !ok {
		return result, errors.New("provider type cast failed")
	}
	return result, nil
}
