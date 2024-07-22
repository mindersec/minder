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
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	internalpb "github.com/stacklok/minder/internal/proto"
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

func withProperties(properties map[string]any) testRepoOption {
	return func(selRepo *internalpb.SelectorRepository) {
		protoProperties, err := structpb.NewStruct(properties)
		if err != nil {
			panic(err)
		}
		selRepo.Properties = protoProperties
	}
}

func TestSelectSelectorEntity(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name                    string
		exprs                   []*minderv1.Profile_Selector
		selectOptions           []SelectOption
		selectorEntityBld       testSelectorEntityBuilder
		expectedNewSelectionErr string
		expectedSelectErr       error
		selected                bool
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
			selectorEntityBld:       newTestRepoSelectorEntity(),
			expectedNewSelectionErr: "undeclared reference to 'artifact'",
			selected:                false,
		},
		{
			name: "Attempt to use a repo attribute that doesn't exist",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.iamnothere == 'value'",
				},
			},
			selectorEntityBld:       newTestRepoSelectorEntity(),
			expectedNewSelectionErr: "undefined field 'iamnothere'",
			selected:                false,
		},
		{
			name: "Use a property that is defined and true result",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.properties.github['is_fork'] == false",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(
				withProperties(map[string]any{
					"github": map[string]any{"is_fork": false},
				}),
			),
			selected: true,
		},
		{
			name: "Use a string property that is defined and true result",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.properties.license == 'MIT'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(
				withProperties(map[string]any{
					"license": "MIT",
				}),
			),
			selected: true,
		},
		{
			name: "Use a string property that is defined and false result",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.properties.license == 'MIT'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(
				withProperties(map[string]any{
					"license": "BSD",
				}),
			),
			selected: false,
		},
		{
			name: "Use a property that is defined and false result",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.properties.github['is_fork'] == false",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(
				withProperties(map[string]any{
					"github": map[string]any{"is_fork": true},
				}),
			),
			selected: false,
		},
		{
			name: "Properties are non-nil but we use one that is not defined",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.properties.github['is_private'] != true",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(
				withProperties(map[string]any{
					"github": map[string]any{"is_fork": true},
				}),
			),
			expectedSelectErr: ErrResultUnknown,
			selected:          false,
		},
		{
			name: "Attempt to use a property while having nil properties",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.properties.github['is_fork'] != 'true'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			expectedSelectErr: ErrResultUnknown,
			selected:          false,
		},
		{
			name: "The selector shortcuts if evaluation is not needed for properties",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.name == 'testorg/testrepo' || repository.properties.github['is_fork'] != 'true'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          true,
		},
		{
			name: "Attempt to use a property but explicitly tell Select that it's not defined",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.properties.github['is_fork'] != 'true'",
				},
			},
			selectOptions: []SelectOption{
				WithUnknownPaths("repository.properties"),
			},
			selectorEntityBld: newTestRepoSelectorEntity(
				withProperties(map[string]any{
					"github": map[string]any{"is_fork": true},
				}),
			),
			expectedSelectErr: ErrResultUnknown,
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
			if scenario.expectedNewSelectionErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), scenario.expectedNewSelectionErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, sels)

			selected, err := sels.Select(se, scenario.selectOptions...)
			if scenario.expectedSelectErr != nil {
				require.Error(t, err)
				require.Equal(t, scenario.expectedSelectErr, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, scenario.selected, selected)
		})
	}
}

func TestSelectorEntityFillProperties(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name           string
		exprs          []*minderv1.Profile_Selector
		mockFetch      func(*internalpb.SelectorEntity)
		secondSucceeds bool
	}{
		{
			name: "Fetch a property that exists",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.properties.github['is_fork'] == false",
				},
			},
			mockFetch: func(se *internalpb.SelectorEntity) {
				se.Entity.(*internalpb.SelectorEntity_Repository).Repository.Properties = &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"github": {
							Kind: &structpb.Value_StructValue{
								StructValue: &structpb.Struct{
									Fields: map[string]*structpb.Value{
										"is_fork": {
											Kind: &structpb.Value_BoolValue{
												BoolValue: false,
											},
										},
									},
								},
							},
						},
					}}
			},
			secondSucceeds: true,
		},
		{
			name: "Fail to fetch a property",
			exprs: []*minderv1.Profile_Selector{
				{
					Entity:   minderv1.RepositoryEntity.String(),
					Selector: "repository.properties.github['is_private'] == false",
				},
			},
			mockFetch: func(se *internalpb.SelectorEntity) {
				se.Entity.(*internalpb.SelectorEntity_Repository).Repository.Properties = &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"github": {
							Kind: &structpb.Value_StructValue{
								StructValue: &structpb.Struct{
									Fields: map[string]*structpb.Value{
										"is_fork": {
											Kind: &structpb.Value_BoolValue{
												BoolValue: false,
											},
										},
									},
								},
							},
						},
					}}
			},
			secondSucceeds: false,
		},
	}

	for _, scenario := range scenarios {
		env, err := NewEnv()
		require.NoError(t, err)

		seBuilder := newTestRepoSelectorEntity()
		se := seBuilder()

		sels, err := env.NewSelectionFromProfile(se.EntityType, scenario.exprs)
		require.NoError(t, err)
		require.NotNil(t, sels)

		_, err = sels.Select(se, WithUnknownPaths("repository.properties"))
		require.ErrorIs(t, err, ErrResultUnknown)

		// simulate fetching properties
		scenario.mockFetch(se)

		selected, err := sels.Select(se)
		if scenario.secondSucceeds {
			require.NoError(t, err)
			require.True(t, selected)
		} else {
			require.ErrorIs(t, err, ErrResultUnknown)
		}
	}
}
