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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/entities/models"
	"github.com/stacklok/minder/internal/entities/properties"
	internalpb "github.com/stacklok/minder/internal/proto"
	ghprops "github.com/stacklok/minder/internal/providers/github/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

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

func checkProps(t *testing.T, got *structpb.Struct, expected map[string]any) {
	t.Helper()

	gotMap := got.AsMap()
	assert.Equal(t, gotMap, expected)
}

func checkSelEntArtifact(t *testing.T, got, expected *internalpb.SelectorArtifact, propMap map[string]any) {
	t.Helper()

	assert.Equal(t, got.Name, expected.Name)
	assert.Equal(t, got.Type, expected.Type)
	checkProps(t, got.Properties, propMap)
}

func checkSelEntRepo(t *testing.T, got, expected *internalpb.SelectorRepository, propMap map[string]any) {
	t.Helper()

	assert.Equal(t, got.Name, expected.Name)
	assert.Equal(t, got.IsFork, expected.IsFork)
	checkProps(t, got.Properties, propMap)
}

func checkSelEntPullRequest(t *testing.T, got, expected *internalpb.SelectorPullRequest, propMap map[string]any) {
	t.Helper()

	assert.Equal(t, got.Name, expected.Name)
	checkProps(t, got.Properties, propMap)
}

func checkSelEnt(t *testing.T, got, expected *internalpb.SelectorEntity, propMap map[string]any) {
	t.Helper()

	assert.Equal(t, got.EntityType, expected.EntityType)
	switch got.EntityType { // nolint:exhaustive
	case minderv1.Entity_ENTITY_REPOSITORIES:
		checkSelEntRepo(t, got.GetRepository(), expected.GetRepository(), propMap)
	case minderv1.Entity_ENTITY_ARTIFACTS:
		checkSelEntArtifact(t, got.GetArtifact(), expected.GetArtifact(), propMap)
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		checkSelEntPullRequest(t, got.GetPullRequest(), expected.GetPullRequest(), propMap)
	}
}

func TestEntityToSelectorEntity(t *testing.T) {
	t.Parallel()

	trueBool := true

	scenarios := []struct {
		name        string
		entityType  minderv1.Entity
		entityName  string
		entityProps map[string]any
		expSelEnt   *internalpb.SelectorEntity
		checkSelEnt func(proto.Message)
		success     bool
	}{
		{
			name:       "Repository",
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
			expSelEnt: &internalpb.SelectorEntity{
				EntityType: minderv1.Entity_ENTITY_REPOSITORIES,
				Name:       "testorg/testrepo",
				Entity: &internalpb.SelectorEntity_Repository{
					Repository: &internalpb.SelectorRepository{
						Name:      "testorg/testrepo",
						IsFork:    &trueBool,
						IsPrivate: &trueBool,
					},
				},
			},
			success: true,
		},
		{
			name:       "Artifact",
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
			expSelEnt: &internalpb.SelectorEntity{
				EntityType: minderv1.Entity_ENTITY_ARTIFACTS,
				Name:       "testorg/testartifact",
				Entity: &internalpb.SelectorEntity_Artifact{
					Artifact: &internalpb.SelectorArtifact{
						Name: "testorg/testartifact",
						Type: "container",
					},
				},
			},
			success: true,
		},
		{
			name:       "Pull Request",
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
			expSelEnt: &internalpb.SelectorEntity{
				EntityType: minderv1.Entity_ENTITY_PULL_REQUESTS,
				Name:       "testorg/testrepo/12345",
				Entity: &internalpb.SelectorEntity_PullRequest{
					PullRequest: &internalpb.SelectorPullRequest{
						Name: "testorg/testrepo/12345",
					},
				},
			},
			success: true,
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario

		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			entity := buildEntityWithProperties(scenario.entityType, scenario.entityName, scenario.entityProps)

			selEnt := EntityToSelectorEntity(
				context.Background(),
				scenario.entityType,
				entity,
			)

			if scenario.success {
				require.NotNil(t, selEnt)
				require.Equal(t, selEnt.GetEntityType(), scenario.entityType)
				require.Equal(t, selEnt.GetName(), scenario.entityName)
				checkSelEnt(t, selEnt, scenario.expSelEnt, scenario.entityProps)
			} else {
				require.Nil(t, selEnt)
			}
		})
	}
}
