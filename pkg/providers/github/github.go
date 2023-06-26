// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package providers contains the interfaces required to interact with the
// github GraphQL and REST APIs.
package github

import (
	"context"

	"github.com/google/go-github/v52/github"
	"github.com/shurcooL/graphql"
	"golang.org/x/oauth2"
)

// RepositoryInfo is the struct that contains repository information
type RepositoryInfo struct {
	Name           graphql.String
	Description    graphql.String
	StargazerCount graphql.Int
	ForkCount      graphql.Int
	CreatedAt      graphql.String
}

// RepositoryQuery is the struct that contains the GraphQL query for the repository
type RepositoryQuery struct {
	Repository RepositoryInfo `graphql:"repository(owner: $owner, name: $name)"`
}

// GithubClient is the interface that defines the methods required to interact
// with the github GraphQL and REST APIs. We use an interface here so that we
// can mock the github client for testing.
type GithubClient interface {
	RunQuery(ctx context.Context, query interface{}, variables map[string]interface{}) error
	GetGraphQLRepositoryInfo(ctx context.Context, owner string, name string) (*RepositoryInfo, error)
	GetRestAPIRepositoryInfo(ctx context.Context, owner string, name string) (*github.Repository, error)
}

// githubClientWrapper is the struct that implements graphql and rest API
// methods
type githubClientWrapper struct {
	client    *github.Client
	ghGraphQL *graphql.Client
}

// Client is the struct that contains the github client and implements the
// GithubClient interface
type Client struct {
	GithubClient GithubClient
}

// NewClient creates a new github client, this should be initialized once per
// application and passed around as a dependency. It should be initialized with
// a valid github token.
// e.g.
// client, err := providers.NewClient(ctx, token.AccessToken)
// It can then be used to interact with the github GraphQL and REST APIs.
// e.g. client.GithubClient.GetGraphQLRepositoryInfo(ctx, owner, name)
func NewClient(ctx context.Context, token string) (*Client, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	ghClient := github.NewClient(tc)

	ghGraphQL := graphql.NewClient("https://api.github.com/graphql", tc)

	wrapper := &githubClientWrapper{
		client:    ghClient,
		ghGraphQL: ghGraphQL,
	}

	return &Client{
		GithubClient: wrapper,
	}, nil
}

// RunQuery is a wrapper around the graphql client RunQuery method that
// executes the query and returns the result
func (gcw *githubClientWrapper) RunQuery(ctx context.Context, query interface{}, variables map[string]interface{}) error {
	return gcw.ghGraphQL.Query(ctx, query, variables)
}

// GetGraphQLRepositoryInfo returns the repository information for the given
// owner and name using the github GraphQL API
// Note: These can be removed, but I left them in for now to show how the
// GraphQL and REST APIs can be used
func (gcw *githubClientWrapper) GetGraphQLRepositoryInfo(ctx context.Context,
	owner string,
	name string) (*RepositoryInfo, error) {
	var query RepositoryQuery
	variables := map[string]interface{}{
		"owner": graphql.String(owner),
		"name":  graphql.String(name),
	}
	err := gcw.RunQuery(ctx, &query, variables)
	if err != nil {
		return nil, err
	}
	return &query.Repository, nil
}

// GetRestAPIRepositoryInfo returns the repository information for the given
// owner and name using the github Rest API
// Note: These can be removed, but I left them in for now to show how the
// GraphQL and REST APIs can be used
func (gcw *githubClientWrapper) GetRestAPIRepositoryInfo(ctx context.Context,
	owner string,
	name string) (*github.Repository, error) {
	repo, _, err := gcw.client.Repositories.Get(ctx, owner, name)
	if err != nil {
		return nil, err
	}
	return repo, nil
}
