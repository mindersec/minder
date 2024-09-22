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

	"github.com/stacklok/minder/internal/entities/models"
	"github.com/stacklok/minder/internal/entities/properties"
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

func TestEntityToSelectorEntity(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name        string
		entityType  minderv1.Entity
		entityName  string
		entityProps map[string]any
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
				require.Equal(t, scenario.entityType, selEnt.GetEntityType())
			} else {
				require.Nil(t, selEnt)
			}
		})
	}
}
