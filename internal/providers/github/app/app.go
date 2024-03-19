// Copyright 2024 Stacklok, Inc
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

// Package app provides the GitHub App specific operations
package app

import (
	"context"
	"fmt"

	"github.com/google/go-github/v56/github"

	github2 "github.com/stacklok/minder/internal/providers/github"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// GitHubAppDelegate is the struct that contains the GitHub App specific operations
type GitHubAppDelegate struct {
	client     *github.Client
	credential provifv1.GitHubCredential
}

// Ensure that the GitHubAppDelegate client implements the GitHub Delegate interface
var _ github2.Delegate = (*GitHubAppDelegate)(nil)

// GetCredential returns the GitHub App installation credential
func (g *GitHubAppDelegate) GetCredential() provifv1.GitHubCredential {
	return g.credential
}

// ListUserRepositories returns a list of repositories for the owner
func (g *GitHubAppDelegate) ListUserRepositories(ctx context.Context, owner string) ([]*minderv1.Repository, error) {
	repos, err := g.ListAllRepositories(ctx, false, owner)
	if err != nil {
		return nil, err
	}

	return github2.ConvertRepositories(repos), nil
}

// ListOrganizationRepositories returns a list of repositories for the organization
func (g *GitHubAppDelegate) ListOrganizationRepositories(
	ctx context.Context,
	owner string,
) ([]*minderv1.Repository, error) {
	repos, err := g.ListAllRepositories(ctx, true, owner)
	if err != nil {
		return nil, err
	}

	return github2.ConvertRepositories(repos), nil
}

// ListAllRepositories returns a list of all repositories accessible to the GitHub App installation
func (g *GitHubAppDelegate) ListAllRepositories(ctx context.Context, _ bool, _ string) ([]*github.Repository, error) {
	listOpt := &github.ListOptions{
		PerPage: 100,
	}

	// create a slice to hold the repositories
	var allRepos []*github.Repository
	for {
		var repos *github.ListRepositories
		var resp *github.Response
		var err error

		repos, resp, err = g.client.Apps.ListRepos(ctx, listOpt)

		if err != nil {
			return allRepos, err
		}
		allRepos = append(allRepos, repos.Repositories...)
		if resp.NextPage == 0 {
			break
		}

		listOpt.Page = resp.NextPage
	}

	return allRepos, nil
}

// GetUserId returns the user id for the GitHub App user
func (_ *GitHubAppDelegate) GetUserId(_ context.Context) (int64, error) {
	// TODO: The GitHub App user will have a unique ID which we can find once we create the App.
	// note: this is different from the App ID
	panic("unimplemented")
}

// GetName returns the username for the GitHub App user
func (_ *GitHubAppDelegate) GetName(_ context.Context) (string, error) {
	// TODO: The GitHub App username is the name of the GitHub App, appended with [bot], e.g minder[bot]
	panic("unimplemented")
}

// GetLogin returns the username for the GitHub App user
func (_ *GitHubAppDelegate) GetLogin(_ context.Context) (string, error) {
	// TODO: The GitHub App username is the name of the GitHub App, appended with [bot], e.g minder[bot]
	panic("unimplemented")
}

// GetPrimaryEmail returns the email for the GitHub App user
func (g *GitHubAppDelegate) GetPrimaryEmail(ctx context.Context) (string, error) {
	userId, err := g.GetUserId(ctx)
	if err != nil {
		return "", fmt.Errorf("error getting user ID: %v", err)
	}
	return fmt.Sprintf("%d+github-actions[bot]@users.noreply.github.com", userId), nil
}
