// // Copyright 2023 Stacklok, Inc
// //
// // Licensed under the Apache License, Version 2.0 (the "License");
// // you may not use this file except in compliance with the License.
// // You may obtain a copy of the License at
// //
// //	http://www.apache.org/licenses/LICENSE-2.0
// //
// // Unless required by applicable law or agreed to in writing, software
// // distributed under the License is distributed on an "AS IS" BASIS,
// // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// // See the License for the specific language governing permissions and
// // limitations under the License.

// Package github provides a client for interacting with the GitHub API
package github

import (
	"context"
	"fmt"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"

	"github.com/stacklok/mediator/pkg/providers"
)

// NOTE: This file is for stubbing out client code for GraphQL
// It is currently a placeholder.

// Query implements the RepoProvider interface
func (c *GraphQLClient) Query(ctx context.Context, queryObj any, variables map[string]any) error {
	return c.client.Query(ctx, queryObj, variables)
}

type pageInfo struct {
	EndCursor   githubv4.String
	HasNextPage bool
}

type repoData struct {
	Name  string
	Owner struct {
		Login string
	}
	DatabaseId int
	IsPrivate  bool
	IsFork     bool
}

func (r repoData) toRepoMetadata() *providers.RepositoryMetadata {
	return &providers.RepositoryMetadata{
		Provider: "graphql",
		Id: &providers.RepoId{
			Name:   r.Name,
			Parent: r.Owner.Login,
			Id:     int32(r.DatabaseId),
		},
		IsPrivate: r.IsPrivate,
		IsFork:    r.IsFork,
	}
}

// ListCallerRepositories implements the RepoProvider interface
func (c *GraphQLClient) ListCallerRepositories(ctx context.Context, includeForks bool) ([]*providers.RepositoryMetadata, error) {
	var query struct {
		Viewer struct {
			Repositories struct {
				Nodes    []repoData
				PageInfo pageInfo
			} `graphql:"repositories(first: 100, after: $repoCursor, ownerAffiliations:[OWNER,COLLABORATOR,ORGANIZATION_MEMBER], isFork: $includeForks)"`
		} // TODO: do we want to only list non-forks?
	}

	forksVar := githubv4.NewBoolean(false)
	if includeForks {
		// a nil value selects both forks and non-forked repos
		forksVar = (*githubv4.Boolean)(nil)
	}
	repoQueryVariables := map[string]interface{}{
		"includeForks": forksVar,
		"repoCursor":   (*githubv4.String)(nil),
	}

	result := make([]*providers.RepositoryMetadata, 0)
	for {
		err := c.client.Query(ctx, &query, repoQueryVariables)
		if err != nil {
			return nil, err
		}
		for _, r := range query.Viewer.Repositories.Nodes {
			result = append(result, r.toRepoMetadata())
		}
		if !query.Viewer.Repositories.PageInfo.HasNextPage {
			break
		}
		repoQueryVariables["repoCursor"] = githubv4.NewString(query.Viewer.Repositories.PageInfo.EndCursor)
	}

	return result, nil
}

// ListRepositories implements the RepoProvider interface
func (c *GraphQLClient) ListRepositories(ctx context.Context, owner string) ([]*providers.RepositoryMetadata, error) {
	var query struct {
		Owner struct {
			Repositories struct {
				Nodes    []repoData
				PageInfo pageInfo
			} `graphql:"repositories(first: 100, after: $repoCursor, ownerAffiliations:[OWNER])"`
		} `graphql:"repositoryOwner(login: $owner)"`
	}

	repoQueryVariables := map[string]interface{}{
		"owner":      githubv4.String(owner),
		"repoCursor": (*githubv4.String)(nil),
	}

	result := make([]*providers.RepositoryMetadata, 0)
	for {
		err := c.client.Query(ctx, &query, repoQueryVariables)
		if err != nil {
			return nil, err
		}
		for _, r := range query.Owner.Repositories.Nodes {
			result = append(result, r.toRepoMetadata())
		}
		if !query.Owner.Repositories.PageInfo.HasNextPage {
			break
		}
		repoQueryVariables["repoCursor"] = githubv4.NewString(query.Owner.Repositories.PageInfo.EndCursor)
	}
	return result, nil
}

func init() {
	providers.RegisterProvider("github",
		func(ctx context.Context, provider string, token oauth2.Token) (providers.RepoProvider, error) {
			if provider != "github" {
				return nil, fmt.Errorf("provider must be github")
			}
			return NewGraphQLClient(ctx, GitHubConfig{
				Token: token.AccessToken,
			})
		})
}
