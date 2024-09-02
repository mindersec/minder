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

type propertyOrigin struct {
	keys    []string
	wrapper GhPropertyWrapper
}

var repoOperationalProperties = []string{
	RepoPropertyHookId,
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

func getRepoWrapper(ctx context.Context, ghCli *go_github.Client, name string) (map[string]any, error) {
	// TODO: this should be a provider interface, even if private
	slice := strings.Split(name, "/")
	if len(slice) != 2 {
		return nil, errors.New("invalid name")
	}

	repo, result, err := ghCli.Repositories.Get(ctx, slice[0], slice[1])
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

	//repoProps[properties.PropertyName], err = getEntityName(minderv1.Entity_ENTITY_REPOSITORIES, props)
	repoProps[properties.PropertyName] = fmt.Sprintf("%s/%s", repo.GetOwner().GetLogin(), repo.GetName())

	return repoProps, nil
}

// RepositoryFetcher is a property fetcher for github repositories
type RepositoryFetcher struct {
	propertyOrigins       []propertyOrigin
	operationalProperties []string
}

// NewRepositoryFetcher creates a new RepositoryFetcher
func NewRepositoryFetcher() *RepositoryFetcher {
	return &RepositoryFetcher{
		propertyOrigins:       repoPropertyDefinitions,
		operationalProperties: repoOperationalProperties,
	}
}

// OperationalProperties returns the operational properties for the repository
func (_ *RepositoryFetcher) OperationalProperties() []string {
	return []string{
		RepoPropertyHookId,
		RepoPropertyHookUrl,
	}
}

// WrapperForProperty returns the property wrapper for the given property key
func (r *RepositoryFetcher) WrapperForProperty(propertyKey string) GhPropertyWrapper {
	for _, po := range r.propertyOrigins {
		for _, k := range po.keys {
			if k == propertyKey {
				return po.wrapper
			}
		}
	}

	return nil
}

// AllPropertyWrappers returns all property wrappers for the repository
func (r *RepositoryFetcher) AllPropertyWrappers() []GhPropertyWrapper {
	wrappers := make([]GhPropertyWrapper, 0, len(r.propertyOrigins))
	for _, po := range r.propertyOrigins {
		wrappers = append(wrappers, po.wrapper)
	}
	return wrappers
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
