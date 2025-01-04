// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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

	"github.com/mindersec/minder/internal/entities/properties"
	pbinternal "github.com/mindersec/minder/internal/proto"
	v1 "github.com/mindersec/minder/pkg/providers/v1"
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
	// PullPropertyAuthorLogin is the login of the author of the pull request
	PullPropertyAuthorLogin = "github/pull_author_login"
	// PullPropertyAction is an operational property that represents the action that was taken on the pull request
	PullPropertyAction = "github/pull_action"
	// PullPropertyBaseCloneURL is the URL used to clone the repository
	PullPropertyBaseCloneURL = "github/clone_url"
	// PullPropertyTargetCloneURL is the URL used to clone the target repository
	PullPropertyTargetCloneURL = "github/target_url"
	// PullPropertyBaseRef is the base ref of the pull request
	PullPropertyBaseRef = "github/base_ref"
	// PullPropertyTargetRef is the target ref of the pull request
	PullPropertyTargetRef = "github/pull_ref"
)

var prPropertyDefinitions = []propertyOrigin{
	{
		keys: []string{
			// general entity
			properties.PropertyName,
			properties.PropertyUpstreamID,
			properties.PullRequestCommitSHA,
			properties.PullRequestBaseCloneURL,
			properties.PullRequestBaseDefaultBranch,
			properties.PullRequestTargetCloneURL,
			properties.PullRequestTargetBranch,
			properties.PullRequestUpstreamURL,
			// github-specific
			PullPropertyURL,
			PullPropertyNumber,
			PullPropertySha,
			PullPropertyRepoOwner,
			PullPropertyRepoName,
			PullPropertyAuthorID,
			PullPropertyAuthorLogin,
			PullPropertyAction,
			PullPropertyBaseCloneURL,
			PullPropertyTargetCloneURL,
			PullPropertyBaseRef,
			PullPropertyTargetRef,
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
		properties.PropertyUpstreamID:           properties.NumericalValueToUpstreamID(prReply.GetID()),
		properties.PropertyName:                 fmt.Sprintf("%s/%s/%d", owner, name, intId),
		properties.PullRequestCommitSHA:         prReply.GetHead().GetSHA(),
		properties.PullRequestBaseCloneURL:      prReply.GetBase().GetRepo().GetCloneURL(),
		properties.PullRequestBaseDefaultBranch: prReply.GetBase().GetRepo().GetDefaultBranch(),
		properties.PullRequestTargetCloneURL:    prReply.GetHead().GetRepo().GetCloneURL(),
		properties.PullRequestTargetBranch:      prReply.GetHead().GetRef(),
		properties.PullRequestUpstreamURL:       prReply.GetHTMLURL(),
		// github-specific
		PullPropertyURL: prReply.GetHTMLURL(),
		// our proto representation uses int64 for the number but GH uses int
		PullPropertyNumber:         int64(prReply.GetNumber()),
		PullPropertySha:            prReply.GetHead().GetSHA(),
		PullPropertyRepoOwner:      owner,
		PullPropertyRepoName:       name,
		PullPropertyAuthorID:       prReply.GetUser().GetID(),
		PullPropertyAuthorLogin:    prReply.GetUser().GetLogin(),
		PullPropertyBaseCloneURL:   prReply.GetBase().GetRepo().GetCloneURL(),
		PullPropertyTargetCloneURL: prReply.GetHead().GetRepo().GetCloneURL(),
		PullPropertyBaseRef:        prReply.GetBase().GetRef(),
		PullPropertyTargetRef:      prReply.GetHead().GetRef(),
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
func PullRequestV1FromProperties(props *properties.Properties) (*pbinternal.PullRequest, error) {
	return &pbinternal.PullRequest{
		Url:            props.GetProperty(PullPropertyURL).GetString(),
		CommitSha:      props.GetProperty(PullPropertySha).GetString(),
		TargetRef:      props.GetProperty(PullPropertyTargetRef).GetString(),
		BaseRef:        props.GetProperty(PullPropertyBaseRef).GetString(),
		Number:         props.GetProperty(PullPropertyNumber).GetInt64(),
		RepoOwner:      props.GetProperty(PullPropertyRepoOwner).GetString(),
		RepoName:       props.GetProperty(PullPropertyRepoName).GetString(),
		AuthorId:       props.GetProperty(PullPropertyAuthorID).GetInt64(),
		Action:         props.GetProperty(PullPropertyAction).GetString(),
		BaseCloneUrl:   props.GetProperty(PullPropertyBaseCloneURL).GetString(),
		TargetCloneUrl: props.GetProperty(PullPropertyTargetCloneURL).GetString(),
		Properties:     props.ToProtoStruct(),
	}, nil
}
