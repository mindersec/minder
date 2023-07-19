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
	"net/http"
	"net/url"

	"github.com/google/go-github/v53/github"
	"github.com/shurcooL/graphql"
	"golang.org/x/oauth2"
)

// GitHubConfig is the struct that contains the configuration for the GitHub client
// Token: is the GitHub API token retrieved from the provider_access_tokens table
// in the database
// Endpoint: is the GitHub API endpoint
// If using the public GitHub API, Endpoint can be left blank
// disable revive linting for this struct as there is nothing wrong with the
// naming convention
type GitHubConfig struct { //revive:disable-line:exported
	Token    string
	Endpoint string
}

// Github is the string that represents the GitHub provider
const Github = "github"

// RepositoryListResult is a struct that contains the information about a GitHub repository
type RepositoryListResult struct {
	Repositories []*github.Repository
}

// RestAPI is the interface for interacting with the GitHub REST API
// Add methods here for interacting with the GitHub Rest API
// e.g. GetRepositoryRestInfo(ctx context.Context, owner string, name string) (*RepositoryInfo, error)
type RestAPI interface {
	GetAuthenticatedUser(context.Context) (*github.User, error)
	GetRepository(context.Context, string, string) (*github.Repository, error)
	ListAllRepositories(context.Context, bool, string) (RepositoryListResult, error)
	GetBranchProtection(context.Context, string, string, string) (*github.Protection, error)
	ListAllPackages(context.Context, bool, string) (PackageListResult, error)

	// NewRequest allows for building raw and custom requests
	NewRequest(method, urlStr string, body any, opts ...github.RequestOption) (*http.Request, error)
	Do(ctx context.Context, req *http.Request, v any) (*github.Response, error)
}

// GraphQLAPI is the interface for interacting with the GitHub GraphQL API
// Add methods here for interacting with the GitHub GraphQL API
// e.g. GetRepositoryGraphInfo(ctx context.Context, owner string, name string) (*RepositoryInfo, error)
type GraphQLAPI interface {
	RunQuery(ctx context.Context, query interface{}, variables map[string]interface{}) error
}

// RestClient is the struct that contains the GitHub REST API client
type RestClient struct {
	client *github.Client
}

// GraphQLClient is the struct that contains the GitHub GraphQL API client
type GraphQLClient struct {
	client *graphql.Client
}

// NewRestClient creates a new GitHub REST API client
// BaseURL defaults to the public GitHub API, if needing to use a customer domain
// endpoint (as is the case with GitHub Enterprise), set the Endpoint field in
// the GitHubConfig struct
func NewRestClient(ctx context.Context, config GitHubConfig) (RestAPI, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.Token},
	)
	tc := oauth2.NewClient(ctx, ts)

	ghClient := github.NewClient(tc)

	if config.Endpoint != "" {
		parsedURL, err := url.Parse(config.Endpoint)
		if err != nil {
			return nil, err
		}
		ghClient.BaseURL = parsedURL
	}

	return &RestClient{
		client: ghClient,
	}, nil
}

// NewGraphQLClient creates a new GitHub GraphQL API client
func NewGraphQLClient(ctx context.Context, config GitHubConfig) (GraphQLAPI, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.Token},
	)
	tc := oauth2.NewClient(ctx, ts)

	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = "https://api.github.com/graphql"
	}

	ghGraphQL := graphql.NewClient(endpoint, tc)

	return &GraphQLClient{
		client: ghGraphQL,
	}, nil
}

// RunQuery executes a GraphQL query
func (gc *GraphQLClient) RunQuery(ctx context.Context, query interface{}, variables map[string]interface{}) error {
	return gc.client.Query(ctx, query, variables)
}
