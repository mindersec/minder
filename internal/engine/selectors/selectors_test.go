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
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/profiles/models"
	internalpb "github.com/stacklok/minder/internal/proto"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestNewSelectorEngine(t *testing.T) {
	t.Parallel()

	env := NewEnv()
	require.NotNil(t, env)
	require.NotNil(t, env.entityEnvs)
	require.NotNil(t, env.entityEnvs[minderv1.Entity_ENTITY_REPOSITORIES])
	require.NotNil(t, env.entityEnvs[minderv1.Entity_ENTITY_ARTIFACTS])
}

type testSelectorEntityBuilder func() *internalpb.SelectorEntity
type testRepoOption func(selRepo *internalpb.SelectorRepository)
type testArtifactOption func(selArtifact *internalpb.SelectorArtifact)
type testPrOption func(selPr *internalpb.SelectorPullRequest)

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

func repoWithProperties(properties map[string]any) testRepoOption {
	return func(selRepo *internalpb.SelectorRepository) {
		protoProperties, err := structpb.NewStruct(properties)
		if err != nil {
			panic(err)
		}
		selRepo.Properties = protoProperties
	}
}

func prWithProperties(properties map[string]any) testPrOption {
	return func(selPr *internalpb.SelectorPullRequest) {
		protoProperties, err := structpb.NewStruct(properties)
		if err != nil {
			panic(err)
		}
		selPr.Properties = protoProperties
	}
}

func newTestPullRequestSelectorEntity(prOpts ...testPrOption) testSelectorEntityBuilder {
	return func() *internalpb.SelectorEntity {
		pr := &internalpb.SelectorEntity{
			EntityType: minderv1.Entity_ENTITY_PULL_REQUESTS,
			Name:       "testorg/testrepo/123",
			Entity: &internalpb.SelectorEntity_PullRequest{
				PullRequest: &internalpb.SelectorPullRequest{
					Name: "testorg/testrepo/123",
				},
			},
		}

		for _, opt := range prOpts {
			opt(pr.Entity.(*internalpb.SelectorEntity_PullRequest).PullRequest)
		}

		return pr
	}
}

func TestSelectSelectorEntity(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name                       string
		exprs                      []models.ProfileSelector
		selectOptions              []SelectOption
		selectorEntityBld          testSelectorEntityBuilder
		expectedNewSelectionErrMsg string
		expectedNewSelectionErr    error
		expectedSelectErr          error
		expectedStructuredErr      *ErrStructure
		selected                   bool
		index                      int
	}{
		{
			name:              "No selectors",
			exprs:             []models.ProfileSelector{},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          true,
		},
		{
			name: "Simple true repository expression",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.name == 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          true,
		},
		{
			name: "Simple true artifact expression",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_ARTIFACTS,
					Selector: "artifact.type == 'container'",
				},
			},
			selectorEntityBld: newTestArtifactSelectorEntity(),
			selected:          true,
		},
		{
			name: "Simple false artifact expression",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_ARTIFACTS,
					Selector: "artifact.type != 'container'",
				},
			},
			selectorEntityBld: newTestArtifactSelectorEntity(),
			selected:          false,
		},
		{
			name: "Simple false repository expression",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.name != 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          false,
		},
		{
			name: "Simple true pull request expression",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_PULL_REQUESTS,
					Selector: "pull_request.name == 'testorg/testrepo/123'",
				},
			},
			selectorEntityBld: newTestPullRequestSelectorEntity(),
			selected:          true,
		},
		{
			name: "Simple false pull request expression",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_PULL_REQUESTS,
					Selector: "pull_request.name != 'testorg/testrepo/123'",
				},
			},
			selectorEntityBld: newTestPullRequestSelectorEntity(),
			selected:          false,
		},
		{
			name: "Simple true generic entity expression for repo entity type",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "entity.name == 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          true,
		},
		{
			name: "Simple false generic entity expression for repo entity type",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "entity.name != 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          false,
		},
		{
			name: "Simple true generic entity expression for unspecified entity type",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_UNSPECIFIED,
					Selector: "entity.name == 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          true,
		},
		{
			name: "Simple false generic entity expression for unspecified entity type",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_UNSPECIFIED,
					Selector: "entity.name != 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          false,
		},
		{
			name: "Expressions for different types than the entity are skipped",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_ARTIFACTS,
					Selector: "artifact.name != 'namespace/containername'",
				},
				{
					Entity:   minderv1.Entity_ENTITY_PULL_REQUESTS,
					Selector: "pull_request.name != 'namespace/containername'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          true,
		},
		{
			name: "Expression on is_fork bool attribute set to true",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.is_fork == true",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(withIsFork(true)),
			selected:          true,
		},
		{
			name: "Expression on is_fork bool attribute set to false",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.is_fork == true",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(withIsFork(false)),
			selected:          false,
		},
		{
			name: "Expression on is_fork bool attribute set to nil and true expression",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.is_fork == true",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          false,
		},
		{
			name: "Expression on is_fork bool attribute set to nil and false expression",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.is_fork == false",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          true,
		},
		{
			name: "Wrong entity type - repo selector uses artifact",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "artifact.name != 'testorg/testrepo'",
				},
			},
			selectorEntityBld:          newTestRepoSelectorEntity(),
			expectedNewSelectionErrMsg: "undeclared reference to 'artifact'",
			selected:                   false,
		},
		{
			name: "CEL expression that does not parse",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.name == ",
				},
			},
			selectorEntityBld:       newTestRepoSelectorEntity(),
			expectedNewSelectionErr: &ParseError{},
			selected:                false,
		},
		{
			name: "Attempt to use a repo attribute that doesn't exist",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.iamnothere == 'value'",
				},
			},
			selectorEntityBld:          newTestRepoSelectorEntity(),
			expectedNewSelectionErrMsg: "undefined field 'iamnothere'",
			expectedNewSelectionErr:    &CheckError{},
			expectedStructuredErr: &ErrStructure{
				Err: ErrKindCheck,
				Details: ErrDetails{
					Errors: []ErrInstance{
						{
							Line: 1,
							Col:  10,
							Msg:  "undefined field 'iamnothere'",
						},
					},
					Source: "repository.iamnothere == 'value'",
				},
			},
			selected: false,
		},
		{
			name: "Use a property that is defined and true result",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.properties.github['is_fork'] == false",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(
				repoWithProperties(map[string]any{
					"github": map[string]any{"is_fork": false},
				}),
			),
			selected: true,
		},
		{
			name: "Use a string property that is defined and true result",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.properties.license == 'MIT'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(
				repoWithProperties(map[string]any{
					"license": "MIT",
				}),
			),
			selected: true,
		},
		{
			name: "Use a string property that is defined and false result",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.properties.license == 'MIT'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(
				repoWithProperties(map[string]any{
					"license": "BSD",
				}),
			),
			selected: false,
		},
		{
			name: "Use a property that is defined and false result",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.properties.github['is_fork'] == false",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(
				repoWithProperties(map[string]any{
					"github": map[string]any{"is_fork": true},
				}),
			),
			selected: false,
		},
		{
			name: "Properties are non-nil but we use one that is not defined",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.properties.github['is_private'] != true",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(
				repoWithProperties(map[string]any{
					"github": map[string]any{"is_fork": true},
				}),
			),
			expectedSelectErr: ErrResultUnknown,
			selected:          false,
		},
		{
			name: "Attempt to use a property while having nil properties",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.properties.github['is_fork'] != 'true'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			expectedSelectErr: ErrResultUnknown,
			selected:          false,
		},
		{
			name: "The selector shortcuts if evaluation is not needed for properties",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.name == 'testorg/testrepo' || repository.properties.github['is_fork'] != 'true'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(),
			selected:          true,
		},
		{
			name: "Attempt to use a property but explicitly tell Select that it's not defined",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.properties.github['is_fork'] != 'true'",
				},
			},
			selectOptions: []SelectOption{
				WithUnknownPaths("repository.properties"),
			},
			selectorEntityBld: newTestRepoSelectorEntity(
				repoWithProperties(map[string]any{
					"github": map[string]any{"is_fork": true},
				}),
			),
			expectedSelectErr: ErrResultUnknown,
			selected:          false,
		},
		{
			name: "Use a PR property that is defined and true result",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_PULL_REQUESTS,
					Selector: "pull_request.properties.github['is_draft'] == false",
				},
			},
			selectorEntityBld: newTestPullRequestSelectorEntity(
				prWithProperties(map[string]any{
					"github": map[string]any{"is_draft": false},
				}),
			),
			selected: true,
		},
		{
			name: "Use a PR property that is defined and false result",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_PULL_REQUESTS,
					Selector: "pull_request.properties.github['is_draft'] == false",
				},
			},
			selectorEntityBld: newTestPullRequestSelectorEntity(
				prWithProperties(map[string]any{
					"github": map[string]any{"is_draft": true},
				}),
			),
			selected: false,
		},
		{
			name: "Multiple selectors with the same entity type, the first one is true",
			exprs: []models.ProfileSelector{
				{
					// true expression, should be evaluated and the entity should be kept for selection
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.name == 'testorg/testrepo'",
				},
				{
					// false expression, should cause the entity to be skipped
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.is_fork == false",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(withIsFork(true)),
			index:             1,
			selected:          false,
		},
		{
			name: "Multiple selectors with different entity types",
			exprs: []models.ProfileSelector{
				{
					// this one will be skipped as it's for a different entity
					Entity:   minderv1.Entity_ENTITY_PULL_REQUESTS,
					Selector: "pull_request.name == 'testorg/testrepo/123'",
				},
				{
					// false expression, should cause the entity to be skipped
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.name != 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(withIsFork(true)),
			index:             1,
			selected:          false,
		},
		{
			name: "Multiple selectors with generic entity type",
			exprs: []models.ProfileSelector{
				{
					// true expression, will be evaluated, but evaluates to true
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
					Selector: "repository.name == 'testorg/testrepo'",
				},
				{
					// false expression, will cause the entity to be skipped
					Entity:   minderv1.Entity_ENTITY_UNSPECIFIED,
					Selector: "entity.name != 'testorg/testrepo'",
				},
			},
			selectorEntityBld: newTestRepoSelectorEntity(withIsFork(true)),
			index:             1,
			selected:          false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			env := NewEnv()

			se := scenario.selectorEntityBld()

			sels, err := env.NewSelectionFromProfile(se.EntityType, scenario.exprs)
			if scenario.expectedNewSelectionErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), scenario.expectedNewSelectionErrMsg)
			}
			if scenario.expectedNewSelectionErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, scenario.expectedNewSelectionErr)
			}
			if scenario.expectedStructuredErr != nil {
				testErrUnmarshallableValue(t, err, scenario.expectedStructuredErr)
			}

			if scenario.expectedNewSelectionErrMsg != "" ||
				scenario.expectedNewSelectionErr != nil ||
				scenario.expectedStructuredErr != nil {
				return
			}

			require.NoError(t, err)
			require.NotNil(t, sels)

			selected, matchedSelector, err := sels.Select(se, scenario.selectOptions...)
			if scenario.expectedSelectErr != nil {
				require.Error(t, err)
				require.Equal(t, scenario.expectedSelectErr, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, scenario.selected, selected)
			if !selected {
				require.Equal(t, scenario.exprs[scenario.index].Selector, matchedSelector)
			}
		})
	}
}

func testErrUnmarshallableValue(t *testing.T, err error, expected *ErrStructure) {
	t.Helper()

	var ce *CheckError
	var pe *ParseError
	var jsonString string

	if errors.As(err, &ce) {
		jsonString = ce.Error()
	} else if errors.As(err, &pe) {
		jsonString = pe.Error()
	} else {
		t.Fatalf("error is not of type CheckError or ParseError")
	}

	// both errors unwrap to ErrSelectorCheck
	require.ErrorIs(t, err, ErrSelectorCheck)

	var structuredErr ErrStructure
	if err := json.NewDecoder(strings.NewReader(jsonString)).Decode(&structuredErr); err != nil {
		t.Fatalf("failed to unmarshal error: %v", err)
	}

	require.Equal(t, expected, &structuredErr)
}

func TestSelectorEntityFillProperties(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name           string
		exprs          []models.ProfileSelector
		mockFetch      func(*internalpb.SelectorEntity)
		secondSucceeds bool
	}{
		{
			name: "Fetch a property that exists",
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
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
			exprs: []models.ProfileSelector{
				{
					Entity:   minderv1.Entity_ENTITY_REPOSITORIES,
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
		env := NewEnv()

		seBuilder := newTestRepoSelectorEntity()
		se := seBuilder()

		sels, err := env.NewSelectionFromProfile(se.EntityType, scenario.exprs)
		require.NoError(t, err)
		require.NotNil(t, sels)

		_, _, err = sels.Select(se, WithUnknownPaths("repository.properties"))
		require.ErrorIs(t, err, ErrResultUnknown)

		// simulate fetching properties
		scenario.mockFetch(se)

		selected, _, err := sels.Select(se)
		if scenario.secondSucceeds {
			require.NoError(t, err)
			require.True(t, selected)
		} else {
			require.ErrorIs(t, err, ErrResultUnknown)
		}
	}
}
