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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	internalpb "github.com/stacklok/minder/internal/proto"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func repoToSelectorEntity(t *testing.T, name, class string, repo *minderv1.Repository) *internalpb.SelectorEntity {
	t.Helper()

	return &internalpb.SelectorEntity{
		EntityType: minderv1.Entity_ENTITY_REPOSITORIES,
		Name:       fmt.Sprintf("%s/%s", repo.GetOwner(), repo.GetName()),
		Provider: &internalpb.SelectorProvider{
			Name:  name,
			Class: class,
		},
		Entity: &internalpb.SelectorEntity_Repository{
			Repository: &internalpb.SelectorRepository{
				Name: fmt.Sprintf("%s/%s", repo.GetOwner(), repo.GetName()),
				Provider: &internalpb.SelectorProvider{
					Name:  name,
					Class: class,
				},
				IsFork:    proto.Bool(repo.GetIsFork()),
				IsPrivate: proto.Bool(repo.GetIsFork()),
			},
		},
	}
}

func artifactToSelectorEntity(t *testing.T, name, class string, artifact *minderv1.Artifact) *internalpb.SelectorEntity {
	t.Helper()

	return &internalpb.SelectorEntity{
		EntityType: minderv1.Entity_ENTITY_ARTIFACTS,
		Name:       fmt.Sprintf("%s/%s", artifact.GetOwner(), artifact.GetName()),
		Provider: &internalpb.SelectorProvider{
			Name:  name,
			Class: class,
		},
		Entity: &internalpb.SelectorEntity_Artifact{
			Artifact: &internalpb.SelectorArtifact{
				Name: fmt.Sprintf("%s/%s", artifact.GetOwner(), artifact.GetName()),
				Provider: &internalpb.SelectorProvider{
					Name:  name,
					Class: class,
				},
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

func (m *fullProvider) RepoToSelectorEntity(_ context.Context, repo *minderv1.Repository) *internalpb.SelectorEntity {
	return repoToSelectorEntity(m.t, m.name, m.class, repo)
}

func (m *fullProvider) ArtifactToSelectorEntity(_ context.Context, artifact *minderv1.Artifact) *internalpb.SelectorEntity {
	return artifactToSelectorEntity(m.t, m.name, m.class, artifact)
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

func (m *repoOnlyProvider) RepoToSelectorEntity(_ context.Context, repo *minderv1.Repository) *internalpb.SelectorEntity {
	return repoToSelectorEntity(m.t, m.name, m.class, repo)
}

func TestEntityToSelectorEntity(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name       string
		provider   provifv1.Provider
		entityType minderv1.Entity
		entity     proto.Message
		success    bool
	}{
		{
			name:       "Repository",
			provider:   newMockProvider(t, "github", "github"),
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			entity: &minderv1.Repository{
				Owner:     "testorg",
				Name:      "testrepo",
				IsFork:    true,
				IsPrivate: true,
			},
			success: true,
		},
		{
			name:       "Artifact",
			provider:   newMockProvider(t, "github", "github"),
			entityType: minderv1.Entity_ENTITY_ARTIFACTS,
			entity: &minderv1.Artifact{
				Owner: "testorg",
				Name:  "testartifact",
				Type:  "container",
			},
			success: true,
		},
		{
			name:       "Repository with RepoOnlyProvider",
			provider:   newRepoOnlyProvider(t, "github", "github"),
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			entity: &minderv1.Repository{
				Owner:     "testorg",
				Name:      "testrepo",
				IsFork:    true,
				IsPrivate: true,
			},
			success: true,
		},
		{
			name:       "Artifact with RepoOnlyProvider",
			provider:   newRepoOnlyProvider(t, "github", "github"),
			entityType: minderv1.Entity_ENTITY_ARTIFACTS,
			entity: &minderv1.Artifact{
				Owner: "testorg",
				Name:  "testartifact",
				Type:  "container",
			},
			success: false,
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario

		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			selEnt := EntityToSelectorEntity(
				context.Background(),
				scenario.provider,
				scenario.entityType,
				scenario.entity,
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
