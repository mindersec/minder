// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/entities/properties"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// V1 is the version of the providers interface
const (
	V1 = "v1"
)

// ErrEntityNotFound is the error returned when an entity is not found
var ErrEntityNotFound = errors.New("entity not found")

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// Provider is the general interface for all providers
type Provider interface {
	// CanImplement returns true/false depending on whether the Provider
	// can implement the specified trait
	CanImplement(trait minderv1.ProviderType) bool

	// FetchAllProperties fetches all properties for the given entity
	FetchAllProperties(
		ctx context.Context, getByProps *properties.Properties, entType minderv1.Entity, cachedProps *properties.Properties,
	) (*properties.Properties, error)
	// FetchProperty fetches a single property for the given entity
	FetchProperty(
		ctx context.Context, getByProps *properties.Properties, entType minderv1.Entity, key string) (*properties.Property, error)
	// GetEntityName forms an entity name from the given properties
	// The name is used to identify the entity within minder and is how
	// it will be stored in the database.
	GetEntityName(entType minderv1.Entity, props *properties.Properties) (string, error)

	// SupportsEntity returns true if the provider supports the given entity type
	SupportsEntity(entType minderv1.Entity) bool

	// RegisterEntity ensures that the service provider has the necessary information
	// to know that the entity is handled by Minder. This could be creating a webhook
	// for a particular repository or artifact.
	// Note that the provider might choose to update the properties of the entity
	// adding the information about the registration. e.g. The webhook ID and URL.
	RegisterEntity(ctx context.Context, entType minderv1.Entity, props *properties.Properties) (*properties.Properties, error)

	// DeregisterEntity rolls back the registration of the entity. This could be deleting
	// a webhook for a particular repository or artifact. Note that this assumes a pre-registered
	// entity and thus requires the entity to have been registered before. Therefore, you should
	// either call this after RegisterEntity or after a FetchAllProperties call on an already
	// registered entity.
	//
	// When implementing, try to make this idempotent. That is, if the entity is already deregistered,
	// (e.g. a webhook is already deleted), then this should not return an error.
	DeregisterEntity(ctx context.Context, entType minderv1.Entity, props *properties.Properties) error

	// ReregisterEntity runs the necessary updates to the entity registration. This could be
	// updating the webhook URL or secret for a particular repository or artifact. This is useful
	// for secret rotation.
	ReregisterEntity(ctx context.Context, entType minderv1.Entity, props *properties.Properties) error

	// PropertiesToProtoMessage is the interface for converting properties to a proto message
	// this is temporary until we can get rid of the typed proto messages in EntityInfoWrapper
	// and the engine. That's also why we just didn't add the method to the generic Provider
	// interface.
	PropertiesToProtoMessage(entType minderv1.Entity, props *properties.Properties) (protoreflect.ProtoMessage, error)
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
	DeleteHook(ctx context.Context, owner, repo string, id int64) error
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

// PullRequestCommentType is the type of the pull request comment
type PullRequestCommentType string

const (
	// PullRequestCommentTypeApprove is the type for an approval
	PullRequestCommentTypeApprove PullRequestCommentType = "approve"
	// PullRequestCommentTypeRequestChanges is the type for a request for changes
	PullRequestCommentTypeRequestChanges PullRequestCommentType = "request_changes"
	// PullRequestCommentTypeComment is the type for a regular comment
	PullRequestCommentTypeComment PullRequestCommentType = "comment"
)

// PullRequestCommentInfo is the information for a pull request comment to
// be issued by the provider
type PullRequestCommentInfo struct {
	// The commit sha for the pull request
	Commit string `json:"commit,omitempty"`
	// An optional header for the comment. If aggregating multiple comments, this
	// could be used as a header.
	Header string `json:"header,omitempty"`
	// The comment body
	Body string `json:"body,omitempty"`
	// The priority of the comment. This is used to determine the order of the comments
	// when aggregating multiple comments. Lower values are higher priority.
	Priority int `json:"priority,omitempty"`
	// The type of the comment
	Type PullRequestCommentType `json:"type,omitempty"`
	//
}

// CommentResultMeta is the metadata for the comment result
type CommentResultMeta struct {
	ID          string    `json:"review_id,omitempty"`
	SubmittedAt time.Time `json:"submitted_at,omitempty"`
	URL         string    `json:"pull_request_url,omitempty"`
}

// PullRequestCommenter is the interface for commenting on pull requests
// The provider must implement this interface if it supports commenting on pull requests.
// Providers are assumed to support discovering the pull request by the properties
// as well as discovering the *one* comment they're supposed to work on.
// That is, the provider may issue one comment and aggregate multiple comments into one.
type PullRequestCommenter interface {
	Provider

	// CommentOnPullRequest issues comments on a pull request
	CommentOnPullRequest(
		ctx context.Context, getByProps *properties.Properties, comment PullRequestCommentInfo) (*CommentResultMeta, error)
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
