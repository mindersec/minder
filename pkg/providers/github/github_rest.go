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

	"github.com/google/go-github/v53/github"
)

// ListAllRepositories returns a list of all repositories for the authenticated user
// Two APIs are available, contigent on whether the token is for a user or an organization
func (c *RestClient) ListAllRepositories(ctx context.Context, isOrg bool) (RepositoryListResult, error) {
	opt := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
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
			repos, resp, err = c.client.Repositories.ListByOrg(ctx, "", orgOpt)
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

// ListAllRepositories returns a list of all repositories for the authenticated user
// Two APIs are available, contigent on whether the token is for a user or an organization
func (c *RestClient) GetBranchProtection(ctx context.Context, owner string, repo_name string, branch_name string) (github.Protection, error) {
	protection, _, err := c.client.Repositories.GetBranchProtection(ctx, owner, repo_name, branch_name)
	if err != nil {
		return github.Protection{}, err
	}
	return *protection, nil
}
