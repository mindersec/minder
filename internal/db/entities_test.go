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

package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func Test_EntityCrud(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, proj.ID)

	t.Run("CreateEntity", func(t *testing.T) {
		t.Parallel()

		const testRepoName = "testorg/testrepo"

		ent, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
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

		entGet, err := testQueries.GetEntityByID(context.Background(), GetEntityByIDParams{
			ID:       ent.ID,
			Projects: []uuid.UUID{proj.ID},
		})
		require.NoError(t, err)
		require.Equal(t, entGet, ent)

		err = testQueries.DeleteEntity(context.Background(), DeleteEntityParams{
			ID:        ent.ID,
			ProjectID: proj.ID,
		})
		require.NoError(t, err)

		entGet, err = testQueries.GetEntityByID(context.Background(), GetEntityByIDParams{
			ID:       ent.ID,
			Projects: []uuid.UUID{proj.ID},
		})
		require.ErrorIs(t, err, sql.ErrNoRows)
		require.Empty(t, entGet)
	})

	t.Run("No such entity", func(t *testing.T) {
		t.Parallel()

		ent, err := testQueries.GetEntityByName(context.Background(), GetEntityByNameParams{
			ProjectID:  proj.ID,
			Name:       "garbage/nosuchentity",
			EntityType: EntitiesRepository,
		})
		require.ErrorIs(t, err, sql.ErrNoRows)
		require.Empty(t, ent)
	})

	t.Run("Same names different types", func(t *testing.T) {
		t.Parallel()

		const testEntName = "testorg/testent"

		entRepo, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
			EntityType:     EntitiesRepository,
			Name:           testEntName,
			ProjectID:      proj.ID,
			ProviderID:     prov.ID,
			OriginatedFrom: uuid.NullUUID{},
		})
		require.NoError(t, err)
		require.NotEmpty(t, entRepo)
		require.NotEqual(t, entRepo.ID, uuid.Nil)
		require.Equal(t, entRepo.EntityType, EntitiesRepository)
		require.Equal(t, entRepo.Name, testEntName)

		entArtifact, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
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
		require.NotEqual(t, entArtifact.ID, uuid.Nil)
		require.Equal(t, entArtifact.EntityType, EntitiesArtifact)
		require.Equal(t, entArtifact.Name, testEntName)
		require.Equal(t, entArtifact.OriginatedFrom, uuid.NullUUID{
			UUID:  entRepo.ID,
			Valid: true,
		})

		getRepo, err := testQueries.GetEntityByName(context.Background(), GetEntityByNameParams{
			ProjectID:  proj.ID,
			Name:       testEntName,
			EntityType: EntitiesRepository,
		})
		require.NoError(t, err)
		require.NotEmpty(t, getRepo)
		require.Equal(t, getRepo, entRepo)

		getArtifact, err := testQueries.GetEntityByName(context.Background(), GetEntityByNameParams{
			ProjectID:  proj.ID,
			Name:       testEntName,
			EntityType: EntitiesRepository,
		})
		require.NoError(t, err)
		require.NotEmpty(t, getRepo)
		require.Equal(t, getArtifact, entRepo)
	})
}

func Test_PropertyCrud(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, proj.ID)

	t.Run("UpsertProperty", func(t *testing.T) {
		t.Parallel()

		const testRepoName = "testorg/testrepo_props"

		ent, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
			EntityType:     EntitiesRepository,
			Name:           testRepoName,
			ProjectID:      proj.ID,
			ProviderID:     prov.ID,
			OriginatedFrom: uuid.NullUUID{},
		})
		require.NoError(t, err)
		require.NotEmpty(t, ent)

		dbProp, err := testQueries.GetAllPropertyValuesV1(context.Background(), ent.ID)
		require.NoError(t, err)
		require.Empty(t, dbProp)

		prop, err := testQueries.UpsertPropertyValueV1(context.Background(), UpsertPropertyValueV1Params{
			EntityID: ent.ID,
			Key:      "testkey",
			Value:    "testvalue",
		})
		require.NoError(t, err)
		require.NotEmpty(t, prop)

		prop, err = testQueries.UpsertPropertyValueV1(context.Background(), UpsertPropertyValueV1Params{
			EntityID: ent.ID,
			Key:      "anotherkey",
			Value:    "anothervalue",
		})
		require.NoError(t, err)
		require.NotEmpty(t, prop)

		dbProp, err = testQueries.GetAllPropertyValuesV1(context.Background(), ent.ID)
		require.NoError(t, err)
		require.Len(t, dbProp, 2)

		propTestKey := propertyByKey(t, dbProp, "testkey")
		require.Equal(t, propTestKey.Value, "testvalue")
		propAnotherKey := propertyByKey(t, dbProp, "anotherkey")
		require.Equal(t, propAnotherKey.Value, "anothervalue")

		keyVal, err := testQueries.GetPropertyValueV1(context.Background(), ent.ID, "testkey")
		require.NoError(t, err)
		require.Equal(t, keyVal.Value, "testvalue")

		anotherKeyVal, err := testQueries.GetPropertyValueV1(context.Background(), ent.ID, "anotherkey")
		require.NoError(t, err)
		require.Equal(t, anotherKeyVal.Value, "anothervalue")
	})

	t.Run("GetTypedEntitiesByPropertyV1", func(t *testing.T) {
		t.Parallel()

		const testRepoName = "testorg/testrepo_getbyprops"
		const testArtifactName = "testorg/testartifact_getbyprops"

		repo, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
			EntityType:     EntitiesRepository,
			Name:           testRepoName,
			ProjectID:      proj.ID,
			ProviderID:     prov.ID,
			OriginatedFrom: uuid.NullUUID{},
		})
		require.NoError(t, err)
		require.NotEmpty(t, repo)

		_, err = testQueries.UpsertPropertyValueV1(context.Background(), UpsertPropertyValueV1Params{
			EntityID: repo.ID,
			Key:      "sharedkey",
			Value:    "sharedvalue",
		})
		require.NoError(t, err)

		_, err = testQueries.UpsertPropertyValueV1(context.Background(), UpsertPropertyValueV1Params{
			EntityID: repo.ID,
			Key:      "repokey",
			Value:    "repovalue",
		})
		require.NoError(t, err)

		art, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
			EntityType:     EntitiesArtifact,
			Name:           testArtifactName,
			ProjectID:      proj.ID,
			ProviderID:     prov.ID,
			OriginatedFrom: uuid.NullUUID{},
		})
		require.NoError(t, err)
		require.NotEmpty(t, art)

		_, err = testQueries.UpsertPropertyValueV1(context.Background(), UpsertPropertyValueV1Params{
			EntityID: art.ID,
			Key:      "sharedkey",
			Value:    "sharedvalue",
		})
		require.NoError(t, err)

		getEnt, err := testQueries.GetTypedEntitiesByPropertyV1(
			context.Background(), proj.ID, EntitiesRepository, "sharedkey", "sharedvalue")
		require.NoError(t, err)
		require.Len(t, getEnt, 1)
		require.Equal(t, getEnt[0].ID, repo.ID)

		getEnt, err = testQueries.GetTypedEntitiesByPropertyV1(
			context.Background(), proj.ID, EntitiesArtifact, "sharedkey", "sharedvalue")
		require.NoError(t, err)
		require.Len(t, getEnt, 1)
		require.Equal(t, getEnt[0].ID, art.ID)

		getEnt, err = testQueries.GetTypedEntitiesByPropertyV1(
			context.Background(), proj.ID, EntitiesRepository, "repokey", "repovalue")
		require.NoError(t, err)
		require.Len(t, getEnt, 1)
		require.Equal(t, getEnt[0].ID, repo.ID)
	})
}

func propertyByKey(t *testing.T, props []PropertyValueV1, key string) PropertyValueV1 {
	t.Helper()

	for _, prop := range props {
		if prop.Key == key {
			return prop
		}
	}
	return PropertyValueV1{}
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
