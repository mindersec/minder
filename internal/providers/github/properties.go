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

package github

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
)

type propertyWrapper func(ctx context.Context, ghCli *GitHub, name string) (map[string]any, error)

type propertyOrigin struct {
	keys    []string
	wrapper propertyWrapper
}

type propertyFetcher struct {
	ghCli *GitHub

	propertyOrigins []propertyOrigin
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

func newRepoPropertyFetcher(ghCli *GitHub) *propertyFetcher {
	return &propertyFetcher{
		ghCli:           ghCli,
		propertyOrigins: repoPropertyDefinitions,
	}
}

func newPropertyFetcher(ghCli *GitHub, entType minderv1.Entity) *propertyFetcher {
	if entType != minderv1.Entity_ENTITY_REPOSITORIES {
		return nil
	}

	return newRepoPropertyFetcher(ghCli)
}

func getRepoWrapper(ctx context.Context, ghCli *GitHub, name string) (map[string]any, error) {
	// TODO: this should be a provider interface, even if private
	slice := strings.Split(name, "/")
	if len(slice) != 2 {
		return nil, errors.New("invalid name")
	}

	repo, err := ghCli.GetRepository(ctx, slice[0], slice[1])
	if err != nil {
		return nil, err
	}

	repoProps := map[string]any{
		// general entity
		properties.PropertyName:       fmt.Sprintf("%s/%s", repo.GetOwner().GetLogin(), repo.GetName()),
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

	return repoProps, nil
}

// FetchProperty fetches a single property for the given entity
func (c *GitHub) FetchProperty(
	ctx context.Context, name string, entType minderv1.Entity, key string,
) (*properties.Property, error) {
	pf := newPropertyFetcher(c, entType)

	for _, po := range pf.propertyOrigins {
		for _, k := range po.keys {
			if k == key {
				props, err := po.wrapper(ctx, pf.ghCli, name)
				if err != nil {
					return nil, err
				}

				value, ok := props[key]
				if !ok {
					return nil, errors.New("requested property not found in result")
				}
				return properties.NewProperty(value), nil
			}
		}
	}

	return nil, errors.New("property not found")
}

// FetchAllProperties fetches all properties for the given entity
func (c *GitHub) FetchAllProperties(
	ctx context.Context, name string, entType minderv1.Entity,
) (*properties.Properties, error) {
	pf := newPropertyFetcher(c, entType)

	result := make(map[string]any)

	for _, po := range pf.propertyOrigins {
		props, err := po.wrapper(ctx, pf.ghCli, name)
		if err != nil {
			return nil, err
		}

		for k, v := range props {
			result[k] = v
		}
	}

	return properties.NewProperties(result), nil
}
