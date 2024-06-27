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
	internalpb "github.com/stacklok/minder/internal/proto"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/engine/entities"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestNewSelectorEngine(t *testing.T) {
	t.Parallel()

	env, err := NewEnv()
	require.NoError(t, err)
	require.NotNil(t, env)
	require.NotNil(t, env.entityEnvs)
	require.NotNil(t, env.entityEnvs[minderv1.Entity_ENTITY_REPOSITORIES])
	require.NotNil(t, env.entityEnvs[minderv1.Entity_ENTITY_ARTIFACTS])
}

type testSelectorEntityBuilder func() *internalpb.SelectorEntity
type testRepoOption func(selRepo *internalpb.SelectorRepository)
type testArtifactOption func(selArtifact *internalpb.SelectorArtifact)

func newTestArtifactSelectorEntity(artifactOpts ...testArtifactOption) testSelectorEntityBuilder {
	return func() *internalpb.SelectorEntity {
		artifact := &internalpb.SelectorEntity{
			EntityType: minderv1.Entity_ENTITY_ARTIFACTS,
			Name:       "testorg/testartifact",
			Entity: &internalpb.SelectorEntity_Artifact{
				Artifact: &internalpb.SelectorArtifact{
					Name: "testorg/testartifact",
					Type: "container",
				},
			},
		}

		for _, opt := range artifactOpts {
			opt(artifact.Entity.(*internalpb.SelectorEntity_Artifact).Artifact)
		}

		return artifact
	}
}

func newTestRepoSelectorEntity(repoOpts ...testRepoOption) testSelectorEntityBuilder {
	return func() *internalpb.SelectorEntity {
		repo := &internalpb.SelectorEntity{
			EntityType: minderv1.Entity_ENTITY_REPOSITORIES,
			Name:       "testorg/testrepo",
			Entity: &internalpb.SelectorEntity_Repository{
				Repository: &internalpb.SelectorRepository{
					Name: "testorg/testrepo",
				},
			},
		}

		for _, opt := range repoOpts {
			opt(repo.Entity.(*internalpb.SelectorEntity_Repository).Repository)
		}

		return repo
	}
}

func withIsFork(isFork bool) testRepoOption {
	return func(selRepo *internalpb.SelectorRepository) {
		selRepo.IsFork = &isFork
	}
}

func TestSelectSelectorEntity(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name              string
		exprs             []*minderv1.Profile_Selector
		selectorEntityBld testSelectorEntityBuilder
		expectedErr       string
		selected          bool
	}{
		{
			name:              "No selectors",
			exprs:             []*minderv1.Profile_Selector{},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          true,
		},
		{
			name: "Simple true repository expression",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.name == 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          true,
		},
		{
			name: "Simple true artifact expression",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.ArtifactEntity.String(),
					Selector: "artifact.type == 'container'",
				},
			},
			selectorEntityBld: newTestArtifactSelectorEntity(),
			selected:          true,
		},
		{
			name: "Simple false artifact expression",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.ArtifactEntity.String(),
					Selector: "artifact.type != 'container'",
				},
			},
			selectorEntityBld: newTestArtifactSelectorEntity(),
			selected:          false,
		},
		{
			name: "Simple false repository expression",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.name != 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          false,
		},
		{
			name: "Simple true generic entity expression for repo entity type",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "entity.name == 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          true,
		},
		{
			name: "Simple false generic entity expression for repo entity type",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "entity.name != 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          false,
		},
		{
			name: "Simple true generic entity expression for unspecified entity type",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   "",
					Selector: "entity.name == 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          true,
		},
		{
			name: "Simple false generic entity expression for unspecified entity type",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   "",
					Selector: "entity.name != 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          false,
		},
		{
			name: "Expressions for different types than the entity are skipped",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.ArtifactEntity.String(),
					Selector: "artifact.name != 'namespace/containername'",
				},
				{
					Entity:   minderv1.PullRequestEntity.String(),
					Selector: "pull_request.name != 'namespace/containername'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          true,
		},
		{
			name: "Expression on is_fork bool attribute set to true",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.is_fork == true",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(withIsFork(true)),
			selected:          true,
		},
		{
			name: "Expression on is_fork bool attribute set to false",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.is_fork == true",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(withIsFork(false)),
			selected:          false,
		},
		{
			name: "Expression on is_fork bool attribute set to nil and true expression",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.is_fork == true",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          false,
		},
		{
			name: "Expression on is_fork bool attribute set to nil and false expression",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.is_fork == false",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          true,
		},
		{
			name: "Wrong entity type - repo selector uses artifact",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "artifact.name != 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			expectedErr:       "undeclared reference to 'artifact'",
			selected:          false,
		},
		{
			name: "Attempt to use a repo attribute that doesn't exist",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.iamnothere == 'value'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			expectedErr:       "undefined field 'iamnothere'",
			selected:          false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			env, err := NewEnv()
			require.NoError(t, err)

			se := scenario.selectorEntityBld()

			sels, err := env.NewSelectionFromProfile(se.EntityType, scenario.exprs)
			if scenario.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), scenario.expectedErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, sels)

			selected, err := sels.Select(se)
			require.NoError(t, err)
			require.Equal(t, scenario.selected, selected)
		})
	}
}

func TestSelectEntityInfoWrapper(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name           string
		eiwConstructor func() *entities.EntityInfoWrapper
		exprs          []*minderv1.Profile_Selector
		selected       bool
		expectedErr    string
	}{
		{
			name: "Simple true repository expression",
			eiwConstructor: func() *entities.EntityInfoWrapper {
				eiw := entities.NewEntityInfoWrapper()
				eiw.WithRepository(&minderv1.Repository{
					Owner:     "stacklok",
					Name:      "minder",
					IsPrivate: false,
					IsFork:    false,
				})
				return eiw
			},
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "entity.name == 'stacklok/minder'",
				},
			},
			selected: true,
		},
		{
			name: "Simple true artifact expression",
			eiwConstructor: func() *entities.EntityInfoWrapper {
				eiw := entities.NewEntityInfoWrapper()
				eiw.WithArtifact(&minderv1.Artifact{
					Owner: "stacklok",
					Name:  "minder",
					Type:  "container",
				})
				return eiw
			},
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.ArtifactEntity.String(),
					Selector: "artifact.type == 'container'",
				},
			},
			selected: true,
		},
		{
			name: "Simple false artifact expression",
			eiwConstructor: func() *entities.EntityInfoWrapper {
				eiw := entities.NewEntityInfoWrapper()
				eiw.WithArtifact(&minderv1.Artifact{
					Owner: "stacklok",
					Name:  "minder",
					Type:  "container",
				})
				return eiw
			},
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.ArtifactEntity.String(),
					Selector: "artifact.type != 'container'",
				},
			},
			selected: false,
		},
		{
			name: "Simple false repository expression",
			eiwConstructor: func() *entities.EntityInfoWrapper {
				eiw := entities.NewEntityInfoWrapper()
				eiw.WithRepository(&minderv1.Repository{
					Owner:     "stacklok",
					Name:      "minder",
					IsPrivate: false,
					IsFork:    false,
				})
				return eiw
			},
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "entity.name != 'stacklok/minder'",
				},
			},
			selected: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			env, err := NewEnv()
			require.NoError(t, err)

			eiw := scenario.eiwConstructor()

			sels, err := env.NewSelectionFromProfile(eiw.Type, scenario.exprs)
			if scenario.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), scenario.expectedErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, sels)

			selected, err := sels.SelectEiw(eiw)
			require.NoError(t, err)
			require.Equal(t, scenario.selected, selected)
		})
	}
}
