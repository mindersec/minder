//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package properties provides utility functions for fetching and managing properties
package properties

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	go_github "github.com/google/go-github/v63/github"

	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// PullPropertyURL is the URL of the pull request
	PullPropertyURL = "github/pull_url"
	// PullPropertyNumber is the number of the pull request
	PullPropertyNumber = "github/pull_number"
	// PullPropertySha is the sha of the pull request
	PullPropertySha = "github/pull_sha"
	// PullPropertyRepoOwner is the owner of the repository
	PullPropertyRepoOwner = "github/repo_owner"
	// PullPropertyRepoName is the name of the repository
	PullPropertyRepoName = "github/repo_name"
	// PullPropertyAuthorID is the ID of the author of the pull request
	PullPropertyAuthorID = "github/pull_author_id"
	// PullPropertyAction is an operational property that represents the action that was taken on the pull request
	PullPropertyAction = "github/pull_action"
)

var prPropertyDefinitions = []propertyOrigin{
	{
		keys: []string{
			// general entity
			properties.PropertyName,
			properties.PropertyUpstreamID,
			// github-specific
			PullPropertyURL,
			PullPropertyNumber,
			PullPropertySha,
			PullPropertyRepoOwner,
			PullPropertyRepoName,
			PullPropertyAuthorID,
			PullPropertyAction,
		},
		wrapper: getPrWrapper,
	},
}

var prOperationalProperties = []string{
	PullPropertyAction,
}

// PullRequestFetcher is a property fetcher for github repositories
type PullRequestFetcher struct {
	propertyFetcherBase
}

// NewPullRequestFetcher creates a new PullRequestFetcher
func NewPullRequestFetcher() *PullRequestFetcher {
	return &PullRequestFetcher{
		propertyFetcherBase: propertyFetcherBase{
			operationalProperties: prOperationalProperties,
			propertyOrigins:       prPropertyDefinitions,
		},
	}
}

// GetName returns the name of the pull request
func (_ *PullRequestFetcher) GetName(props *properties.Properties) (string, error) {
	prOwner, err := props.GetProperty(PullPropertyRepoOwner).AsString()
	if err != nil {
		return "", fmt.Errorf("error fetching pr owner property: %w", err)
	}

	prName, err := props.GetProperty(PullPropertyRepoName).AsString()
	if err != nil {
		return "", fmt.Errorf("error fetching pr name property: %w", err)
	}

	prID, err := props.GetProperty(PullPropertyNumber).AsInt64()
	if err != nil {
		return "", fmt.Errorf("error fetching pr id property: %w", err)
	}

	return fmt.Sprintf("%s/%s/%d", prOwner, prName, prID), nil
}

func parsePrName(input string) (string, string, int64, error) {
	parts := strings.Split(input, "/")
	if len(parts) != 3 {
		return "", "", 0, fmt.Errorf("invalid input format")
	}

	repoOwner := parts[0]
	repoName := parts[1]
	prNumber, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid prNumber: %w", err)
	}

	return repoOwner, repoName, prNumber, nil
}

func getPrWrapper(
	ctx context.Context, ghCli *go_github.Client, isOrg bool, getByProps *properties.Properties,
) (map[string]any, error) {
	_ = isOrg

	owner, name, id64, err := getPrWrapperAttrsFromProps(getByProps)
	if err != nil {
		return nil, fmt.Errorf("error getting pr wrapper attributes: %w", err)
	}

	if id64 > math.MaxInt {
		return nil, fmt.Errorf("pr number is too large")
	}
	intId := int(id64)

	prReply, result, err := ghCli.PullRequests.Get(ctx, owner, name, intId)
	if err != nil {
		if result != nil && result.StatusCode == http.StatusNotFound {
			return nil, v1.ErrEntityNotFound
		}
		return nil, err
	}

	prProps := map[string]any{
		// general entity
		properties.PropertyUpstreamID: prReply.GetID(),
		properties.PropertyName:       fmt.Sprintf("%s/%s/%d", owner, name, intId),
		// github-specific
		PullPropertyURL: prReply.GetHTMLURL(),
		// our proto representation uses int64 for the number but GH uses int
		PullPropertyNumber:    int64(prReply.GetNumber()),
		PullPropertySha:       prReply.GetHead().GetSHA(),
		PullPropertyRepoOwner: owner,
		PullPropertyRepoName:  name,
		PullPropertyAuthorID:  prReply.GetUser().GetID(),
	}

	return prProps, nil
}

func getPrWrapperAttrsFromProps(props *properties.Properties) (string, string, int64, error) {
	repoOwnerP := props.GetProperty(PullPropertyRepoOwner)
	repoNameP := props.GetProperty(PullPropertyRepoName)
	repoIdP := props.GetProperty(PullPropertyNumber)
	if repoOwnerP != nil && repoNameP != nil && repoIdP != nil {
		return repoOwnerP.GetString(), repoNameP.GetString(), repoIdP.GetInt64(), nil
	}

	prNameP := props.GetProperty(properties.PropertyName)
	if prNameP != nil {
		return parsePrName(prNameP.GetString())
	}

	return "", "", 0, fmt.Errorf("missing required properties")
}

// PullRequestV1FromProperties creates a PullRequestV1 from a properties object
func PullRequestV1FromProperties(props *properties.Properties) (*minderv1.PullRequest, error) {
	return &minderv1.PullRequest{
		Url:       props.GetProperty(PullPropertyURL).GetString(),
		CommitSha: props.GetProperty(PullPropertySha).GetString(),
		Number:    props.GetProperty(PullPropertyNumber).GetInt64(),
		RepoOwner: props.GetProperty(PullPropertyRepoOwner).GetString(),
		RepoName:  props.GetProperty(PullPropertyRepoName).GetString(),
		AuthorId:  props.GetProperty(PullPropertyAuthorID).GetInt64(),
		Action:    props.GetProperty(PullPropertyAction).GetString(),
	}, nil
}
