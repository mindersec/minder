// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package selectors

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/db"
	dbf "github.com/mindersec/minder/internal/db/fixtures"
	"github.com/mindersec/minder/internal/entities/models"
	internalpb "github.com/mindersec/minder/internal/proto"
	ghprops "github.com/mindersec/minder/internal/providers/github/properties"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

var (
	githubProvider = db.Provider{
		Name:  "github",
		Class: db.ProviderClassGithubApp,
	}
	gitlabProvider = db.Provider{
		Name:  "gitlab",
		Class: db.ProviderClassGitlab,
	}
)

func withGetProviderByID(result db.Provider, err error) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			GetProviderByID(gomock.Any(), gomock.Any()).
			Return(result, err)
	}
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

func checkProps(t *testing.T, got *structpb.Struct, expected map[string]any) {
	t.Helper()

	gotMap := got.AsMap()
	assert.Equal(t, gotMap, expected)
}

func checkSelEntArtifact(t *testing.T, got, expected *internalpb.SelectorArtifact, propMap map[string]any, expProvider *db.Provider) {
	t.Helper()

	assert.Equal(t, got.Name, expected.Name)
	assert.Equal(t, got.Type, expected.Type)
	assert.Equal(t, got.GetProvider().GetName(), expProvider.Name)
	assert.Equal(t, got.GetProvider().GetClass(), string(expProvider.Class))
	checkProps(t, got.Properties, propMap)
}

func checkSelEntRepo(t *testing.T, got, expected *internalpb.SelectorRepository, propMap map[string]any, expProvider *db.Provider) {
	t.Helper()

	assert.Equal(t, got.Name, expected.Name)
	assert.Equal(t, got.IsFork, expected.IsFork)
	assert.Equal(t, got.GetProvider().GetName(), expProvider.Name)
	assert.Equal(t, got.GetProvider().GetClass(), string(expProvider.Class))
	checkProps(t, got.Properties, propMap)
}

func checkSelEntPullRequest(t *testing.T, got, expected *internalpb.SelectorPullRequest, propMap map[string]any, expProvider *db.Provider) {
	t.Helper()

	assert.Equal(t, got.Name, expected.Name)
	assert.Equal(t, got.GetProvider().GetName(), expProvider.Name)
	assert.Equal(t, got.GetProvider().GetClass(), string(expProvider.Class))
	checkProps(t, got.Properties, propMap)
}

func checkSelEnt(t *testing.T, got, expected *internalpb.SelectorEntity, propMap map[string]any, expProvider *db.Provider) {
	t.Helper()

	assert.Equal(t, got.EntityType, expected.EntityType)
	switch got.EntityType { // nolint:exhaustive
	case minderv1.Entity_ENTITY_REPOSITORIES:
		checkSelEntRepo(t, got.GetRepository(), expected.GetRepository(), propMap, expProvider)
	case minderv1.Entity_ENTITY_ARTIFACTS:
		checkSelEntArtifact(t, got.GetArtifact(), expected.GetArtifact(), propMap, expProvider)
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		checkSelEntPullRequest(t, got.GetPullRequest(), expected.GetPullRequest(), propMap, expProvider)
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
		expDbProv   *db.Provider
		dbSetup     dbf.DBMockBuilder
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
			dbSetup: dbf.NewDBMock(
				withGetProviderByID(githubProvider, nil),
			),
			expDbProv: &githubProvider,
			success:   true,
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
			dbSetup: dbf.NewDBMock(
				withGetProviderByID(githubProvider, nil),
			),
			expDbProv: &githubProvider,
			success:   true,
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
			dbSetup: dbf.NewDBMock(
				withGetProviderByID(gitlabProvider, nil),
			),
			expDbProv: &gitlabProvider,
			success:   true,
		},
		{
			name:       "Repository but no querier provided",
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
			expDbProv: &db.Provider{},
			success:   true,
		},
		{
			name:       "Invalid Entity Type",
			entityType: minderv1.Entity_ENTITY_BUILD,
			entityName: "testorg/testbuild",
			dbSetup:    dbf.NewDBMock(),
			success:    false,
		},
		{
			name:       "Invalid Provider",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			entityName: "testorg/testrepo",
			dbSetup: dbf.NewDBMock(
				withGetProviderByID(db.Provider{}, sql.ErrNoRows),
			),
			expDbProv: &githubProvider,
			success:   false,
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario

		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var mockQuerier db.Store
			if scenario.dbSetup != nil {
				mockQuerier = scenario.dbSetup(ctrl)
			}

			entity := buildEntityWithProperties(scenario.entityType, scenario.entityName, scenario.entityProps)

			selEnt := EntityToSelectorEntity(
				ctx,
				mockQuerier,
				scenario.entityType,
				entity,
			)

			if scenario.success {
				require.NotNil(t, selEnt)
				require.Equal(t, selEnt.GetEntityType(), scenario.entityType)
				require.Equal(t, selEnt.GetName(), scenario.entityName)
				require.Equal(t, selEnt.GetProvider().GetName(), scenario.expDbProv.Name)
				require.Equal(t, selEnt.GetProvider().GetClass(), string(scenario.expDbProv.Class))
				checkSelEnt(t, selEnt, scenario.expSelEnt, scenario.entityProps, scenario.expDbProv)
			} else {
				require.Nil(t, selEnt)
			}
		})
	}
}
