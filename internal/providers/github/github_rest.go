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

	"github.com/google/go-github/v53/github"

	mediatorv1 "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

var (
	// ErrNotFound Denotes if the call returned a 404
	ErrNotFound = errors.New("not found")
)

// ListUserRepositories returns a list of all repositories for the authenticated user
func (c *RestClient) ListUserRepositories(ctx context.Context, owner string) ([]*mediatorv1.Repository, error) {
	repos, err := c.ListAllRepositories(ctx, false, owner)
	if err != nil {
		return nil, err
	}

	return convertRepositories(repos), nil
}

// ListOrganizationRepsitories returns a list of all repositories for the organization
func (c *RestClient) ListOrganizationRepsitories(ctx context.Context, owner string) ([]*mediatorv1.Repository, error) {
	repos, err := c.ListAllRepositories(ctx, true, owner)
	if err != nil {
		return nil, err
	}

	return convertRepositories(repos), nil
}

func convertRepositories(repos []*github.Repository) []*mediatorv1.Repository {
	var converted []*mediatorv1.Repository
	for _, repo := range repos {
		converted = append(converted, convertRepository(repo))
	}
	return converted
}

func convertRepository(repo *github.Repository) *mediatorv1.Repository {
	return &mediatorv1.Repository{
		Name:      repo.GetName(),
		Owner:     repo.GetOwner().GetLogin(),
		RepoId:    int32(repo.GetID()), // FIXME this is a 64 bit int
		HookUrl:   repo.GetHooksURL(),
		DeployUrl: repo.GetDeploymentsURL(),
		CloneUrl:  repo.GetCloneURL(),
		IsPrivate: *repo.Private,
		IsFork:    *repo.Fork,
	}
}

// ListAllRepositories returns a list of all repositories for the authenticated user
// Two APIs are available, contigent on whether the token is for a user or an organization
func (c *RestClient) ListAllRepositories(ctx context.Context, isOrg bool, owner string) ([]*github.Repository, error) {
	opt := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
		Affiliation: "owner",
	}

	orgOpt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	// create a slice to hold the repositories
	var allRepos []*github.Repository
	for {
		var repos []*github.Repository
		var resp *github.Response
		var err error

		if isOrg {
			repos, resp, err = c.client.Repositories.ListByOrg(ctx, owner, orgOpt)
		} else {
			repos, resp, err = c.client.Repositories.List(ctx, "", opt)
		}

		if err != nil {
			return allRepos, err
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}

		if isOrg {
			orgOpt.Page = resp.NextPage
		} else {
			opt.Page = resp.NextPage
		}
	}

	return allRepos, nil
}

// ListAllPackages returns a list of all packages for the authenticated user
func (c *RestClient) ListAllPackages(ctx context.Context, isOrg bool, owner string, artifactType string,
	pageNumber int, itemsPerPage int) ([]*github.Package, error) {
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
			artifacts, resp, err = c.client.Users.ListPackages(ctx, "", opt)
		}
		if err != nil {
			return allContainers, err
		}

		allContainers = append(allContainers, artifacts...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allContainers, nil
}

// ListPackagesByRepository returns a list of all packages for an specific repository
func (c *RestClient) ListPackagesByRepository(ctx context.Context, isOrg bool, owner string, artifactType string,
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
			artifacts, resp, err = c.client.Users.ListPackages(ctx, "", opt)
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

// GetPackageByName returns a single package for the authenticated user or for the org
func (c *RestClient) GetPackageByName(ctx context.Context, isOrg bool, owner string, package_type string,
	package_name string) (*github.Package, error) {
	var pkg *github.Package
	var err error

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

// GetPackageVersions returns a list of all package versions for the authenticated user or org
func (c *RestClient) GetPackageVersions(ctx context.Context, isOrg bool, owner string, package_type string,
	package_name string) ([]*github.PackageVersion, error) {
	var versions []*github.PackageVersion
	var err error
	state := "active"

	if isOrg {
		versions, _, err = c.client.Organizations.PackageGetAllVersions(ctx, owner, package_type,
			package_name, &github.PackageListOptions{PackageType: &package_type, State: &state})
		if err != nil {
			return nil, err
		}
	} else {
		versions, _, err = c.client.Users.PackageGetAllVersions(ctx, "", package_type,
			package_name, &github.PackageListOptions{PackageType: &package_type, State: &state})
		if err != nil {
			return nil, err
		}
	}

	return versions, nil
}

// GetPackageVersionByTag returns a single package version for the specific tag
func (c *RestClient) GetPackageVersionByTag(ctx context.Context, isOrg bool, owner string, package_type string,
	package_name string, tag string) (*github.PackageVersion, error) {
	var versions []*github.PackageVersion
	var err error
	state := "active"

	if isOrg {
		versions, _, err = c.client.Organizations.PackageGetAllVersions(ctx, owner, package_type,
			package_name, &github.PackageListOptions{PackageType: &package_type, State: &state})
		if err != nil {
			return nil, err
		}
	} else {
		versions, _, err = c.client.Users.PackageGetAllVersions(ctx, "", package_type,
			package_name, &github.PackageListOptions{PackageType: &package_type, State: &state})
		if err != nil {
			return nil, err
		}
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

// GetPackageVersionById returns a single package version for the specific id
func (c *RestClient) GetPackageVersionById(
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
func (c *RestClient) GetPullRequest(
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
func (c *RestClient) ListFiles(
	ctx context.Context,
	owner string,
	repo string,
	prNumber int,
	perPage int,
	pageNumber int,
) ([]*github.CommitFile, *github.Response, error) {
	opt := &github.ListOptions{
		Page:    pageNumber,
		PerPage: perPage,
	}
	return c.client.PullRequests.ListFiles(ctx, owner, repo, prNumber, opt)
}

// CreateReview is a wrapper for the GitHub API to create a review
func (c *RestClient) CreateReview(
	ctx context.Context, owner, repo string, number int, reviewRequest *github.PullRequestReviewRequest,
) (*github.PullRequestReview, error) {
	review, _, err := c.client.PullRequests.CreateReview(ctx, owner, repo, number, reviewRequest)
	if err != nil {
		return nil, fmt.Errorf("error creating review: %w", err)
	}
	return review, nil
}

// ListReviews is a wrapper for the GitHub API to list reviews
func (c *RestClient) ListReviews(
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
func (c *RestClient) DismissReview(
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
func (c *RestClient) SetCommitStatus(
	ctx context.Context, owner, repo, ref string, status *github.RepoStatus,
) (*github.RepoStatus, error) {
	status, _, err := c.client.Repositories.CreateStatus(ctx, owner, repo, ref, status)
	if err != nil {
		return nil, fmt.Errorf("error creating commit status: %w", err)
	}
	return status, nil
}

// GetRepository returns a single repository for the authenticated user
func (c *RestClient) GetRepository(ctx context.Context, owner string, name string) (*github.Repository, error) {
	// create a slice to hold the repositories
	repo, _, err := c.client.Repositories.Get(ctx, owner, name)
	if err != nil {
		return nil, fmt.Errorf("error getting repository: %w", err)
	}

	return repo, nil
}

// GetBranchProtection returns the branch protection for a given branch
func (c *RestClient) GetBranchProtection(ctx context.Context, owner string,
	repo_name string, branch_name string) (*github.Protection, error) {
	protection, _, err := c.client.Repositories.GetBranchProtection(ctx, owner, repo_name, branch_name)
	if err != nil {
		return nil, err
	}
	return protection, nil
}

// UpdateBranchProtection updates the branch protection for a given branch
func (c *RestClient) UpdateBranchProtection(
	ctx context.Context, owner, repo, branch string, preq *github.ProtectionRequest,
) error {
	_, _, err := c.client.Repositories.UpdateBranchProtection(ctx, owner, repo, branch, preq)
	return err
}

// GetAuthenticatedUser returns the authenticated user
func (c *RestClient) GetAuthenticatedUser(ctx context.Context) (*github.User, error) {
	user, _, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetBaseURL returns the base URL for the REST API.
func (c *RestClient) GetBaseURL() string {
	return c.client.BaseURL.String()
}

// NewRequest creates an API request. A relative URL can be provided in urlStr,
// which will be resolved to the BaseURL of the Client. Relative URLS should
// always be specified without a preceding slash. If specified, the value
// pointed to by body is JSON encoded and included as the request body.
func (c *RestClient) NewRequest(method, url string, body any) (*http.Request, error) {
	return c.client.NewRequest(method, url, body)
}

// Do sends an API request and returns the API response.
func (c *RestClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	var buf bytes.Buffer

	// The GitHub client closes the response body, so we need to capture it
	// in a buffer so that we can return it to the caller
	resp, err := c.client.Do(ctx, req, &buf)
	if err != nil {
		return nil, err
	}

	resp.Response.Body = io.NopCloser(&buf)

	return resp.Response, nil
}

// GetToken returns the token used to authenticate with the GitHub API
func (c *RestClient) GetToken() string {
	if c.token != "" {
		return c.token
	}
	return ""
}

// GetOwner returns the owner of the repository
func (c *RestClient) GetOwner() string {
	if c.owner != "" {
		return c.owner
	}
	return ""
}

// ListHooks lists all Hooks for the specified repository.
func (c *RestClient) ListHooks(ctx context.Context, owner, repo string) ([]*github.Hook, error) {
	list, _, err := c.client.Repositories.ListHooks(ctx, owner, repo, nil)
	return list, err
}

// DeleteHook deletes a specified Hook.
func (c *RestClient) DeleteHook(ctx context.Context, owner, repo string, id int64) error {
	_, err := c.client.Repositories.DeleteHook(ctx, owner, repo, id)
	return err
}

// CreateHook creates a new Hook.
func (c *RestClient) CreateHook(ctx context.Context, owner, repo string, hook *github.Hook) (*github.Hook, error) {
	h, _, err := c.client.Repositories.CreateHook(ctx, owner, repo, hook)
	return h, err
}

// CreateSecurityAdvisory creates a new security advisory
func (c *RestClient) CreateSecurityAdvisory(ctx context.Context, owner, repo, severity, summary, description string,
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
func (c *RestClient) CloseSecurityAdvisory(ctx context.Context, owner, repo, id string) error {
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error closing security advisory: %v", resp.Status)
	}
	return nil
}
