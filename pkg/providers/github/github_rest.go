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

type PackageResult struct {
	Package     *github.Package
	LastVersion *github.PackageVersion
}

// PackageListResult is a struct to hold the results of a package list
type PackageListResult struct {
	Packages []*PackageResult
}

// ListAllPackages returns a list of all packages of type container for the authenticated user
func (c *RestClient) ListAllContainers(ctx context.Context, isOrg bool) (PackageListResult, error) {
	opt := &github.PackageListOptions{
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 100,
		},
		PackageType: github.String("container"),
	}

	optVersion := &github.PackageListOptions{
		ListOptions: github.ListOptions{
			Page:    1,
			PerPage: 1,
		},
	}

	user, err := c.GetAuthenticatedUser(ctx)
	if err != nil {
		return PackageListResult{}, err
	}

	// create a slice to hold the containers
	var allContainers []*PackageResult
	for {
		var containers []*github.Package
		var resp *github.Response
		var err error

		if isOrg {
			containers, resp, err = c.client.Organizations.ListPackages(ctx, "", opt)
		} else {
			containers, resp, err = c.client.Users.ListPackages(ctx, *user.Login, opt)
		}

		if err != nil {
			return PackageListResult{}, err
		}

		// read last version of each container
		for _, container := range containers {
			// list all versions of the container
			var versions []*github.PackageVersion
			if isOrg {
				versions, _, err = c.client.Organizations.PackageGetAllVersions(ctx, "", *container.PackageType, *container.Name, optVersion)
			} else {
				versions, _, err = c.client.Users.PackageGetAllVersions(ctx, "", *container.PackageType, *container.Name, optVersion)
			}
			if err != nil {
				return PackageListResult{}, err
			}

			if len(versions) > 0 {
				allContainers = append(allContainers, &PackageResult{
					Package:     container,
					LastVersion: versions[0],
				})
			} else {
				allContainers = append(allContainers, &PackageResult{
					Package:     container,
					LastVersion: nil,
				})
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return PackageListResult{Packages: allContainers}, nil
}

// GetRepository returns a single repository for the authenticated user
func (c *RestClient) GetRepository(ctx context.Context, owner, name string) (*github.Repository, error) {
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
