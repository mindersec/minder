// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/util/rand"
)

func Test_EntityCrud(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, proj.ID)
	ctx := context.Background()

	t.Run("CreateEntity", func(t *testing.T) {
		t.Parallel()

		const testRepoName = "testorg/testrepo"

		ent, err := testQueries.CreateEntity(ctx, CreateEntityParams{
			EntityType:     EntitiesRepository,
			Name:           testRepoName,
			ProjectID:      proj.ID,
			ProviderID:     prov.ID,
			OriginatedFrom: uuid.NullUUID{},
		})
		require.NoError(t, err)
		require.NotEmpty(t, ent)
		require.NotEqual(t, ent.ID, uuid.Nil)
		require.Equal(t, ent.EntityType, EntitiesRepository)
		require.Equal(t, ent.Name, testRepoName)
		require.Equal(t, ent.ProjectID, proj.ID)
		require.Equal(t, ent.ProviderID, prov.ID)
		require.Equal(t, ent.OriginatedFrom, uuid.NullUUID{})

		entGet, err := testQueries.GetEntityByID(ctx, GetEntityByIDParams{
			ID:         ent.ID,
			ProjectID:  proj.ID,
			ProviderID: prov.ID,
		})
		require.NoError(t, err)
		require.Equal(t, entGet, ent)

		err = testQueries.DeleteEntity(ctx, DeleteEntityParams{
			ID:         ent.ID,
			ProjectID:  proj.ID,
			ProviderID: prov.ID,
		})
		require.NoError(t, err)

		entGet, err = testQueries.GetEntityByID(ctx, GetEntityByIDParams{
			ID:         ent.ID,
			ProjectID:  proj.ID,
			ProviderID: prov.ID,
		})
		require.ErrorIs(t, err, sql.ErrNoRows)
		require.Empty(t, entGet)
	})

	t.Run("No such entity", func(t *testing.T) {
		t.Parallel()

		ent, err := testQueries.GetEntityByName(ctx, GetEntityByNameParams{
			ProjectID:  proj.ID,
			Name:       "garbage/nosuchentity",
			EntityType: EntitiesRepository,
			ProviderID: prov.ID,
		})
		require.ErrorIs(t, err, sql.ErrNoRows)
		require.Empty(t, ent)
	})

	t.Run("Same names different types", func(t *testing.T) {
		t.Parallel()

		const testEntName = "testorg/testent"

		entRepo, err := testQueries.CreateEntity(ctx, CreateEntityParams{
			EntityType:     EntitiesRepository,
			Name:           testEntName,
			ProjectID:      proj.ID,
			ProviderID:     prov.ID,
			OriginatedFrom: uuid.NullUUID{},
		})
		require.NoError(t, err)
		require.NotEmpty(t, entRepo)

		entArtifact, err := testQueries.CreateEntity(ctx, CreateEntityParams{
			EntityType: EntitiesArtifact,
			Name:       testEntName,
			ProjectID:  proj.ID,
			ProviderID: prov.ID,
			OriginatedFrom: uuid.NullUUID{
				UUID:  entRepo.ID,
				Valid: true,
			},
		})
		require.NoError(t, err)
		require.NotEmpty(t, entArtifact)

		getRepo, err := testQueries.GetEntityByName(ctx, GetEntityByNameParams{
			ProjectID:  proj.ID,
			Name:       testEntName,
			EntityType: EntitiesRepository,
			ProviderID: prov.ID,
		})
		require.NoError(t, err)
		require.Equal(t, getRepo, entRepo)

		getArtifact, err := testQueries.GetEntityByName(ctx, GetEntityByNameParams{
			ProjectID:  proj.ID,
			Name:       testEntName,
			EntityType: EntitiesArtifact,
			ProviderID: prov.ID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, getArtifact)
		require.Equal(t, getArtifact, entArtifact)
	})
}

func Test_PropertyCrud(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, proj.ID)
	ctx := context.Background()

	t.Run("UpsertProperty", func(t *testing.T) {
		t.Parallel()

		const testRepoName = "testorg/testrepo_props"

		ent, err := testQueries.CreateEntity(ctx, CreateEntityParams{
			EntityType:     EntitiesRepository,
			Name:           testRepoName,
			ProjectID:      proj.ID,
			ProviderID:     prov.ID,
			OriginatedFrom: uuid.NullUUID{},
		})
		require.NoError(t, err)
		require.NotEmpty(t, ent)

		dbProp, err := testQueries.GetAllPropertiesForEntity(ctx, GetAllPropertiesForEntityParams{
			EntityID:   ent.ID,
			ProjectID:  proj.ID,
			ProviderID: prov.ID,
		})
		require.NoError(t, err)
		require.Empty(t, dbProp)

		prop, err := testQueries.UpsertPropertyValueV1(ctx, UpsertPropertyValueV1Params{
			EntityID: ent.ID,
			Key:      "testkey",
			Value:    "testvalue",
		})
		require.NoError(t, err)
		require.NotEmpty(t, prop)

		prop, err = testQueries.UpsertPropertyValueV1(ctx, UpsertPropertyValueV1Params{
			EntityID: ent.ID,
			Key:      "anotherkey",
			Value:    "anothervalue",
		})
		require.NoError(t, err)
		require.NotEmpty(t, prop)

		dbProp, err = testQueries.GetAllPropertiesForEntity(ctx, GetAllPropertiesForEntityParams{
			EntityID:   ent.ID,
			ProjectID:  proj.ID,
			ProviderID: prov.ID,
		})
		require.NoError(t, err)
		require.Len(t, dbProp, 2)
		require.Equal(t, "testvalue", propertyByKey(t, dbProp, "testkey"))
		require.Equal(t, "anothervalue", propertyByKey(t, dbProp, "anotherkey"))

		keyVal, err := testQueries.GetProperty(ctx, GetPropertyParams{
			EntityID:   ent.ID,
			Key:        "testkey",
			ProjectID:  proj.ID,
			ProviderID: prov.ID,
		})
		require.NoError(t, err)
		value, err := PropValueFromDbV1(keyVal.Value)
		require.NoError(t, err)
		require.Equal(t, "testvalue", value)

		anotherKeyVal, err := testQueries.GetProperty(ctx, GetPropertyParams{
			EntityID:   ent.ID,
			Key:        "anotherkey",
			ProjectID:  proj.ID,
			ProviderID: prov.ID,
		})
		require.NoError(t, err)
		anotherValue, err := PropValueFromDbV1(anotherKeyVal.Value)
		require.NoError(t, err)
		require.Equal(t, "anothervalue", anotherValue)
	})

	t.Run("GetTypedEntitiesByPropertyV1", func(t *testing.T) {
		t.Parallel()

		const testRepoName = "testorg/testrepo_getbyprops"
		const testArtifactName = "testorg/testartifact_getbyprops"

		repo, err := testQueries.CreateEntity(ctx, CreateEntityParams{
			EntityType:     EntitiesRepository,
			Name:           testRepoName,
			ProjectID:      proj.ID,
			ProviderID:     prov.ID,
			OriginatedFrom: uuid.NullUUID{},
		})
		require.NoError(t, err)

		_, err = testQueries.UpsertPropertyValueV1(ctx, UpsertPropertyValueV1Params{
			EntityID: repo.ID,
			Key:      "sharedkey",
			Value:    "sharedvalue",
		})
		require.NoError(t, err)

		_, err = testQueries.UpsertPropertyValueV1(ctx, UpsertPropertyValueV1Params{
			EntityID: repo.ID,
			Key:      "repokey",
			Value:    "repovalue",
		})
		require.NoError(t, err)

		art, err := testQueries.CreateEntity(ctx, CreateEntityParams{
			EntityType:     EntitiesArtifact,
			Name:           testArtifactName,
			ProjectID:      proj.ID,
			ProviderID:     prov.ID,
			OriginatedFrom: uuid.NullUUID{},
		})
		require.NoError(t, err)

		_, err = testQueries.UpsertPropertyValueV1(ctx, UpsertPropertyValueV1Params{
			EntityID: art.ID,
			Key:      "sharedkey",
			Value:    "sharedvalue",
		})
		require.NoError(t, err)

		getEnt, err := testQueries.GetTypedEntitiesByPropertyV1(
			ctx, EntitiesRepository, "sharedkey", "sharedvalue",
			GetTypedEntitiesOptions{
				ProjectID:  proj.ID,
				ProviderID: prov.ID,
			})
		require.NoError(t, err)
		require.Len(t, getEnt, 1)
		require.Equal(t, getEnt[0].ID, repo.ID)

		getEnt, err = testQueries.GetTypedEntitiesByPropertyV1(
			ctx, EntitiesArtifact, "sharedkey", "sharedvalue",
			GetTypedEntitiesOptions{
				ProjectID:  proj.ID,
				ProviderID: prov.ID,
			})
		require.NoError(t, err)
		require.Len(t, getEnt, 1)
		require.Equal(t, getEnt[0].ID, art.ID)

		getEnt, err = testQueries.GetTypedEntitiesByPropertyV1(
			ctx, EntitiesRepository, "repokey", "repovalue",
			GetTypedEntitiesOptions{
				ProjectID:  proj.ID,
				ProviderID: prov.ID,
			})
		require.NoError(t, err)
		require.Len(t, getEnt, 1)
		require.Equal(t, getEnt[0].ID, repo.ID)

		getEnt, err = testQueries.GetTypedEntitiesByPropertyV1(
			ctx, EntitiesRepository, "repokey", "repovalue",
			GetTypedEntitiesOptions{
				ProjectID:  proj.ID,
				ProviderID: prov.ID,
			})
		require.NoError(t, err)
		require.Len(t, getEnt, 1)
		require.Equal(t, getEnt[0].ID, repo.ID)

		getEnt, err = testQueries.GetTypedEntitiesByPropertyV1(
			ctx, EntitiesRepository, "repokey", "repovalue",
			GetTypedEntitiesOptions{
				ProjectID:  proj.ID,
				ProviderID: prov.ID,
			})
		require.NoError(t, err)
		require.Len(t, getEnt, 1)
		require.Equal(t, getEnt[0].ID, repo.ID)

		getEnt, err = testQueries.GetTypedEntitiesByPropertyV1(
			ctx, EntitiesRepository, "repokey", "repovalue",
			GetTypedEntitiesOptions{
				ProjectID:  proj.ID,
				ProviderID: uuid.New(),
			})
		require.NoError(t, err)
		require.Empty(t, getEnt)
	})
}

func propertyByKey(t *testing.T, props []Property, key string) any {
	t.Helper()

	for _, prop := range props {
		if prop.Key == key {
			value, err := PropValueFromDbV1(prop.Value)
			require.NoError(t, err)
			return value
		}
	}
	require.FailNowf(t, "property not found", "missing property with key %q", key)
	return nil
}

func Test_PropertyHelpers(t *testing.T) {
	t.Parallel()

	t.Run("TestFromTo", func(t *testing.T) {
		t.Parallel()

		// TODO: we run into the issue with the large integers here again..
		strValue := "hello world"

		jsonValue, err := PropValueToDbV1(strValue)
		require.NoError(t, err)

		propValue, err := PropValueFromDbV1(jsonValue)
		require.NoError(t, err)
		require.Equal(t, strValue, propValue)
	})

	t.Run("Bad Version", func(t *testing.T) {
		t.Parallel()

		_, err := PropValueFromDbV1([]byte(`{"version": "2", "value": "hello world"}`))
		require.Error(t, err)
		require.ErrorIs(t, err, ErrBadPropVersion)
	})
}

func Test_GetEntitiesByHierarchy(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()
	org := createRandomOrganization(t)

	proj := createRandomProject(t, org.ID)
	proj2 := createRandomProject(t, proj.ID)
	proj3 := createRandomProject(t, proj2.ID)
	proj4 := createRandomProject(t, proj3.ID)

	prov := createRandomProvider(t, proj.ID)
	ctx := context.Background()

	entRepo, err := testQueries.CreateEntity(ctx, CreateEntityParams{
		EntityType:     EntitiesRepository,
		Name:           rand.RandomName(seed),
		ProjectID:      proj.ID,
		ProviderID:     prov.ID,
		OriginatedFrom: uuid.NullUUID{},
	})
	require.NoError(t, err)
	require.NotEmpty(t, entRepo)

	entPkg, err := testQueries.CreateEntity(ctx, CreateEntityParams{
		EntityType:     EntitiesRepository,
		Name:           rand.RandomName(seed),
		ProjectID:      proj2.ID,
		ProviderID:     prov.ID,
		OriginatedFrom: uuid.NullUUID{},
	})
	require.NoError(t, err)
	require.NotEmpty(t, entPkg)

	entPr, err := testQueries.CreateEntity(ctx, CreateEntityParams{
		EntityType:     EntitiesRepository,
		Name:           rand.RandomName(seed),
		ProjectID:      proj3.ID,
		ProviderID:     prov.ID,
		OriginatedFrom: uuid.NullUUID{},
	})
	require.NoError(t, err)
	require.NotEmpty(t, entPr)

	scenarios := []struct {
		name            string
		projects        []uuid.UUID
		expectedNumEnts int
	}{
		{
			name:            "Get all entities",
			projects:        []uuid.UUID{proj.ID, proj2.ID, proj3.ID},
			expectedNumEnts: 3,
		},
		{
			name:            "Get artifact and PR entities",
			projects:        []uuid.UUID{proj2.ID, proj3.ID},
			expectedNumEnts: 2,
		},
		{
			name:            "Get PR only",
			projects:        []uuid.UUID{proj3.ID},
			expectedNumEnts: 1,
		},
		{
			name:            "empty project",
			projects:        []uuid.UUID{proj4.ID},
			expectedNumEnts: 0,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ents, err := testQueries.GetEntitiesByProjectHierarchy(ctx, scenario.projects)
			require.NoError(t, err)
			require.Len(t, ents, scenario.expectedNumEnts)
		})
	}
}
