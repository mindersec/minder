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
	"strings"
	"time"

	backoffv4 "github.com/cenkalti/backoff/v4"
	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v60/github"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/db"
	engerrors "github.com/stacklok/minder/internal/engine/errors"
	gitclient "github.com/stacklok/minder/internal/providers/git"
	"github.com/stacklok/minder/internal/providers/ratecache"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// ExpensiveRestCallTimeout is the timeout for expensive REST calls
	ExpensiveRestCallTimeout = 15 * time.Second
	// MaxRateLimitWait is the maximum time to wait for a rate limit to reset
	MaxRateLimitWait = 5 * time.Minute
	// MaxRateLimitRetries is the maximum number of retries for rate limit errors after waiting
	MaxRateLimitRetries = 1
	// DefaultRateLimitWaitTime is the default time to wait for a rate limit to reset
	DefaultRateLimitWaitTime = 1 * time.Minute
)

var (
	// ErrNotFound Denotes if the call returned a 404
	ErrNotFound = errors.New("not found")
)

// GitHub is the struct that contains the shared GitHub client operations
type GitHub struct {
	client   *github.Client
	cache    ratecache.RestClientCache
	delegate Delegate
}

// Ensure that the GitHub client implements the GitHub interface
var _ provifv1.GitHub = (*GitHub)(nil)

// Delegate is the interface that contains operations that differ between different GitHub actors (user vs app)
type Delegate interface {
	GetCredential() provifv1.GitHubCredential
	ListAllRepositories(context.Context) ([]*minderv1.Repository, error)
	GetUserId(ctx context.Context) (int64, error)
	GetName(ctx context.Context) (string, error)
	GetLogin(ctx context.Context) (string, error)
	GetPrimaryEmail(ctx context.Context) (string, error)
	GetOwner() string
}

// NewGitHub creates a new GitHub client
func NewGitHub(
	client *github.Client,
	cache ratecache.RestClientCache,
	delegate Delegate,
) *GitHub {
	return &GitHub{
		client:   client,
		cache:    cache,
		delegate: delegate,
	}
}

// ListPackagesByRepository returns a list of all packages for a specific repository
func (c *GitHub) ListPackagesByRepository(ctx context.Context, isOrg bool, owner string, artifactType string,
	repositoryId int64, pageNumber int, itemsPerPage int) ([]*github.Package, error) {
	opt := &github.PackageListOptions{
		PackageType: &artifactType,
		ListOptions: github.ListOptions{
			Page:    pageNumber,
			PerPage: itemsPerPage,
		},
	}
	// create a slice to hold the containers
	var allContainers []*github.Package
	for {
		var artifacts []*github.Package
		var resp *github.Response
		var err error

		if isOrg {
			artifacts, resp, err = c.client.Organizations.ListPackages(ctx, owner, opt)
		} else {
			artifacts, resp, err = c.client.Users.ListPackages(ctx, owner, opt)
		}
		if err != nil {
			if resp.StatusCode == http.StatusNotFound {
				return allContainers, fmt.Errorf("packages not found for repository %d: %w", repositoryId, ErrNotFound)
			}

			return allContainers, err
		}

		// now just append the ones belonging to the repository
		for _, artifact := range artifacts {
			if artifact.Repository.GetID() == repositoryId {
				allContainers = append(allContainers, artifact)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allContainers, nil
}

// GetPackageVersions returns a list of all package versions for the authenticated user or org
func (c *GitHub) GetPackageVersions(ctx context.Context, isOrg bool, owner string, package_type string,
	package_name string) ([]*github.PackageVersion, error) {
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
		if isOrg {
			v, resp, err = c.client.Organizations.PackageGetAllVersions(ctx, owner, package_type, package_name, opt)
		} else {
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

// GetPackageVersionByTag returns a single package version for the specific tag
func (c *GitHub) GetPackageVersionByTag(ctx context.Context, isOrg bool, owner string, package_type string,
	package_name string, tag string) (*github.PackageVersion, error) {

	// since the GH API sometimes returns container and sometimes CONTAINER as the type, let's just lowercase it
	package_type = strings.ToLower(package_type)

	// get all versions
	versions, err := c.GetPackageVersions(ctx, isOrg, owner, package_type, package_name)
	if err != nil {
		return nil, err
	}

	// iterate for all versions until we find the specific tag
	for _, version := range versions {
		tags := version.Metadata.Container.Tags
		for _, t := range tags {
			if t == tag {
				return version, nil
			}
		}
	}
	return nil, nil

}

// GetPackageByName returns a single package for the authenticated user or for the org
func (c *GitHub) GetPackageByName(ctx context.Context, isOrg bool, owner string, package_type string,
	package_name string) (*github.Package, error) {
	var pkg *github.Package
	var err error

	// since the GH API sometimes returns container and sometimes CONTAINER as the type, let's just lowercase it
	package_type = strings.ToLower(package_type)

	if isOrg {
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
func (c *GitHub) GetPackageVersionById(
	ctx context.Context,
	isOrg bool,
	owner string,
	packageType string,
	packageName string,
	version int64,
) (*github.PackageVersion, error) {
	var pkgVersion *github.PackageVersion
	var err error

	if isOrg {
		pkgVersion, _, err = c.client.Organizations.PackageGetVersion(ctx, owner, packageType, packageName, version)
		if err != nil {
			return nil, err
		}
	} else {
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
	protection, _, err := c.client.Repositories.GetBranchProtection(ctx, owner, repo_name, branch_name)
	if err != nil {
		return nil, err
	}
	return protection, nil
}

// UpdateBranchProtection updates the branch protection for a given branch
func (c *GitHub) UpdateBranchProtection(
	ctx context.Context, owner, repo, branch string, preq *github.ProtectionRequest,
) error {
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

// GetOwner returns the owner of the repository
func (c *GitHub) GetOwner() string {
	return c.delegate.GetOwner()
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
func (c *GitHub) DeleteHook(ctx context.Context, owner, repo string, id int64) (*github.Response, error) {
	resp, err := c.client.Repositories.DeleteHook(ctx, owner, repo, id)
	return resp, err
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
	delegator := gitclient.NewGit(c.delegate.GetCredential())
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
	if prov.Class.ProviderClass == db.ProviderClassGithub {
		return true
	}
	return false
}
