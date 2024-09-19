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

package selectors

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/stacklok/minder/internal/entities/models"
	"github.com/stacklok/minder/internal/entities/properties"
	internalpb "github.com/stacklok/minder/internal/proto"
	ghprops "github.com/stacklok/minder/internal/providers/github/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func repoToSelectorEntity(t *testing.T, name, class string, repoEntWithProps *models.EntityWithProperties) *internalpb.SelectorEntity {
	t.Helper()

	var isFork *bool
	if propIsFork, err := repoEntWithProps.Properties.GetProperty(properties.RepoPropertyIsFork).AsBool(); err == nil {
		isFork = proto.Bool(propIsFork)
	}

	var isPrivate *bool
	if propIsPrivate, err := repoEntWithProps.Properties.GetProperty(properties.RepoPropertyIsPrivate).AsBool(); err == nil {
		isPrivate = proto.Bool(propIsPrivate)
	}

	return &internalpb.SelectorEntity{
		EntityType: minderv1.Entity_ENTITY_REPOSITORIES,
		Name:       repoEntWithProps.Entity.Name,
		Provider: &internalpb.SelectorProvider{
			Name:  name,
			Class: class,
		},
		Entity: &internalpb.SelectorEntity_Repository{
			Repository: &internalpb.SelectorRepository{
				Name: repoEntWithProps.Entity.Name,
				Provider: &internalpb.SelectorProvider{
					Name:  name,
					Class: class,
				},
				IsFork:     isFork,
				IsPrivate:  isPrivate,
				Properties: repoEntWithProps.Properties.ToProtoStruct(),
			},
		},
	}
}

func artifactToSelectorEntity(t *testing.T, name, class string, artifactEntWithProps *models.EntityWithProperties) *internalpb.SelectorEntity {
	t.Helper()

	return &internalpb.SelectorEntity{
		EntityType: minderv1.Entity_ENTITY_ARTIFACTS,
		Name:       artifactEntWithProps.Entity.Name,
		Provider: &internalpb.SelectorProvider{
			Name:  name,
			Class: class,
		},
		Entity: &internalpb.SelectorEntity_Artifact{
			Artifact: &internalpb.SelectorArtifact{
				Name: artifactEntWithProps.Entity.Name,
				Provider: &internalpb.SelectorProvider{
					Name:  name,
					Class: class,
				},
				Properties: artifactEntWithProps.Properties.ToProtoStruct(),
			},
		},
	}
}

func pullRequestToSelectorEntity(t *testing.T, name, class string, pullRequestEntityWithProps *models.EntityWithProperties) *internalpb.SelectorEntity {
	t.Helper()

	return &internalpb.SelectorEntity{
		EntityType: minderv1.Entity_ENTITY_PULL_REQUESTS,
		Name:       pullRequestEntityWithProps.Entity.Name,
		Provider: &internalpb.SelectorProvider{
			Name:  name,
			Class: class,
		},
		Entity: &internalpb.SelectorEntity_PullRequest{
			PullRequest: &internalpb.SelectorPullRequest{
				Name:       pullRequestEntityWithProps.Entity.Name,
				Properties: pullRequestEntityWithProps.Properties.ToProtoStruct(),
			},
		},
	}
}

type fullProvider struct {
	name  string
	class string
	t     *testing.T
}

func (_ *fullProvider) CanImplement(_ minderv1.ProviderType) bool {
	return true
}

func (m *fullProvider) RepoToSelectorEntity(_ context.Context, repoEntWithProps *models.EntityWithProperties) *internalpb.SelectorEntity {
	return repoToSelectorEntity(m.t, m.name, m.class, repoEntWithProps)
}

func (m *fullProvider) ArtifactToSelectorEntity(_ context.Context, artifactEntWithProps *models.EntityWithProperties) *internalpb.SelectorEntity {
	return artifactToSelectorEntity(m.t, m.name, m.class, artifactEntWithProps)
}

func (m *fullProvider) PullRequestToSelectorEntity(_ context.Context, pullRequestEntityWithProps *models.EntityWithProperties) *internalpb.SelectorEntity {
	return pullRequestToSelectorEntity(m.t, m.name, m.class, pullRequestEntityWithProps)
}

func (_ *fullProvider) FetchAllProperties(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ *properties.Properties,
) (*properties.Properties, error) {
	return nil, nil
}

func (_ *fullProvider) FetchProperty(_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ string) (*properties.Property, error) {
	return nil, nil
}

func (_ *fullProvider) GetEntityName(_ minderv1.Entity, _ *properties.Properties) (string, error) {
	return "", nil
}

func (_ *fullProvider) SupportsEntity(_ minderv1.Entity) bool {
	return false
}

func (_ *fullProvider) RegisterEntity(_ context.Context, _ minderv1.Entity, _ *properties.Properties) (*properties.Properties, error) {
	return nil, nil
}

func (_ *fullProvider) DeregisterEntity(_ context.Context, _ minderv1.Entity, _ *properties.Properties) error {
	return nil
}

func (_ *fullProvider) ReregisterEntity(_ context.Context, _ minderv1.Entity, _ *properties.Properties) error {
	return nil
}

func newMockProvider(t *testing.T, name, class string) *fullProvider {
	t.Helper()

	return &fullProvider{
		name:  name,
		class: class,
		t:     t,
	}
}

type repoOnlyProvider struct {
	name  string
	class string
	t     *testing.T
}

func newRepoOnlyProvider(t *testing.T, name, class string) *repoOnlyProvider {
	t.Helper()

	return &repoOnlyProvider{
		name:  name,
		class: class,
		t:     t,
	}
}

func (_ *repoOnlyProvider) CanImplement(_ minderv1.ProviderType) bool {
	return true
}

func (_ *repoOnlyProvider) FetchAllProperties(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ *properties.Properties,
) (*properties.Properties, error) {
	return nil, nil
}

func (_ *repoOnlyProvider) FetchProperty(_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ string) (*properties.Property, error) {
	return nil, nil
}

func (_ *repoOnlyProvider) GetEntityName(_ minderv1.Entity, _ *properties.Properties) (string, error) {
	return "", nil
}

func (_ *repoOnlyProvider) SupportsEntity(minderv1.Entity) bool {
	return false
}

func (_ *repoOnlyProvider) RegisterEntity(_ context.Context, _ minderv1.Entity, _ *properties.Properties) (*properties.Properties, error) {
	return nil, nil
}

func (_ *repoOnlyProvider) DeregisterEntity(_ context.Context, _ minderv1.Entity, _ *properties.Properties) error {
	return nil
}

func (_ *repoOnlyProvider) ReregisterEntity(_ context.Context, _ minderv1.Entity, _ *properties.Properties) error {
	return nil
}

func (m *repoOnlyProvider) RepoToSelectorEntity(_ context.Context, repoEntWithProps *models.EntityWithProperties) *internalpb.SelectorEntity {
	return repoToSelectorEntity(m.t, m.name, m.class, repoEntWithProps)
}

func buildEntityWithProperties(entityType minderv1.Entity, name string, propMap map[string]any) *models.EntityWithProperties {
	props, err := properties.NewProperties(propMap)
	if err != nil {
		panic(err)
	}
	entity := &models.EntityWithProperties{
		Entity: models.EntityInstance{
			Type: entityType,
			Name: name,
		},
		Properties: props,
	}

	return entity
}

func TestEntityToSelectorEntity(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name        string
		provider    provifv1.Provider
		entityType  minderv1.Entity
		entityName  string
		entityProps map[string]any
		success     bool
	}{
		{
			name:       "Repository",
			provider:   newMockProvider(t, "github", "github"),
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			entityName: "testorg/testrepo",
			entityProps: map[string]any{
				properties.PropertyUpstreamID:    "12345",
				properties.RepoPropertyIsFork:    true,
				properties.RepoPropertyIsPrivate: true,
				ghprops.RepoPropertyId:           "12345",
				ghprops.RepoPropertyName:         "testrepo",
				ghprops.RepoPropertyOwner:        "testorg",
			},
			success: true,
		},
		{
			name:       "Artifact",
			provider:   newMockProvider(t, "github", "github"),
			entityType: minderv1.Entity_ENTITY_ARTIFACTS,
			entityName: "testorg/testartifact",
			entityProps: map[string]any{
				properties.PropertyUpstreamID:      "67890",
				ghprops.ArtifactPropertyOwner:      "testorg/testartifact",
				ghprops.ArtifactPropertyName:       "testorg/testartifact",
				ghprops.ArtifactPropertyType:       "container",
				ghprops.ArtifactPropertyCreatedAt:  "2024-01-01T00:00:00Z",
				ghprops.ArtifactPropertyVisibility: "public",
			},
			success: true,
		},
		{
			name:       "Pull Request",
			provider:   newMockProvider(t, "github", "github"),
			entityType: minderv1.Entity_ENTITY_PULL_REQUESTS,
			entityName: "testorg/testrepo/12345",
			entityProps: map[string]any{
				properties.PropertyUpstreamID: "12345",
				ghprops.PullPropertyURL:       "https://github.com/testorg/testrepo/pull/12345",
				ghprops.PullPropertyNumber:    "12345",
				ghprops.PullPropertySha:       "abc123",
				ghprops.PullPropertyRepoOwner: "testorg",
				ghprops.PullPropertyRepoName:  "testrepo",
				ghprops.PullPropertyAuthorID:  "56789",
				ghprops.PullPropertyAction:    "opened",
			},
			success: true,
		},
		{
			name:       "Repository with RepoOnlyProvider",
			provider:   newRepoOnlyProvider(t, "github", "github"),
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			entityName: "testorg/testrepo",
			entityProps: map[string]any{
				properties.PropertyUpstreamID:    "12345",
				properties.RepoPropertyIsFork:    true,
				properties.RepoPropertyIsPrivate: true,
				ghprops.RepoPropertyId:           "12345",
				ghprops.RepoPropertyName:         "testrepo",
				ghprops.RepoPropertyOwner:        "testorg",
			},
			success: true,
		},
		{
			name:       "Artifact with RepoOnlyProvider",
			provider:   newRepoOnlyProvider(t, "github", "github"),
			entityType: minderv1.Entity_ENTITY_ARTIFACTS,
			entityName: "testorg/testartifact",
			entityProps: map[string]any{
				properties.PropertyUpstreamID:      "67890",
				ghprops.ArtifactPropertyOwner:      "testorg/testartifact",
				ghprops.ArtifactPropertyName:       "testorg/testartifact",
				ghprops.ArtifactPropertyType:       "container",
				ghprops.ArtifactPropertyCreatedAt:  "2024-01-01T00:00:00Z",
				ghprops.ArtifactPropertyVisibility: "public",
			},
			success: false,
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario

		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			entity := buildEntityWithProperties(scenario.entityType, scenario.entityName, scenario.entityProps)

			selEnt := EntityToSelectorEntity(
				context.Background(),
				scenario.provider,
				scenario.entityType,
				entity,
			)

			if scenario.success {
				require.NotNil(t, selEnt)
				require.Equal(t, scenario.entityType, selEnt.GetEntityType())
			} else {
				require.Nil(t, selEnt)
			}
		})
	}
}
