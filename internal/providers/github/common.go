// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package github provides a client for interacting with the GitHub API
package github

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	backoffv4 "github.com/cenkalti/backoff/v4"
	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v63/github"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/mindersec/minder/internal/db"
	engerrors "github.com/mindersec/minder/internal/engine/errors"
	entprops "github.com/mindersec/minder/internal/entities/properties"
	gitclient "github.com/mindersec/minder/internal/providers/git"
	"github.com/mindersec/minder/internal/providers/github/ghcr"
	"github.com/mindersec/minder/internal/providers/github/properties"
	"github.com/mindersec/minder/internal/providers/ratecache"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	config "github.com/mindersec/minder/pkg/config/server"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

const (
	// ExpensiveRestCallTimeout is the timeout for expensive REST calls
	ExpensiveRestCallTimeout = 15 * time.Second
	// MaxRateLimitWait is the maximum time to wait for a rate limit to reset
	MaxRateLimitWait = 5 * time.Minute
	// MaxRateLimitRetries is the maximum number of retries for rate limit errors after waiting
	MaxRateLimitRetries = 1
	// DefaultRateLimitWaitTime is the default time to wait for a rate limit to reset
	DefaultRateLimitWaitTime = 1 * time.Minute

	githubBranchNotFoundMsg = "Branch not found"
)

var (
	// ErrNotFound denotes if the call returned a 404
	ErrNotFound = errors.New("not found")
	// ErrBranchNotFound denotes if the branch was not found
	ErrBranchNotFound = errors.New("branch not found")
	// ErrNoPackageListingClient denotes if there is no package listing client available
	ErrNoPackageListingClient = errors.New("no package listing client available")
	// ErroNoCheckPermissions is a fixed error returned when the credentialed
	// identity has not been authorized to use the checks API
	ErroNoCheckPermissions = errors.New("missing permissions: check")
	// ErrBranchNameEmpty is a fixed error returned when the branch name is empty
	ErrBranchNameEmpty = errors.New("branch name cannot be empty")
)

// GitHub is the struct that contains the shared GitHub client operations
type GitHub struct {
	client               *github.Client
	packageListingClient *github.Client
	cache                ratecache.RestClientCache
	delegate             Delegate
	ghcrwrap             *ghcr.ImageLister
	gitConfig            config.GitConfig
	webhookConfig        *config.WebhookConfig
	propertyFetchers     properties.GhPropertyFetcherFactory
}

// Ensure that the GitHub client implements the GitHub interface
var _ provifv1.GitHub = (*GitHub)(nil)

// ClientService is an interface for GitHub operations
// It is used to mock GitHub operations in tests, but in order to generate
// mocks, the interface must be exported
type ClientService interface {
	GetInstallation(ctx context.Context, id int64, jwt string) (*github.Installation, *github.Response, error)
	GetUserIdFromToken(ctx context.Context, token *oauth2.Token) (*int64, error)
	ListUserInstallations(ctx context.Context, token *oauth2.Token) ([]*github.Installation, error)
	DeleteInstallation(ctx context.Context, id int64, jwt string) (*github.Response, error)
	GetOrgMembership(ctx context.Context, token *oauth2.Token, org string) (*github.Membership, *github.Response, error)
}

var _ ClientService = (*ClientServiceImplementation)(nil)

// ClientServiceImplementation is the implementation of the ClientService interface
type ClientServiceImplementation struct{}

// GetInstallation is a wrapper for the GitHub API to get an installation
func (ClientServiceImplementation) GetInstallation(
	ctx context.Context,
	installationID int64,
	jwt string,
) (*github.Installation, *github.Response, error) {
	ghClient := github.NewClient(nil).WithAuthToken(jwt)
	return ghClient.Apps.GetInstallation(ctx, installationID)
}

// GetUserIdFromToken is a wrapper for the GitHub API to get the user id from a token
func (ClientServiceImplementation) GetUserIdFromToken(ctx context.Context, token *oauth2.Token) (*int64, error) {
	ghClient := github.NewClient(nil).WithAuthToken(token.AccessToken)

	user, _, err := ghClient.Users.Get(ctx, "")
	if err != nil {
		return nil, err
	}

	return user.ID, nil
}

// ListUserInstallations is a wrapper for the GitHub API to list user installations
func (ClientServiceImplementation) ListUserInstallations(
	ctx context.Context, token *oauth2.Token,
) ([]*github.Installation, error) {
	ghClient := github.NewClient(nil).WithAuthToken(token.AccessToken)

	installations, _, err := ghClient.Apps.ListUserInstallations(ctx, nil)
	return installations, err
}

// DeleteInstallation is a wrapper for the GitHub API to delete an installation
func (ClientServiceImplementation) DeleteInstallation(ctx context.Context, id int64, jwt string) (*github.Response, error) {
	ghClient := github.NewClient(nil).WithAuthToken(jwt)
	return ghClient.Apps.DeleteInstallation(ctx, id)
}

// GetOrgMembership is a wrapper for the GitHub API to get users' organization membership
func (ClientServiceImplementation) GetOrgMembership(
	ctx context.Context, token *oauth2.Token, org string,
) (*github.Membership, *github.Response, error) {
	ghClient := github.NewClient(nil).WithAuthToken(token.AccessToken)
	return ghClient.Organizations.GetOrgMembership(ctx, "", org)
}

// Delegate is the interface that contains operations that differ between different GitHub actors (user vs app)
type Delegate interface {
	GetCredential() provifv1.GitHubCredential
	ListAllRepositories(context.Context) ([]*minderv1.Repository, error)
	GetUserId(ctx context.Context) (int64, error)
	GetMinderUserId(ctx context.Context) (int64, error)
	GetName(ctx context.Context) (string, error)
	GetLogin(ctx context.Context) (string, error)
	GetPrimaryEmail(ctx context.Context) (string, error)
	GetOwner() string
	IsOrg() bool
}

// NewGitHub creates a new GitHub client
func NewGitHub(
	client *github.Client,
	packageListingClient *github.Client,
	cache ratecache.RestClientCache,
	delegate Delegate,
	cfg *config.ProviderConfig,
	whcfg *config.WebhookConfig,
	propertyFetchers properties.GhPropertyFetcherFactory,
) *GitHub {
	var gitConfig config.GitConfig
	if cfg != nil {
		gitConfig = cfg.Git
	}
	return &GitHub{
		client:               client,
		packageListingClient: packageListingClient,
		cache:                cache,
		delegate:             delegate,
		ghcrwrap:             ghcr.FromGitHubClient(client, delegate.GetOwner()),
		gitConfig:            gitConfig,
		webhookConfig:        whcfg,
		propertyFetchers:     propertyFetchers,
	}
}

// CanImplement returns true/false depending on whether the Provider
// can implement the specified trait
func (_ *GitHub) CanImplement(trait minderv1.ProviderType) bool {
	return trait == minderv1.ProviderType_PROVIDER_TYPE_GITHUB ||
		trait == minderv1.ProviderType_PROVIDER_TYPE_GIT ||
		trait == minderv1.ProviderType_PROVIDER_TYPE_REST ||
		trait == minderv1.ProviderType_PROVIDER_TYPE_REPO_LISTER
}

// ListPackagesByRepository returns a list of all packages for a specific repository
func (c *GitHub) ListPackagesByRepository(
	ctx context.Context,
	owner string,
	artifactType string,
	repositoryId int64,
	pageNumber int,
	itemsPerPage int,
) ([]*github.Package, error) {
	opt := &github.PackageListOptions{
		PackageType: &artifactType,
		ListOptions: github.ListOptions{
			Page:    pageNumber,
			PerPage: itemsPerPage,
		},
	}
	// create a slice to hold the containers
	var allContainers []*github.Package

	if c.packageListingClient == nil {
		zerolog.Ctx(ctx).Error().Msg("No client available for listing packages")
		return allContainers, ErrNoPackageListingClient
	}

	type listPackagesRespWrapper struct {
		artifacts []*github.Package
		resp      *github.Response
	}
	op := func() (listPackagesRespWrapper, error) {
		var artifacts []*github.Package
		var resp *github.Response
		var err error

		if c.IsOrg() {
			artifacts, resp, err = c.packageListingClient.Organizations.ListPackages(ctx, owner, opt)
		} else {
			artifacts, resp, err = c.packageListingClient.Users.ListPackages(ctx, owner, opt)
		}

		listPackagesResp := listPackagesRespWrapper{
			artifacts: artifacts,
			resp:      resp,
		}

		if isRateLimitError(err) {
			waitErr := c.waitForRateLimitReset(ctx, err)
			if waitErr == nil {
				return listPackagesResp, err
			}
			return listPackagesResp, backoffv4.Permanent(err)
		}

		return listPackagesResp, backoffv4.Permanent(err)
	}

	for {
		result, err := performWithRetry(ctx, op)
		if err != nil {
			if result.resp != nil && result.resp.StatusCode == http.StatusNotFound {
				return allContainers, fmt.Errorf("packages not found for repository %d: %w", repositoryId, ErrNotFound)
			}

			return allContainers, err
		}

		// now just append the ones belonging to the repository
		for _, artifact := range result.artifacts {
			if artifact.Repository.GetID() == repositoryId {
				allContainers = append(allContainers, artifact)
			}
		}

		if result.resp.NextPage == 0 {
			break
		}
		opt.Page = result.resp.NextPage
	}

	return allContainers, nil
}

// getPackageVersions returns a list of all package versions for the authenticated user or org
func (c *GitHub) getPackageVersions(ctx context.Context, owner string, package_type string, package_name string,
) ([]*github.PackageVersion, error) {
	state := "active"

	// since the GH API sometimes returns container and sometimes CONTAINER as the type, let's just lowercase it
	package_type = strings.ToLower(package_type)

	opt := &github.PackageListOptions{
		PackageType: &package_type,
		State:       &state,
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 100,
		},
	}

	// create a slice to hold the versions
	var allVersions []*github.PackageVersion

	// loop until we get all package versions
	for {
		var v []*github.PackageVersion
		var resp *github.Response
		var err error
		if c.IsOrg() {
			v, resp, err = c.client.Organizations.PackageGetAllVersions(ctx, owner, package_type, package_name, opt)
		} else {
			package_name = url.PathEscape(package_name)
			v, resp, err = c.client.Users.PackageGetAllVersions(ctx, owner, package_type, package_name, opt)
		}
		if err != nil {
			return nil, err
		}

		// append to the slice
		allVersions = append(allVersions, v...)

		// if there is no next page, break
		if resp.NextPage == 0 {
			break
		}

		// update the page
		opt.Page = resp.NextPage
	}

	// return the slice
	return allVersions, nil
}

// GetPackageByName returns a single package for the authenticated user or for the org
func (c *GitHub) GetPackageByName(ctx context.Context, owner string, package_type string, package_name string,
) (*github.Package, error) {
	var pkg *github.Package
	var err error

	// since the GH API sometimes returns container and sometimes CONTAINER as the type, let's just lowercase it
	package_type = strings.ToLower(package_type)

	if c.IsOrg() {
		pkg, _, err = c.client.Organizations.GetPackage(ctx, owner, package_type, package_name)
		if err != nil {
			return nil, err
		}
	} else {
		pkg, _, err = c.client.Users.GetPackage(ctx, "", package_type, package_name)
		if err != nil {
			return nil, err
		}
	}
	return pkg, nil
}

// GetPackageVersionById returns a single package version for the specific id
func (c *GitHub) GetPackageVersionById(ctx context.Context, owner string, packageType string, packageName string,
	version int64) (*github.PackageVersion, error) {
	var pkgVersion *github.PackageVersion
	var err error

	if c.IsOrg() {
		pkgVersion, _, err = c.client.Organizations.PackageGetVersion(ctx, owner, packageType, packageName, version)
		if err != nil {
			return nil, err
		}
	} else {
		packageName = url.PathEscape(packageName)
		pkgVersion, _, err = c.client.Users.PackageGetVersion(ctx, owner, packageType, packageName, version)
		if err != nil {
			return nil, err
		}
	}

	return pkgVersion, nil
}

// GetPullRequest is a wrapper for the GitHub API to get a pull request
func (c *GitHub) GetPullRequest(
	ctx context.Context,
	owner string,
	repo string,
	number int,
) (*github.PullRequest, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// ListFiles is a wrapper for the GitHub API to list files in a pull request
func (c *GitHub) ListFiles(
	ctx context.Context,
	owner string,
	repo string,
	prNumber int,
	perPage int,
	pageNumber int,
) ([]*github.CommitFile, *github.Response, error) {
	type listFilesRespWrapper struct {
		files []*github.CommitFile
		resp  *github.Response
	}

	op := func() (listFilesRespWrapper, error) {
		opt := &github.ListOptions{
			Page:    pageNumber,
			PerPage: perPage,
		}
		files, resp, err := c.client.PullRequests.ListFiles(ctx, owner, repo, prNumber, opt)

		listFileResp := listFilesRespWrapper{
			files: files,
			resp:  resp,
		}

		if isRateLimitError(err) {
			waitErr := c.waitForRateLimitReset(ctx, err)
			if waitErr == nil {
				return listFileResp, err
			}
			return listFileResp, backoffv4.Permanent(err)
		}

		return listFileResp, backoffv4.Permanent(err)
	}

	resp, err := performWithRetry(ctx, op)
	return resp.files, resp.resp, err
}

// CommentOnPullRequest implements the CommentOnPullRequest method of the GitHub interface
func (c *GitHub) CommentOnPullRequest(
	ctx context.Context, getByProps *entprops.Properties, comment provifv1.PullRequestCommentInfo,
) (*provifv1.CommentResultMeta, error) {
	owner := getByProps.GetProperty(properties.PullPropertyRepoOwner).GetString()
	name := getByProps.GetProperty(properties.PullPropertyRepoName).GetString()
	prNum := getByProps.GetProperty(properties.PullPropertyNumber).GetInt64()
	authorID := getByProps.GetProperty(properties.PullPropertyAuthorID).GetInt64()

	authorizedUser, err := c.delegate.GetMinderUserId(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get authenticated user: %w", err)
	}
	mci, err := c.findPreviousStatusComment(ctx, owner, name, prNum, authorizedUser)
	if err != nil {
		return nil, fmt.Errorf("could not find previous status comment: %w", err)
	}

	mci, err = c.updateOrSubmitComment(
		ctx, mci, comment, owner, name, prNum, comment.Commit, authorID, authorizedUser)
	if err != nil {
		// this should be fatal. In case we can't submit the review, we can't proceed
		return nil, fmt.Errorf("could not submit review: %w", err)
	}
	return mci.ToCommentResultMeta(), nil
}

// CreateReview is a wrapper for the GitHub API to create a review
func (c *GitHub) CreateReview(
	ctx context.Context, owner, repo string, number int, reviewRequest *github.PullRequestReviewRequest,
) (*github.PullRequestReview, error) {
	review, _, err := c.client.PullRequests.CreateReview(ctx, owner, repo, number, reviewRequest)
	if err != nil {
		return nil, fmt.Errorf("error creating review: %w", err)
	}
	return review, nil
}

// UpdateReview is a wrapper for the GitHub API to update a review
func (c *GitHub) UpdateReview(
	ctx context.Context, owner, repo string, number int, reviewId int64, body string,
) (*github.PullRequestReview, error) {
	review, _, err := c.client.PullRequests.UpdateReview(ctx, owner, repo, number, reviewId, body)
	if err != nil {
		return nil, fmt.Errorf("error updating review: %w", err)
	}
	return review, nil
}

// ListIssueComments is a wrapper for the GitHub API to get all comments in a review
func (c *GitHub) ListIssueComments(
	ctx context.Context, owner, repo string, number int, opts *github.IssueListCommentsOptions,
) ([]*github.IssueComment, error) {
	comments, _, err := c.client.Issues.ListComments(ctx, owner, repo, number, opts)
	if err != nil {
		return nil, fmt.Errorf("error getting list of comments: %w", err)
	}
	return comments, nil
}

// ListReviews is a wrapper for the GitHub API to list reviews
func (c *GitHub) ListReviews(
	ctx context.Context,
	owner, repo string,
	number int,
	opt *github.ListOptions,
) ([]*github.PullRequestReview, error) {
	reviews, _, err := c.client.PullRequests.ListReviews(ctx, owner, repo, number, opt)
	if err != nil {
		return nil, fmt.Errorf("error listing reviews for PR %s/%s/%d: %w", owner, repo, number, err)
	}
	return reviews, nil
}

// DismissReview is a wrapper for the GitHub API to dismiss a review
func (c *GitHub) DismissReview(
	ctx context.Context,
	owner, repo string,
	prId int,
	reviewId int64,
	dismissalRequest *github.PullRequestReviewDismissalRequest,
) (*github.PullRequestReview, error) {
	review, _, err := c.client.PullRequests.DismissReview(ctx, owner, repo, prId, reviewId, dismissalRequest)
	if err != nil {
		return nil, fmt.Errorf("error dismissing review %d for PR %s/%s/%d: %w", reviewId, owner, repo, prId, err)
	}
	return review, nil
}

// SetCommitStatus is a wrapper for the GitHub API to set a commit status
func (c *GitHub) SetCommitStatus(
	ctx context.Context, owner, repo, ref string, status *github.RepoStatus,
) (*github.RepoStatus, error) {
	status, _, err := c.client.Repositories.CreateStatus(ctx, owner, repo, ref, status)
	if err != nil {
		return nil, fmt.Errorf("error creating commit status: %w", err)
	}
	return status, nil
}

// GetRepository returns a single repository for the authenticated user
func (c *GitHub) GetRepository(ctx context.Context, owner string, name string) (*github.Repository, error) {
	// create a slice to hold the repositories
	repo, _, err := c.client.Repositories.Get(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("error getting repository: %w", err)
	}

	return repo, nil
}

// GetBranchProtection returns the branch protection for a given branch
func (c *GitHub) GetBranchProtection(ctx context.Context, owner string,
	repo_name string, branch_name string) (*github.Protection, error) {
	var respErr *github.ErrorResponse
	if branch_name == "" {
		return nil, ErrBranchNameEmpty
	}

	protection, _, err := c.client.Repositories.GetBranchProtection(ctx, owner, repo_name, branch_name)
	if errors.As(err, &respErr) {
		if respErr.Message == githubBranchNotFoundMsg {
			return nil, ErrBranchNotFound
		}

		return nil, fmt.Errorf("error getting branch protection: %w", err)
	} else if err != nil {
		return nil, err
	}
	return protection, nil
}

// UpdateBranchProtection updates the branch protection for a given branch
func (c *GitHub) UpdateBranchProtection(
	ctx context.Context, owner, repo, branch string, preq *github.ProtectionRequest,
) error {
	if branch == "" {
		return ErrBranchNameEmpty
	}
	_, _, err := c.client.Repositories.UpdateBranchProtection(ctx, owner, repo, branch, preq)
	return err
}

// GetBaseURL returns the base URL for the REST API.
func (c *GitHub) GetBaseURL() string {
	return c.client.BaseURL.String()
}

// NewRequest creates an API request. A relative URL can be provided in urlStr,
// which will be resolved to the BaseURL of the Client. Relative URLS should
// always be specified without a preceding slash. If specified, the value
// pointed to by body is JSON encoded and included as the request body.
func (c *GitHub) NewRequest(method, requestUrl string, body any) (*http.Request, error) {
	return c.client.NewRequest(method, requestUrl, body)
}

// Do sends an API request and returns the API response.
func (c *GitHub) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	var buf bytes.Buffer

	// The GitHub client closes the response body, so we need to capture it
	// in a buffer so that we can return it to the caller
	resp, err := c.client.Do(ctx, req, &buf)
	if err != nil && resp == nil {
		return nil, err
	}

	if resp.Response != nil {
		resp.Response.Body = io.NopCloser(&buf)
	}
	return resp.Response, err
}

// GetCredential returns the credential used to authenticate with the GitHub API
func (c *GitHub) GetCredential() provifv1.GitHubCredential {
	return c.delegate.GetCredential()
}

// IsOrg returns true if the owner is an organization
func (c *GitHub) IsOrg() bool {
	return c.delegate.IsOrg()
}

// ListHooks lists all Hooks for the specified repository.
func (c *GitHub) ListHooks(ctx context.Context, owner, repo string) ([]*github.Hook, error) {
	list, resp, err := c.client.Repositories.ListHooks(ctx, owner, repo, nil)
	if err != nil && resp.StatusCode == http.StatusNotFound {
		// return empty list so that the caller can ignore the error and iterate over the empty list
		return []*github.Hook{}, fmt.Errorf("hooks not found for repository %s/%s: %w", owner, repo, ErrNotFound)
	}
	return list, err
}

// DeleteHook deletes a specified Hook.
func (c *GitHub) DeleteHook(ctx context.Context, owner, repo string, id int64) error {
	resp, err := c.client.Repositories.DeleteHook(ctx, owner, repo, id)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		// If the hook is not found, we can ignore the
		// error, user might have deleted it manually.
		return nil
	}
	if resp != nil && resp.StatusCode == http.StatusForbidden {
		// We ignore deleting webhooks that we're not
		// allowed to touch. This is usually the case
		// with repository transfer.
		return nil
	}
	return err
}

// CreateHook creates a new Hook.
func (c *GitHub) CreateHook(ctx context.Context, owner, repo string, hook *github.Hook) (*github.Hook, error) {
	h, _, err := c.client.Repositories.CreateHook(ctx, owner, repo, hook)
	return h, err
}

// EditHook edits an existing Hook.
func (c *GitHub) EditHook(ctx context.Context, owner, repo string, id int64, hook *github.Hook) (*github.Hook, error) {
	h, _, err := c.client.Repositories.EditHook(ctx, owner, repo, id, hook)
	return h, err
}

// CreateSecurityAdvisory creates a new security advisory
func (c *GitHub) CreateSecurityAdvisory(ctx context.Context, owner, repo, severity, summary, description string,
	v []*github.AdvisoryVulnerability) (string, error) {
	u := fmt.Sprintf("repos/%v/%v/security-advisories", owner, repo)

	payload := &struct {
		Summary         string                          `json:"summary"`
		Description     string                          `json:"description"`
		Severity        string                          `json:"severity"`
		Vulnerabilities []*github.AdvisoryVulnerability `json:"vulnerabilities"`
	}{
		Summary:         summary,
		Description:     description,
		Severity:        severity,
		Vulnerabilities: v,
	}
	req, err := c.client.NewRequest("POST", u, payload)
	if err != nil {
		return "", err
	}

	res := &struct {
		ID string `json:"ghsa_id"`
	}{}

	resp, err := c.client.Do(ctx, req, res)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("error creating security advisory: %v", resp.Status)
	}
	return res.ID, nil
}

// CloseSecurityAdvisory closes a security advisory
func (c *GitHub) CloseSecurityAdvisory(ctx context.Context, owner, repo, id string) error {
	u := fmt.Sprintf("repos/%v/%v/security-advisories/%v", owner, repo, id)

	payload := &struct {
		State string `json:"state"`
	}{
		State: "closed",
	}

	req, err := c.client.NewRequest("PATCH", u, payload)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(ctx, req, nil)
	if err != nil {
		return err
	}
	// Translate the HTTP status code to an error, nil if between 200 and 299
	return engerrors.HTTPErrorCodeToErr(resp.StatusCode)
}

// CreatePullRequest creates a pull request in a repository.
func (c *GitHub) CreatePullRequest(
	ctx context.Context,
	owner, repo, title, body, head, base string,
) (*github.PullRequest, error) {
	pr, _, err := c.client.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title:               github.String(title),
		Body:                github.String(body),
		Head:                github.String(head),
		Base:                github.String(base),
		MaintainerCanModify: github.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	return pr, nil
}

// ClosePullRequest closes a pull request in a repository.
func (c *GitHub) ClosePullRequest(ctx context.Context, owner, repo string, number int) (*github.PullRequest, error) {
	pr, _, err := c.client.PullRequests.Edit(ctx, owner, repo, number, &github.PullRequest{
		State: github.String("closed"),
	})
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// ListPullRequests lists all pull requests in a repository.
func (c *GitHub) ListPullRequests(
	ctx context.Context,
	owner, repo string,
	opt *github.PullRequestListOptions,
) ([]*github.PullRequest, error) {
	prs, _, err := c.client.PullRequests.List(ctx, owner, repo, opt)
	if err != nil {
		return nil, err
	}

	return prs, nil
}

// CreateIssueComment creates a comment on a pull request or an issue
func (c *GitHub) CreateIssueComment(
	ctx context.Context, owner, repo string, number int, comment string,
) (*github.IssueComment, error) {
	var issueComment *github.IssueComment

	op := func() (any, error) {
		var err error

		issueComment, _, err = c.client.Issues.CreateComment(ctx, owner, repo, number, &github.IssueComment{
			Body: &comment,
		})

		if isRateLimitError(err) {
			waitWrr := c.waitForRateLimitReset(ctx, err)
			if waitWrr == nil {
				return nil, err
			}
			return nil, backoffv4.Permanent(err)
		}

		return nil, backoffv4.Permanent(err)
	}
	_, retryErr := performWithRetry(ctx, op)
	return issueComment, retryErr
}

// UpdateIssueComment updates a comment on a pull request or an issue
func (c *GitHub) UpdateIssueComment(ctx context.Context, owner, repo string, number int64, comment string) error {
	_, _, err := c.client.Issues.EditComment(ctx, owner, repo, number, &github.IssueComment{
		Body: &comment,
	})
	return err
}

// Clone clones a GitHub repository
func (c *GitHub) Clone(ctx context.Context, cloneUrl string, branch string) (*git.Repository, error) {
	delegator := gitclient.NewGit(c.delegate.GetCredential(), gitclient.WithConfig(c.gitConfig))
	return delegator.Clone(ctx, cloneUrl, branch)
}

// AddAuthToPushOptions adds authorization to the push options
func (c *GitHub) AddAuthToPushOptions(ctx context.Context, pushOptions *git.PushOptions) error {
	login, err := c.delegate.GetLogin(ctx)
	if err != nil {
		return fmt.Errorf("cannot get login: %w", err)
	}
	c.delegate.GetCredential().AddToPushOptions(pushOptions, login)
	return nil
}

// ListAllRepositories lists all repositories the credential has access to
func (c *GitHub) ListAllRepositories(ctx context.Context) ([]*minderv1.Repository, error) {
	return c.delegate.ListAllRepositories(ctx)
}

// GetUserId returns the user id for the acting user
func (c *GitHub) GetUserId(ctx context.Context) (int64, error) {
	return c.delegate.GetUserId(ctx)
}

// GetName returns the username for the acting user
func (c *GitHub) GetName(ctx context.Context) (string, error) {
	return c.delegate.GetName(ctx)
}

// GetLogin returns the login for the acting user
func (c *GitHub) GetLogin(ctx context.Context) (string, error) {
	return c.delegate.GetLogin(ctx)
}

// GetPrimaryEmail returns the primary email for the acting user
func (c *GitHub) GetPrimaryEmail(ctx context.Context) (string, error) {
	return c.delegate.GetPrimaryEmail(ctx)
}

// ListImages lists all containers in the GitHub Container Registry
func (c *GitHub) ListImages(ctx context.Context) ([]string, error) {
	return c.ghcrwrap.ListImages(ctx)
}

// GetNamespaceURL returns the URL for the repository
func (c *GitHub) GetNamespaceURL() string {
	return c.ghcrwrap.GetNamespaceURL()
}

// GetArtifactVersions returns a list of all versions for a specific artifact
func (gv *GitHub) GetArtifactVersions(
	ctx context.Context, artifact *minderv1.Artifact,
	filter provifv1.GetArtifactVersionsFilter,
) ([]*minderv1.ArtifactVersion, error) {
	// We don't need to URL-encode the artifact name
	// since this already happens in go-github
	upstreamVersions, err := gv.getPackageVersions(
		ctx, artifact.GetOwner(), artifact.GetTypeLower(), artifact.GetName(),
	)
	if err != nil {
		return nil, fmt.Errorf("error retrieving artifact versions: %w", err)
	}

	out := make([]*minderv1.ArtifactVersion, 0, len(upstreamVersions))
	for _, uv := range upstreamVersions {
		tags := uv.Metadata.Container.Tags

		if err := filter.IsSkippable(uv.CreatedAt.Time, tags); err != nil {
			zerolog.Ctx(ctx).Debug().Str("name", artifact.GetName()).Strs("tags", tags).
				Str("reason", err.Error()).Msg("skipping artifact version")
			continue
		}

		sort.Strings(tags)

		// only the tags and creation time is relevant to us.
		out = append(out, &minderv1.ArtifactVersion{
			Tags: tags,
			// NOTE: GitHub's name is actually a SHA. This is misleading...
			// but it is what it is. We'll use it as the SHA for now.
			Sha:       *uv.Name,
			CreatedAt: timestamppb.New(uv.CreatedAt.Time),
		})
	}

	return out, nil
}

// setAsRateLimited adds the GitHub to the cache as rate limited.
// An optimistic concurrency control mechanism is used to ensure that every request doesn't need
// synchronization. GitHub only adds itself to the cache if it's not already there. It doesn't
// remove itself from the cache when the rate limit is reset. This approach leverages the high
// likelihood of the client or token being rate-limited again. By keeping the client in the cache,
// we can reuse client's rateLimits map, which holds rate limits for different endpoints.
// This reuse of cached rate limits helps avoid unnecessary GitHub API requests when the client
// is rate-limited. Every cache entry has an expiration time, so the cache will eventually evict
// the rate-limited client.
func (c *GitHub) setAsRateLimited() {
	if c.cache != nil {
		c.cache.Set(c.delegate.GetOwner(), c.delegate.GetCredential().GetCacheKey(), db.ProviderTypeGithub, c)
	}
}

// waitForRateLimitReset waits for token wait limit to reset. Returns error if wait time is more
// than MaxRateLimitWait or requests' context is cancelled.
func (c *GitHub) waitForRateLimitReset(ctx context.Context, err error) error {
	var rateLimitError *github.RateLimitError
	isRateLimitErr := errors.As(err, &rateLimitError)

	if isRateLimitErr {
		return c.processPrimaryRateLimitErr(ctx, rateLimitError)
	}

	var abuseRateLimitError *github.AbuseRateLimitError
	isAbuseRateLimitErr := errors.As(err, &abuseRateLimitError)

	if isAbuseRateLimitErr {
		return c.processAbuseRateLimitErr(ctx, abuseRateLimitError)
	}

	return nil
}

func (c *GitHub) processPrimaryRateLimitErr(ctx context.Context, err *github.RateLimitError) error {
	logger := zerolog.Ctx(ctx)
	rate := err.Rate
	if rate.Remaining == 0 {
		c.setAsRateLimited()

		waitTime := DefaultRateLimitWaitTime
		resetTime := rate.Reset.Time
		if !resetTime.IsZero() {
			waitTime = time.Until(resetTime)
		}

		logRateLimitError(logger, "RateLimitError", waitTime, c.delegate.GetOwner(), err.Response)

		if waitTime > MaxRateLimitWait {
			logger.Debug().Msgf("rate limit reset time: %v exceeds maximum wait time: %v", waitTime, MaxRateLimitWait)
			return err
		}

		// Wait for the rate limit to reset
		select {
		case <-time.After(waitTime):
			return nil
		case <-ctx.Done():
			logger.Debug().Err(ctx.Err()).Msg("context done while waiting for rate limit to reset")
			return err
		}
	}

	return nil
}

func (c *GitHub) processAbuseRateLimitErr(ctx context.Context, err *github.AbuseRateLimitError) error {
	logger := zerolog.Ctx(ctx)
	c.setAsRateLimited()

	retryAfter := err.RetryAfter
	waitTime := DefaultRateLimitWaitTime
	if retryAfter != nil && *retryAfter > 0 {
		waitTime = *retryAfter
	}

	logRateLimitError(logger, "AbuseRateLimitError", waitTime, c.delegate.GetOwner(), err.Response)

	if waitTime > MaxRateLimitWait {
		logger.Debug().Msgf("abuse rate limit wait time: %v exceeds maximum wait time: %v", waitTime, MaxRateLimitWait)
		return err
	}

	// Wait for the rate limit to reset
	select {
	case <-time.After(waitTime):
		return nil
	case <-ctx.Done():
		logger.Debug().Err(ctx.Err()).Msg("context done while waiting for rate limit to reset")
		return err
	}
}

func logRateLimitError(logger *zerolog.Logger, errType string, waitTime time.Duration, owner string, resp *http.Response) {
	var method, path string
	if resp != nil && resp.Request != nil {
		method = resp.Request.Method
		path = resp.Request.URL.Path
	}

	event := logger.Debug().
		Str("owner", owner).
		Str("wait_time", waitTime.String()).
		Str("error_type", errType)

	if method != "" {
		event = event.Str("method", method)
	}

	if path != "" {
		event = event.Str("path", path)
	}

	event.Msg("rate limit exceeded")
}

func performWithRetry[T any](ctx context.Context, op backoffv4.OperationWithData[T]) (T, error) {
	exponentialBackOff := backoffv4.NewExponentialBackOff()
	maxRetriesBackoff := backoffv4.WithMaxRetries(exponentialBackOff, MaxRateLimitRetries)
	return backoffv4.RetryWithData(op, backoffv4.WithContext(maxRetriesBackoff, ctx))
}

func isRateLimitError(err error) bool {
	var rateLimitError *github.RateLimitError
	isRateLimitErr := errors.As(err, &rateLimitError)

	var abuseRateLimitError *github.AbuseRateLimitError
	isAbuseRateLimitErr := errors.As(err, &abuseRateLimitError)

	return isRateLimitErr || isAbuseRateLimitErr
}

// IsMinderHook checks if a GitHub hook is a Minder hook
func IsMinderHook(hook *github.Hook, hostURL string) (bool, error) {
	configURL := hook.GetConfig().GetURL()
	if configURL == "" {
		return false, fmt.Errorf("unexpected hook config structure: %v", hook.Config)
	}
	parsedURL, err := url.Parse(configURL)
	if err != nil {
		return false, err
	}
	if parsedURL.Host == hostURL {
		return true, nil
	}

	return false, nil
}

// CanHandleOwner checks if the GitHub provider has the right credentials to handle the owner
func CanHandleOwner(_ context.Context, prov db.Provider, owner string) bool {
	// TODO: this is fragile and does not handle organization renames, in the future we can make sure the credential
	// has admin permissions on the owner
	if prov.Name == fmt.Sprintf("%s-%s", db.ProviderClassGithubApp, owner) {
		return true
	}
	if prov.Class == db.ProviderClassGithub {
		return true
	}
	return false
}

// NewFallbackTokenClient creates a new GitHub client that uses the GitHub App's fallback token
func NewFallbackTokenClient(appConfig config.ProviderConfig) *github.Client {
	if appConfig.GitHubApp == nil {
		return nil
	}
	fallbackToken, err := appConfig.GitHubApp.GetFallbackToken()
	if err != nil || fallbackToken == "" {
		return nil
	}
	var packageListingClient *github.Client

	fallbackTokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: fallbackToken},
	)
	fallbackTokenTC := &http.Client{
		Transport: &oauth2.Transport{
			Base:   http.DefaultClient.Transport,
			Source: fallbackTokenSource,
		},
	}

	packageListingClient = github.NewClient(fallbackTokenTC)
	return packageListingClient
}

// StartCheckRun calls the GitHub API to initialize a new check using the
// supplied options.
func (c *GitHub) StartCheckRun(
	ctx context.Context, owner, repo string, opts *github.CreateCheckRunOptions,
) (*github.CheckRun, error) {
	if opts.StartedAt == nil {
		opts.StartedAt = &github.Timestamp{Time: time.Now()}
	}

	run, resp, err := c.client.Checks.CreateCheckRun(ctx, owner, repo, *opts)
	if err != nil {
		// If error is 403 then it means we are missing permissions
		if resp.StatusCode == 403 {
			return nil, fmt.Errorf("missing permissions: check")
		}
		return nil, ErroNoCheckPermissions
	}
	return run, nil
}

// UpdateCheckRun updates an existing check run in GitHub. The check run is referenced
// using its run ID. This function returns the updated CheckRun srtuct.
func (c *GitHub) UpdateCheckRun(
	ctx context.Context, owner, repo string, checkRunID int64, opts *github.UpdateCheckRunOptions,
) (*github.CheckRun, error) {
	run, resp, err := c.client.Checks.UpdateCheckRun(ctx, owner, repo, checkRunID, *opts)
	if err != nil {
		// If error is 403 then it means we are missing permissions
		if resp.StatusCode == 403 {
			return nil, ErroNoCheckPermissions
		}
		return nil, fmt.Errorf("updating check: %w", err)
	}
	return run, nil
}
