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

package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/db/embedded"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/entities/models"
	"github.com/stacklok/minder/internal/entities/properties"
	mock_github "github.com/stacklok/minder/internal/providers/github/mock"
	ghprop "github.com/stacklok/minder/internal/providers/github/properties"
	"github.com/stacklok/minder/internal/util/rand"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

type githubMockBuilder func(*gomock.Controller) *mock_github.MockGitHub

func newGithubMock(opts ...func(mock *mock_github.MockGitHub)) githubMockBuilder {
	return func(ctrl *gomock.Controller) *mock_github.MockGitHub {
		mock := mock_github.NewMockGitHub(ctrl)
		for _, opt := range opts {
			opt(mock)
		}
		return mock
	}
}

func withUpstreamRepoProperties(repoProperties map[string]any, entType minderv1.Entity) func(mock *mock_github.MockGitHub) {
	return func(mock *mock_github.MockGitHub) {
		props, err := properties.NewProperties(repoProperties)
		if err != nil {
			panic(err)
		}
		mock.EXPECT().
			FetchAllProperties(gomock.Any(), gomock.Any(), entType, gomock.Any()).
			Return(props, nil)
	}
}

func withUpstreamRepoProperty(key string, val any, entType minderv1.Entity) func(mock *mock_github.MockGitHub) {
	return func(mock *mock_github.MockGitHub) {
		prop, err := properties.NewProperty(val)
		if err != nil {
			panic(err)
		}
		mock.EXPECT().
			FetchProperty(gomock.Any(), gomock.Any(), entType, key).
			Return(prop, nil)
	}
}

func insertProperties(ctx context.Context, t *testing.T, store db.Store, entID uuid.UUID, props *properties.Properties) {
	t.Helper()

	for key, prop := range props.Iterate() {
		_, err := store.UpsertPropertyValueV1(ctx, db.UpsertPropertyValueV1Params{
			EntityID: entID,
			Key:      key,
			Value:    prop.RawValue(),
		})
		require.NoError(t, err)
	}
}

func insertPropertiesFromMap(ctx context.Context, t *testing.T, store db.Store, entID uuid.UUID, propMap map[string]any) {
	t.Helper()

	for key, val := range propMap {
		_, err := store.UpsertPropertyValueV1(ctx, db.UpsertPropertyValueV1Params{
			EntityID: entID,
			Key:      key,
			Value:    val,
		})
		require.NoError(t, err)
	}
}

type fetchParams struct {
	entType minderv1.Entity
	entName string

	providerID uuid.UUID
	projectID  uuid.UUID

	other map[string]any
}

type testCtx struct {
	testQueries   db.Store
	dbProj        db.Project
	ghAppProvider db.Provider
}

func createTestCtx(ctx context.Context, t *testing.T) testCtx {
	t.Helper()

	testQueries, td, err := embedded.GetFakeStore()
	require.NoError(t, err, "expected no error when creating embedded store")
	t.Cleanup(td)

	seed := time.Now().UnixNano()
	dbProj, err := testQueries.CreateProject(ctx, db.CreateProjectParams{
		Name:     rand.RandomName(seed),
		Metadata: []byte(`{}`),
	})
	require.NoError(t, err)

	ghAppProvider, err := testQueries.CreateProvider(context.Background(),
		db.CreateProviderParams{
			Name:       rand.RandomName(seed),
			ProjectID:  dbProj.ID,
			Class:      db.ProviderClassGithubApp,
			Implements: []db.ProviderType{db.ProviderTypeGithub, db.ProviderTypeGit},
			AuthFlows:  []db.AuthorizationFlow{db.AuthorizationFlowUserInput},
			Definition: json.RawMessage("{}"),
		})
	require.NoError(t, err)

	return testCtx{
		testQueries:   testQueries,
		dbProj:        dbProj,
		ghAppProvider: ghAppProvider,
	}
}

func TestPropertiesService_SaveProperty(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()

	scenarios := []struct {
		name    string
		key     string
		val     any
		dbSetup func(t *testing.T, entityID uuid.UUID, store db.Store)
		checkFn func(t *testing.T, props *properties.Property)
	}{
		{
			name: "Save a new property",
			dbSetup: func(t *testing.T, entityID uuid.UUID, store db.Store) {
				t.Helper()

				propMap := map[string]any{
					properties.RepoPropertyIsPrivate: true,
					ghprop.RepoPropertyId:            123,
				}
				insertPropertiesFromMap(context.TODO(), t, store, entityID, propMap)
			},
			key: properties.RepoPropertyIsFork,
			val: true,
			checkFn: func(t *testing.T, props *properties.Property) {
				t.Helper()

				require.Equal(t, props.GetBool(), true)
			},
		},
		{
			name: "Update an existing property",
			dbSetup: func(t *testing.T, entityID uuid.UUID, store db.Store) {
				t.Helper()

				propMap := map[string]any{
					properties.RepoPropertyIsPrivate: true,
					ghprop.RepoPropertyId:            int64(123),
				}
				insertPropertiesFromMap(context.TODO(), t, store, entityID, propMap)
			},
			key: ghprop.RepoPropertyId,
			val: int64(456),
			checkFn: func(t *testing.T, props *properties.Property) {
				t.Helper()

				require.Equal(t, props.GetInt64(), int64(456))
			},
		},
		{
			name: "The property no longer exists",
			dbSetup: func(t *testing.T, entityID uuid.UUID, store db.Store) {
				t.Helper()

				propMap := map[string]any{
					properties.RepoPropertyIsPrivate: true,
					ghprop.RepoPropertyId:            123,
				}
				insertPropertiesFromMap(context.TODO(), t, store, entityID, propMap)
			},
			key: properties.RepoPropertyIsPrivate,
			val: nil,
			checkFn: func(t *testing.T, props *properties.Property) {
				t.Helper()

				require.Nil(t, props)
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	tctx := createTestCtx(ctx, t)

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			ent, err := tctx.testQueries.CreateEntity(ctx, db.CreateEntityParams{
				EntityType: entities.EntityTypeToDB(minderv1.Entity_ENTITY_REPOSITORIES),
				Name:       rand.RandomName(seed),
				ProjectID:  tctx.dbProj.ID,
				ProviderID: tctx.ghAppProvider.ID,
			})

			require.NoError(t, err)
			propSvc := NewPropertiesService(tctx.testQueries)

			var prop *properties.Property
			if tt.val != nil {
				prop, err = properties.NewProperty(tt.val)
				require.NoError(t, err)
			}

			err = tctx.testQueries.WithTransactionErr(func(qtx db.ExtendQuerier) error {
				return propSvc.ReplaceProperty(ctx, ent.ID, tt.key, prop,
					CallBuilder().WithStoreOrTransaction(qtx))
			})
			require.NoError(t, err)

			dbProp, err := tctx.testQueries.GetProperty(ctx, db.GetPropertyParams{
				EntityID: ent.ID,
				Key:      tt.key,
			})
			if tt.val == nil {
				require.ErrorIs(t, err, sql.ErrNoRows)
				return
			}

			require.NoError(t, err)
			updatedProp, err := models.DbPropToModel(dbProp)
			require.NoError(t, err)
			tt.checkFn(t, updatedProp)
		})
	}
}

func TestPropertiesService_SaveAllProperties(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()

	scenarios := []struct {
		name    string
		dbSetup func(t *testing.T, entityID uuid.UUID, store db.Store)
		props   map[string]any
		checkFn func(t *testing.T, props *properties.Properties)
	}{
		{
			name: "Replace all properties",
			dbSetup: func(t *testing.T, entityID uuid.UUID, store db.Store) {
				t.Helper()

				propMap := map[string]any{
					properties.RepoPropertyIsPrivate: true,
					ghprop.RepoPropertyId:            int64(123),
				}
				insertPropertiesFromMap(context.TODO(), t, store, entityID, propMap)
			},
			props: map[string]any{
				properties.RepoPropertyIsPrivate: false,
				ghprop.RepoPropertyId:            int64(456),
			},
			checkFn: func(t *testing.T, props *properties.Properties) {
				t.Helper()

				require.Equal(t, props.GetProperty(properties.RepoPropertyIsPrivate).GetBool(), false)
				require.Equal(t, props.GetProperty(ghprop.RepoPropertyId).GetInt64(), int64(456))
			},
		},
		{
			name: "One less property upstream",
			dbSetup: func(t *testing.T, entityID uuid.UUID, store db.Store) {
				t.Helper()

				propMap := map[string]any{
					properties.RepoPropertyIsPrivate: true,
					ghprop.RepoPropertyId:            int64(123),
				}
				insertPropertiesFromMap(context.TODO(), t, store, entityID, propMap)
			},
			props: map[string]any{
				ghprop.RepoPropertyId: int64(456),
			},
			checkFn: func(t *testing.T, props *properties.Properties) {
				t.Helper()

				require.Nil(t, props.GetProperty(properties.RepoPropertyIsPrivate))
				require.Equal(t, props.GetProperty(ghprop.RepoPropertyId).GetInt64(), int64(456))
			},
		},
		{
			name: "One more property upstream",
			dbSetup: func(t *testing.T, entityID uuid.UUID, store db.Store) {
				t.Helper()

				propMap := map[string]any{
					properties.RepoPropertyIsPrivate: true,
					ghprop.RepoPropertyId:            int64(123),
				}
				insertPropertiesFromMap(context.TODO(), t, store, entityID, propMap)
			},
			props: map[string]any{
				properties.RepoPropertyIsPrivate: false,
				properties.RepoPropertyIsFork:    true,
				ghprop.RepoPropertyId:            int64(456),
			},
			checkFn: func(t *testing.T, props *properties.Properties) {
				t.Helper()

				require.Equal(t, props.GetProperty(properties.RepoPropertyIsPrivate).GetBool(), false)
				require.Equal(t, props.GetProperty(properties.RepoPropertyIsFork).GetBool(), true)
				require.Equal(t, props.GetProperty(ghprop.RepoPropertyId).GetInt64(), int64(456))
			},
		},
		{
			name: "No properties upstream",
			dbSetup: func(t *testing.T, entityID uuid.UUID, store db.Store) {
				t.Helper()

				propMap := map[string]any{
					properties.RepoPropertyIsPrivate: true,
					ghprop.RepoPropertyId:            123,
				}
				insertPropertiesFromMap(context.TODO(), t, store, entityID, propMap)
			},
			props: map[string]any{},
			checkFn: func(t *testing.T, props *properties.Properties) {
				t.Helper()

				count := 0
				for range props.Iterate() {
					count++
				}
				require.Equal(t, count, 0)
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	tctx := createTestCtx(ctx, t)

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			ent, err := tctx.testQueries.CreateEntity(ctx, db.CreateEntityParams{
				EntityType: entities.EntityTypeToDB(minderv1.Entity_ENTITY_REPOSITORIES),
				Name:       rand.RandomName(seed),
				ProjectID:  tctx.dbProj.ID,
				ProviderID: tctx.ghAppProvider.ID,
			})

			require.NoError(t, err)
			propSvc := NewPropertiesService(tctx.testQueries)

			props, err := properties.NewProperties(tt.props)
			require.NoError(t, err)

			err = tctx.testQueries.WithTransactionErr(func(qtx db.ExtendQuerier) error {
				return propSvc.ReplaceAllProperties(ctx, ent.ID, props,
					CallBuilder().WithStoreOrTransaction(qtx))
			})
			require.NoError(t, err)

			dbProps, err := tctx.testQueries.GetAllPropertiesForEntity(ctx, ent.ID)
			require.NoError(t, err)

			updatedProps, err := models.DbPropsToModel(dbProps)
			require.NoError(t, err)
			tt.checkFn(t, updatedProps)
		})
	}
}

func TestPropertiesService_RetrieveProperty(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()

	scenarios := []struct {
		name        string
		propName    string
		dbSetup     func(t *testing.T, store db.Store, params fetchParams)
		githubSetup func(params fetchParams) githubMockBuilder
		params      fetchParams
		expectErr   string
		checkResult func(t *testing.T, props *properties.Property)
		opts        []propertiesServiceOption
	}{
		{
			name:     "No cache, fetch from provider",
			propName: properties.RepoPropertyIsPrivate,
			dbSetup: func(_ *testing.T, _ db.Store, _ fetchParams) {
			},
			githubSetup: func(params fetchParams) githubMockBuilder {
				return newGithubMock(
					withUpstreamRepoProperty(properties.RepoPropertyIsPrivate, true, params.entType),
				)
			},
			params: fetchParams{
				entType: minderv1.Entity_ENTITY_REPOSITORIES,
				entName: rand.RandomName(seed),
			},
			checkResult: func(t *testing.T, prop *properties.Property) {
				t.Helper()
				require.Equal(t, prop.GetBool(), true)
			},
		},
		{
			name:     "Cache miss, fetch from provider",
			propName: ghprop.RepoPropertyId,
			dbSetup: func(t *testing.T, store db.Store, params fetchParams) {
				t.Helper()
				ent, err := store.CreateEntity(context.TODO(), db.CreateEntityParams{
					EntityType: entities.EntityTypeToDB(params.entType),
					Name:       params.entName,
					ProjectID:  params.projectID,
					ProviderID: params.providerID,
				})
				require.NoError(t, err)

				// these are different than tt.params.properties
				oldPropMap := map[string]any{
					ghprop.RepoPropertyId: int64(1234),
				}
				insertPropertiesFromMap(context.TODO(), t, store, ent.ID, oldPropMap)
			},
			githubSetup: func(params fetchParams) githubMockBuilder {
				t.Helper()
				return newGithubMock(
					withUpstreamRepoProperty(ghprop.RepoPropertyId, int64(123), params.entType),
				)
			},
			params: fetchParams{
				entType: minderv1.Entity_ENTITY_REPOSITORIES,
				entName: rand.RandomName(seed),
			},
			checkResult: func(t *testing.T, prop *properties.Property) {
				t.Helper()
				require.Equal(t, prop.GetInt64(), int64(123))
			},
			opts: []propertiesServiceOption{
				WithEntityTimeout(bypassCacheTimeout),
			},
		},
		{
			name:     "Cache hit by name, fetch from cache",
			propName: ghprop.RepoPropertyId,
			dbSetup: func(t *testing.T, store db.Store, params fetchParams) {
				t.Helper()

				ent, err := store.CreateEntity(context.TODO(), db.CreateEntityParams{
					EntityType: entities.EntityTypeToDB(params.entType),
					Name:       params.entName,
					ProjectID:  params.projectID,
					ProviderID: params.providerID,
				})
				require.NoError(t, err)

				propMap := map[string]any{
					ghprop.RepoPropertyId: int64(123),
				}
				props, err := properties.NewProperties(propMap)
				require.NoError(t, err)
				insertProperties(context.TODO(), t, store, ent.ID, props)
			},
			githubSetup: func(_ fetchParams) githubMockBuilder {
				return newGithubMock()
			},
			params: fetchParams{
				entType: minderv1.Entity_ENTITY_REPOSITORIES,
				entName: rand.RandomName(seed),
			},
			checkResult: func(t *testing.T, prop *properties.Property) {
				t.Helper()

				require.Equal(t, prop.GetInt64(), int64(123))
			},
		},
		{
			name:     "Cache hit by upstream ID, fetch from cache",
			propName: properties.RepoPropertyIsArchived,
			dbSetup: func(t *testing.T, store db.Store, params fetchParams) {
				t.Helper()

				ent, err := store.CreateEntity(context.TODO(), db.CreateEntityParams{
					EntityType: entities.EntityTypeToDB(params.entType),
					Name:       params.entName,
					ProjectID:  params.projectID,
					ProviderID: params.providerID,
				})
				require.NoError(t, err)

				propMap := map[string]any{
					properties.PropertyUpstreamID:     "this is an upstream ID",
					properties.RepoPropertyIsArchived: true,
				}
				props, err := properties.NewProperties(propMap)
				require.NoError(t, err)
				insertProperties(context.TODO(), t, store, ent.ID, props)
			},
			githubSetup: func(_ fetchParams) githubMockBuilder {
				return newGithubMock()
			},
			params: fetchParams{
				entType: minderv1.Entity_ENTITY_REPOSITORIES,
				other: map[string]any{
					properties.PropertyUpstreamID: "this is an upstream ID",
				},
			},
			checkResult: func(t *testing.T, prop *properties.Property) {
				t.Helper()

				// This is checking for IsArchived
				require.Equal(t, prop.GetBool(), true)
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	tctx := createTestCtx(ctx, t)

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			tt.params.providerID = tctx.ghAppProvider.ID
			tt.params.projectID = tctx.dbProj.ID

			githubSetup := tt.githubSetup(tt.params)
			githubMock := githubSetup(ctrl)

			tt.dbSetup(t, tctx.testQueries, tt.params)

			propSvc := NewPropertiesService(tctx.testQueries, tt.opts...)

			propSearch := map[string]any{}
			if tt.params.entName == "" {
				propSearch[properties.PropertyUpstreamID] = tt.params.other[properties.PropertyUpstreamID]
			} else {
				propSearch[properties.PropertyName] = tt.params.entName
			}
			getByProps, err := properties.NewProperties(propSearch)
			require.NoError(t, err)

			gotProps, err := propSvc.RetrieveProperty(
				ctx, githubMock, tctx.dbProj.ID, tctx.ghAppProvider.ID, getByProps, tt.params.entType, tt.propName, nil)

			if tt.expectErr != "" {
				require.Contains(t, err.Error(), tt.expectErr)
				return
			}

			require.NoError(t, err)
			tt.checkResult(t, gotProps)
		})
	}
}

func TestPropertiesService_EntityWithPropertiesByUpstreamID(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	tctx := createTestCtx(ctx, t)

	scenarios := []struct {
		name          string
		entType       minderv1.Entity
		upstreamID    string
		hint          ByUpstreamIdHint
		dbSetup       func(t *testing.T, store db.Store)
		expectedError error
		checkResult   func(t *testing.T, result *models.EntityWithProperties)
	}{
		{
			name:       "Successful retrieval",
			entType:    minderv1.Entity_ENTITY_REPOSITORIES,
			upstreamID: "123",
			hint: ByUpstreamIdHint{
				projectID:  tctx.dbProj.ID,
				providerID: tctx.ghAppProvider.ID,
			},
			dbSetup: func(t *testing.T, store db.Store) {
				t.Helper()
				ent, err := store.CreateEntity(ctx, db.CreateEntityParams{
					EntityType: entities.EntityTypeToDB(minderv1.Entity_ENTITY_REPOSITORIES),
					Name:       "test-repo",
					ProjectID:  tctx.dbProj.ID,
					ProviderID: tctx.ghAppProvider.ID,
				})
				require.NoError(t, err)

				propMap := map[string]any{
					properties.PropertyUpstreamID:    "123",
					properties.PropertyName:          "test-repo",
					properties.RepoPropertyIsPrivate: true,
				}
				insertPropertiesFromMap(ctx, t, store, ent.ID, propMap)
			},
			checkResult: func(t *testing.T, result *models.EntityWithProperties) {
				t.Helper()
				require.NotNil(t, result)
				require.Equal(t, "test-repo", result.Entity.Name)
				require.Equal(t, "123", result.Properties.GetProperty(properties.PropertyUpstreamID).GetString())
				require.True(t, result.Properties.GetProperty(properties.RepoPropertyIsPrivate).GetBool())
			},
		},
		{
			name:       "Entity not found",
			entType:    minderv1.Entity_ENTITY_REPOSITORIES,
			upstreamID: "456",
			hint: ByUpstreamIdHint{
				projectID:  tctx.dbProj.ID,
				providerID: tctx.ghAppProvider.ID,
			},
			expectedError: ErrEntityNotFound,
		},
		{
			name:       "Multiple entities returned",
			entType:    minderv1.Entity_ENTITY_REPOSITORIES,
			upstreamID: "789",
			hint: ByUpstreamIdHint{
				ProviderImplements: db.NullProviderType{
					ProviderType: db.ProviderTypeGithub,
					Valid:        true,
				},
			},
			dbSetup: func(t *testing.T, store db.Store) {
				t.Helper()
				for i := 0; i < 2; i++ {
					ent, err := store.CreateEntity(ctx, db.CreateEntityParams{
						EntityType: entities.EntityTypeToDB(minderv1.Entity_ENTITY_REPOSITORIES),
						Name:       fmt.Sprintf("test-repo-%d", i),
						ProjectID:  tctx.dbProj.ID,
						ProviderID: tctx.ghAppProvider.ID,
					})
					require.NoError(t, err)

					propMap := map[string]any{
						properties.PropertyUpstreamID: "789",
						properties.PropertyName:       fmt.Sprintf("test-repo-%d", i),
					}
					insertPropertiesFromMap(ctx, t, store, ent.ID, propMap)
				}
			},
			expectedError: ErrMultipleEntities,
		},
		{
			name:       "Property hint mismatch",
			entType:    minderv1.Entity_ENTITY_REPOSITORIES,
			upstreamID: "101112",
			hint: ByUpstreamIdHint{
				projectID:  tctx.dbProj.ID,
				providerID: tctx.ghAppProvider.ID,
				PropName:   properties.RepoPropertyIsPrivate,
				PropValue:  true,
			},
			dbSetup: func(t *testing.T, store db.Store) {
				t.Helper()
				ent, err := store.CreateEntity(ctx, db.CreateEntityParams{
					EntityType: entities.EntityTypeToDB(minderv1.Entity_ENTITY_REPOSITORIES),
					Name:       "test-repo-mismatch",
					ProjectID:  tctx.dbProj.ID,
					ProviderID: tctx.ghAppProvider.ID,
				})
				require.NoError(t, err)

				propMap := map[string]any{
					properties.PropertyUpstreamID:    "101112",
					properties.PropertyName:          "test-repo-mismatch",
					properties.RepoPropertyIsPrivate: false,
				}
				insertPropertiesFromMap(ctx, t, store, ent.ID, propMap)
			},
			expectedError: ErrEntityNotFound,
		},
	}

	for _, tt := range scenarios {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.dbSetup != nil {
				tt.dbSetup(t, tctx.testQueries)
			}

			propSvc := NewPropertiesService(tctx.testQueries)

			result, err := propSvc.EntityWithPropertiesByUpstreamID(ctx, tt.entType, tt.upstreamID, tt.hint, nil)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
				tt.checkResult(t, result)
			}
		})
	}
}

func TestPropertiesService_RetrieveAllProperties(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()

	scenarios := []struct {
		name        string
		dbSetup     func(t *testing.T, store db.Store, params fetchParams)
		githubSetup func(t *testing.T, params fetchParams) githubMockBuilder
		params      fetchParams
		lookupProps map[string]any
		expectErr   string
		checkResult func(t *testing.T, props *properties.Properties)
		opts        []propertiesServiceOption
	}{
		{
			name: "No cache, fetch from provider",
			dbSetup: func(_ *testing.T, _ db.Store, _ fetchParams) {
			},
			githubSetup: func(t *testing.T, params fetchParams) githubMockBuilder {
				t.Helper()

				propMap := map[string]any{
					properties.RepoPropertyIsPrivate: true,
					ghprop.RepoPropertyId:            int64(123),
				}
				return newGithubMock(
					withUpstreamRepoProperties(propMap, params.entType),
				)
			},
			params: fetchParams{
				entType: minderv1.Entity_ENTITY_REPOSITORIES,
				entName: rand.RandomName(seed),
			},
			lookupProps: map[string]any{
				properties.PropertyUpstreamID: "123",
				properties.PropertyName:       rand.RandomName(seed),
			},
			checkResult: func(t *testing.T, props *properties.Properties) {
				t.Helper()

				require.Equal(t, props.GetProperty(properties.RepoPropertyIsPrivate).GetBool(), true)
				require.Equal(t, props.GetProperty(ghprop.RepoPropertyId).GetInt64(), int64(123))
			},
		},
		{
			name: "Cache miss, fetch from provider",
			dbSetup: func(t *testing.T, store db.Store, params fetchParams) {
				t.Helper()

				ent, err := store.CreateEntity(context.TODO(), db.CreateEntityParams{
					EntityType: entities.EntityTypeToDB(params.entType),
					Name:       params.entName,
					ProjectID:  params.projectID,
					ProviderID: params.providerID,
				})
				require.NoError(t, err)

				// these are different than the returned properties in github setup which we also
				// check for
				oldPropMap := map[string]any{
					properties.RepoPropertyIsPrivate: true,
					ghprop.RepoPropertyId:            int64(1234),
				}
				insertPropertiesFromMap(context.TODO(), t, store, ent.ID, oldPropMap)
			},
			githubSetup: func(t *testing.T, params fetchParams) githubMockBuilder {
				t.Helper()

				propMap := map[string]any{
					properties.RepoPropertyIsPrivate: true,
					ghprop.RepoPropertyId:            int64(123),
				}
				return newGithubMock(
					withUpstreamRepoProperties(propMap, params.entType),
				)
			},
			params: fetchParams{
				entType: minderv1.Entity_ENTITY_REPOSITORIES,
				entName: rand.RandomName(seed),
			},
			lookupProps: map[string]any{
				properties.PropertyName: rand.RandomName(seed),
			},
			checkResult: func(t *testing.T, props *properties.Properties) {
				t.Helper()

				require.Equal(t, props.GetProperty(properties.RepoPropertyIsPrivate).GetBool(), true)
				require.Equal(t, props.GetProperty(ghprop.RepoPropertyId).GetInt64(), int64(123))
			},
			opts: []propertiesServiceOption{
				WithEntityTimeout(bypassCacheTimeout),
			},
		},
		{
			name: "Cache hit by name, fetch from cache",
			dbSetup: func(t *testing.T, store db.Store, params fetchParams) {
				t.Helper()

				ent, err := store.CreateEntity(context.TODO(), db.CreateEntityParams{
					EntityType: entities.EntityTypeToDB(params.entType),
					Name:       params.entName,
					ProjectID:  params.projectID,
					ProviderID: params.providerID,
				})
				require.NoError(t, err)

				propMap := map[string]any{
					properties.RepoPropertyIsPrivate: true,
					ghprop.RepoPropertyId:            int64(123),
				}
				props, err := properties.NewProperties(propMap)
				require.NoError(t, err)
				insertProperties(context.TODO(), t, store, ent.ID, props)
			},
			githubSetup: func(t *testing.T, _ fetchParams) githubMockBuilder {
				t.Helper()

				return newGithubMock()
			},
			params: fetchParams{
				entType: minderv1.Entity_ENTITY_REPOSITORIES,
				entName: "testorg/testrepo",
			},
			lookupProps: map[string]any{
				properties.PropertyName: "testorg/testrepo",
			},
			checkResult: func(t *testing.T, props *properties.Properties) {
				t.Helper()

				require.Equal(t, props.GetProperty(properties.RepoPropertyIsPrivate).GetBool(), true)
				require.Equal(t, props.GetProperty(ghprop.RepoPropertyId).GetInt64(), int64(123))
			},
		},
		{
			name: "Cache hit by upstream ID, fetch from cache",
			dbSetup: func(t *testing.T, store db.Store, params fetchParams) {
				t.Helper()

				ent, err := store.CreateEntity(context.TODO(), db.CreateEntityParams{
					EntityType: entities.EntityTypeToDB(params.entType),
					Name:       params.entName,
					ProjectID:  params.projectID,
					ProviderID: params.providerID,
				})
				require.NoError(t, err)

				propMap := map[string]any{
					properties.PropertyUpstreamID:    "456",
					properties.RepoPropertyIsPrivate: true,
					ghprop.RepoPropertyId:            int64(456),
				}
				props, err := properties.NewProperties(propMap)
				require.NoError(t, err)
				insertProperties(context.TODO(), t, store, ent.ID, props)
			},
			githubSetup: func(t *testing.T, _ fetchParams) githubMockBuilder {
				t.Helper()

				return newGithubMock()
			},
			params: fetchParams{
				entType: minderv1.Entity_ENTITY_REPOSITORIES,
				entName: "testorg/testrepo2",
			},
			lookupProps: map[string]any{
				// no name here, we're just looking up by upstream ID
				properties.PropertyUpstreamID: "456",
			},
			checkResult: func(t *testing.T, props *properties.Properties) {
				t.Helper()

				require.Equal(t, props.GetProperty(properties.RepoPropertyIsPrivate).GetBool(), true)
				require.Equal(t, props.GetProperty(ghprop.RepoPropertyId).GetInt64(), int64(456))
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	tctx := createTestCtx(ctx, t)

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			tt.params.providerID = tctx.ghAppProvider.ID
			tt.params.projectID = tctx.dbProj.ID

			githubSetup := tt.githubSetup(t, tt.params)
			githubMock := githubSetup(ctrl)

			tt.dbSetup(t, tctx.testQueries, tt.params)

			propSvc := NewPropertiesService(tctx.testQueries, tt.opts...)

			getByProps, err := properties.NewProperties(tt.lookupProps)
			require.NoError(t, err)

			gotProps, err := propSvc.RetrieveAllProperties(
				ctx, githubMock, tctx.dbProj.ID, tctx.ghAppProvider.ID, getByProps, tt.params.entType,
				ReadBuilder().WithStoreOrTransaction(tctx.testQueries))

			if tt.expectErr != "" {
				require.Contains(t, err.Error(), tt.expectErr)
				return
			}

			require.NoError(t, err)
			tt.checkResult(t, gotProps)
		})
	}
}

func TestPropertiesService_EntityWithProperties(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	scenarios := []struct {
		name           string
		entityID       uuid.UUID
		entName        string
		dbEntBuilder   func(id uuid.UUID, entName string) db.EntityInstance
		dbPropsBuilder func(id uuid.UUID) []db.Property
		checkProps     func(t *testing.T, props *properties.Properties)
	}{
		{
			name:     "Entity with properties",
			entityID: uuid.New(),
			entName:  "myorg/the-props-are-different",
			dbEntBuilder: func(id uuid.UUID, entName string) db.EntityInstance {
				return db.EntityInstance{
					ID:   id,
					Name: entName,
				}
			},
			dbPropsBuilder: func(id uuid.UUID) []db.Property {
				return []db.Property{
					{
						EntityID: id,
						Key:      "name",
						Value:    []byte(`{"value": "myorg/bad-go", "version": "v1"}`),
					},
					{
						EntityID: id,
						Key:      "is_private",
						Value:    []byte(`{"value": false, "version": "v1"}`),
					},
				}
			},
			checkProps: func(t *testing.T, props *properties.Properties) {
				t.Helper()

				require.Equal(t, props.GetProperty("name").GetString(), "myorg/bad-go")
				require.Equal(t, props.GetProperty("is_private").GetBool(), false)
			},
		},
		{
			name:     "Entity without properties",
			entityID: uuid.New(),
			entName:  "myorg/noprops",
			dbEntBuilder: func(id uuid.UUID, entName string) db.EntityInstance {
				return db.EntityInstance{
					ID:   id,
					Name: entName,
				}
			},
			dbPropsBuilder: func(_ uuid.UUID) []db.Property {
				return []db.Property{}
			},
			checkProps: func(t *testing.T, props *properties.Properties) {
				t.Helper()

				require.Equal(t, props.GetProperty("name").GetString(), "myorg/noprops")
			},
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockDB := mockdb.NewMockStore(ctrl)

			mockDB.EXPECT().
				GetEntityByID(ctx, tt.entityID).
				Return(tt.dbEntBuilder(tt.entityID, tt.entName), nil)
			mockDB.EXPECT().
				GetAllPropertiesForEntity(ctx, tt.entityID).
				Return(tt.dbPropsBuilder(tt.entityID), nil)

			ps := NewPropertiesService(mockDB)
			result, err := ps.EntityWithPropertiesByID(ctx, tt.entityID, nil)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, result.Entity.ID, tt.entityID)
			require.Equal(t, result.Entity.Name, tt.entName)
			tt.checkProps(t, result.Properties)
		})
	}
}
