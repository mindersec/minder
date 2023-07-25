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
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v53/github"
)

// ListAllRepositories returns a list of all repositories for the authenticated user
// Two APIs are available, contigent on whether the token is for a user or an organization
func (c *RestClient) ListAllRepositories(ctx context.Context, isOrg bool, owner string) (RepositoryListResult, error) {
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
			return RepositoryListResult{}, err
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

	return RepositoryListResult{
		Repositories: allRepos,
	}, nil
}

// PackageListResult is a struct to hold the results of a package list
type PackageListResult struct {
	Packages []*github.Package
}

// ListAllPackages returns a list of all packages for the authenticated user
func (c *RestClient) ListAllPackages(ctx context.Context, isOrg bool, artifactType string,
	pageNumber int, itemsPerPage int) (PackageListResult, error) {
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
			artifacts, resp, err = c.client.Organizations.ListPackages(ctx, "", opt)
		} else {
			user, err1 := c.GetAuthenticatedUser(ctx)
			if err1 != nil {
				return PackageListResult{}, err1
			}
			artifacts, resp, err = c.client.Users.ListPackages(ctx, user.GetLogin(), opt)
		}
		if err != nil {
			return PackageListResult{}, err
		}

		allContainers = append(allContainers, artifacts...)

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return PackageListResult{Packages: allContainers}, nil
}

// GetPackageByName returns a single package for the authenticated user or for the org
func (c *RestClient) GetPackageByName(ctx context.Context, isOrg bool, package_type string,
	package_name string) (*github.Package, error) {
	var pkg *github.Package
	var err error

	if isOrg {
		pkg, _, err = c.client.Organizations.GetPackage(ctx, "", package_type, package_name)
		if err != nil {
			return nil, err
		}
	} else {
		user, err := c.GetAuthenticatedUser(ctx)
		if err != nil {
			return nil, err
		}

		pkg, _, err = c.client.Users.GetPackage(ctx, user.GetLogin(), package_type, package_name)
		if err != nil {
			return nil, err
		}
	}
	return pkg, nil
}

// GetPackageVersions returns a list of all package versions for the authenticated user or org
func (c *RestClient) GetPackageVersions(ctx context.Context, isOrg bool, package_type string,
	package_name string) ([]*github.PackageVersion, error) {
	var versions []*github.PackageVersion
	var err error
	state := "active"

	if isOrg {
		versions, _, err = c.client.Organizations.PackageGetAllVersions(ctx, "", package_type,
			package_name, &github.PackageListOptions{PackageType: &package_type, State: &state})
		if err != nil {
			return nil, err
		}
	} else {
		user, err := c.GetAuthenticatedUser(ctx)
		if err != nil {
			return nil, err
		}

		versions, _, err = c.client.Users.PackageGetAllVersions(ctx, user.GetLogin(), package_type,
			package_name, &github.PackageListOptions{PackageType: &package_type, State: &state})
		if err != nil {
			return nil, err
		}
	}

	return versions, nil
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

// CheckIfTokenIsForOrganization is to determine if the token is for a user or an organization
// TODO: There may be more efficient ways to do this, then calling the API,
// perhaps during the enrollment process
func (c *RestClient) CheckIfTokenIsForOrganization(ctx context.Context) (bool, error) {
	user, _, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return false, err
	}

	if *user.Type == "Organization" {
		return true, nil
	}

	return false, nil
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

// GetAuthenticatedUser returns the authenticated user
func (c *RestClient) GetAuthenticatedUser(ctx context.Context) (*github.User, error) {
	user, _, err := c.client.Users.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	return user, nil
}

// NewRequest creates an API request. A relative URL can be provided in urlStr,
// which will be resolved to the BaseURL of the Client. Relative URLS should
// always be specified without a preceding slash. If specified, the value
// pointed to by body is JSON encoded and included as the request body.
func (c *RestClient) NewRequest(method, url string, body interface{}, opts ...github.RequestOption) (*http.Request, error) {
	return c.client.NewRequest(method, url, body, opts...)
}

// Do sends an API request and returns the API response.
func (c *RestClient) Do(ctx context.Context, req *http.Request, v interface{}) (*github.Response, error) {
	return c.client.Do(ctx, req, v)
}
