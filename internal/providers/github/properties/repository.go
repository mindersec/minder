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

package properties

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	go_github "github.com/google/go-github/v63/github"

	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	// RepoPropertyId represents the github repository ID (numerical)
	RepoPropertyId = "github/repo_id"
	// RepoPropertyName represents the github repository name
	RepoPropertyName = "github/repo_name"
	// RepoPropertyOwner represents the github repository owner
	RepoPropertyOwner = "github/repo_owner"
	// RepoPropertyDeployURL represents the github repository deployment URL
	RepoPropertyDeployURL = "github/deploy_url"
	// RepoPropertyCloneURL represents the github repository clone URL
	RepoPropertyCloneURL = "github/clone_url"
	// RepoPropertyDefaultBranch represents the github repository default branch
	RepoPropertyDefaultBranch = "github/default_branch"
	// RepoPropertyLicense represents the github repository license
	RepoPropertyLicense = "github/license"

	// RepoPropertyHookId represents the github repository hook ID
	RepoPropertyHookId = "github/hook_id"
	// RepoPropertyHookUrl represents the github repository hook URL
	RepoPropertyHookUrl = "github/hook_url"
	// RepoPropertyHookName represents the github repository hook name
	RepoPropertyHookName = "github/hook_name"
	// RepoPropertyHookType represents the github repository hook type
	RepoPropertyHookType = "github/hook_type"
	// RepoPropertyHookUiid represents the github repository hook UIID
	RepoPropertyHookUiid = "github/hook_uiid"
)

var repoOperationalProperties = []string{
	RepoPropertyHookId,
	RepoPropertyHookUrl,
}

var repoPropertyDefinitions = []propertyOrigin{
	{
		keys: []string{
			// general entity
			properties.PropertyName,
			properties.PropertyUpstreamID,
			// general repo
			properties.RepoPropertyIsPrivate,
			properties.RepoPropertyIsArchived,
			properties.RepoPropertyIsFork,
			// github-specific
			RepoPropertyId,
			RepoPropertyName,
			RepoPropertyOwner,
			RepoPropertyDeployURL,
			RepoPropertyCloneURL,
			RepoPropertyDefaultBranch,
			RepoPropertyLicense,
		},
		wrapper: getRepoWrapper,
	},
}

func getRepoWrapper(
	ctx context.Context, ghCli *go_github.Client, getByProps *properties.Properties,
) (map[string]any, error) {

	name, owner, err := getNameOwnerFromProps(getByProps)
	if err != nil {
		return nil, fmt.Errorf("error getting name and owner from properties: %w", err)
	}

	repo, result, err := ghCli.Repositories.Get(ctx, owner, name)
	if err != nil {
		if result != nil && result.StatusCode == http.StatusNotFound {
			return nil, v1.ErrEntityNotFound
		}
		return nil, err
	}

	repoProps := map[string]any{
		// general entity
		properties.PropertyUpstreamID: fmt.Sprintf("%d", repo.GetID()),
		// general repo
		properties.RepoPropertyIsPrivate:  repo.GetPrivate(),
		properties.RepoPropertyIsArchived: repo.GetArchived(),
		properties.RepoPropertyIsFork:     repo.GetFork(),
		// github-specific
		RepoPropertyId:            repo.GetID(),
		RepoPropertyName:          repo.GetName(),
		RepoPropertyOwner:         repo.GetOwner().GetLogin(),
		RepoPropertyDeployURL:     repo.GetDeploymentsURL(),
		RepoPropertyCloneURL:      repo.GetCloneURL(),
		RepoPropertyDefaultBranch: repo.GetDefaultBranch(),
		RepoPropertyLicense:       repo.GetLicense().GetSPDXID(),
	}

	repoProps[properties.PropertyName] = fmt.Sprintf("%s/%s", repo.GetOwner().GetLogin(), repo.GetName())

	return repoProps, nil
}

func getNameOwnerFromProps(props *properties.Properties) (string, string, error) {
	repoNameP := props.GetProperty(RepoPropertyName)
	repoOwnerP := props.GetProperty(RepoPropertyOwner)
	if repoNameP != nil && repoOwnerP != nil {
		return repoNameP.GetString(), repoOwnerP.GetString(), nil
	}

	repoNameP = props.GetProperty(properties.PropertyName)
	if repoNameP != nil {
		slice := strings.Split(repoNameP.GetString(), "/")
		if len(slice) != 2 {
			return "", "", errors.New("invalid repo name")
		}

		return slice[1], slice[0], nil
	}

	return "", "", errors.New("missing required properties, either repo-name and repo-owner or name")
}

// RepositoryFetcher is a property fetcher for github repositories
type RepositoryFetcher struct {
	propertyFetcherBase
}

// NewRepositoryFetcher creates a new RepositoryFetcher
func NewRepositoryFetcher() *RepositoryFetcher {
	return &RepositoryFetcher{
		propertyFetcherBase: propertyFetcherBase{
			operationalProperties: repoOperationalProperties,
			propertyOrigins:       repoPropertyDefinitions,
		},
	}
}

// GetName returns the name of the repository
func (_ *RepositoryFetcher) GetName(props *properties.Properties) (string, error) {
	repoNameP := props.GetProperty(RepoPropertyName)
	repoOwnerP := props.GetProperty(RepoPropertyOwner)

	if repoNameP == nil || repoOwnerP == nil {
		return "", errors.New("missing required properties")
	}

	repoName := repoNameP.GetString()
	if repoName == "" {
		return "", errors.New("missing required repo-name property value")
	}

	repoOwner := repoOwnerP.GetString()
	if repoOwner == "" {
		return "", errors.New("missing required repo-owner property value")
	}

	return fmt.Sprintf("%s/%s", repoOwner, repoName), nil
}

// RepoV1FromProperties creates a minderv1.Repository from a properties.Properties
func RepoV1FromProperties(repoProperties *properties.Properties) (*minderv1.Repository, error) {
	name, err := repoProperties.GetProperty(RepoPropertyName).AsString()
	if err != nil {
		return nil, fmt.Errorf("error fetching name property: %w", err)
	}

	owner, err := repoProperties.GetProperty(RepoPropertyOwner).AsString()
	if err != nil {
		return nil, fmt.Errorf("error fetching owner property: %w", err)
	}

	repoId, err := repoProperties.GetProperty(RepoPropertyId).AsInt64()
	if err != nil {
		return nil, fmt.Errorf("error fetching repo_id property: %w", err)
	}

	isPrivate, err := repoProperties.GetProperty(properties.RepoPropertyIsPrivate).AsBool()
	if err != nil {
		return nil, fmt.Errorf("error fetching is_private property: %w", err)
	}

	isFork, err := repoProperties.GetProperty(properties.RepoPropertyIsFork).AsBool()
	if err != nil {
		return nil, fmt.Errorf("error fetching is_fork property: %w", err)
	}

	pbRepo := &minderv1.Repository{
		Name:          name,
		Owner:         owner,
		RepoId:        repoId,
		HookId:        repoProperties.GetProperty(RepoPropertyHookId).GetInt64(),
		HookUrl:       repoProperties.GetProperty(RepoPropertyHookUrl).GetString(),
		DeployUrl:     repoProperties.GetProperty(RepoPropertyDeployURL).GetString(),
		CloneUrl:      repoProperties.GetProperty(RepoPropertyCloneURL).GetString(),
		HookType:      repoProperties.GetProperty(RepoPropertyHookType).GetString(),
		HookName:      repoProperties.GetProperty(RepoPropertyHookName).GetString(),
		HookUuid:      repoProperties.GetProperty(RepoPropertyHookUiid).GetString(),
		IsPrivate:     isPrivate,
		IsFork:        isFork,
		DefaultBranch: repoProperties.GetProperty(RepoPropertyDefaultBranch).GetString(),
		License:       repoProperties.GetProperty(RepoPropertyLicense).GetString(),
	}

	return pbRepo, nil
}
