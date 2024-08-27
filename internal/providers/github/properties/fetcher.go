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

	go_github "github.com/google/go-github/v63/github"

	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// GhPropertyWrapper is a function that fetches a property from the GitHub API
type GhPropertyWrapper func(ctx context.Context, ghCli *go_github.Client, name string) (map[string]any, error)

// GhPropertyFetcher is an interface for fetching properties from the GitHub API
type GhPropertyFetcher interface {
	WrapperForProperty(propertyKey string) GhPropertyWrapper
	AllPropertyWrappers() []GhPropertyWrapper
	OperationalProperties() []string
	GetName(props *properties.Properties) (string, error)
}

// GhPropertyFetcherFactory is an interface for creating GhPropertyFetcher instances
type GhPropertyFetcherFactory interface {
	EntityPropertyFetcher(entType minderv1.Entity) GhPropertyFetcher
}

type ghEntityFetcher struct{}

// NewPropertyFetcherFactory creates a new GhPropertyFetcherFactory
func NewPropertyFetcherFactory() GhPropertyFetcherFactory {
	return ghEntityFetcher{}
}

// EntityPropertyFetcher returns a GhPropertyFetcher for the given entity type
func (_ ghEntityFetcher) EntityPropertyFetcher(entType minderv1.Entity) GhPropertyFetcher {
	if entType == minderv1.Entity_ENTITY_REPOSITORIES {
		return NewRepositoryFetcher()
	}

	return nil
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
		return nil, fmt.Errorf("error fetching is_archived property: %w", err)
	}

	isFork, err := repoProperties.GetProperty(properties.RepoPropertyIsFork).AsBool()
	if err != nil {
		return nil, fmt.Errorf("error fetching is_archived property: %w", err)
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
