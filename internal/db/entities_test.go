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

func Test_ListEntitiesOriginatedFrom(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()
	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, proj.ID)

	parent, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
		EntityType:     EntitiesRepository,
		Name:           rand.RandomName(seed),
		ProjectID:      proj.ID,
		ProviderID:     prov.ID,
		OriginatedFrom: uuid.NullUUID{},
	})
	require.NoError(t, err)

	otherParent, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
		EntityType:     EntitiesRepository,
		Name:           rand.RandomName(seed + 1),
		ProjectID:      proj.ID,
		ProviderID:     prov.ID,
		OriginatedFrom: uuid.NullUUID{},
	})
	require.NoError(t, err)

	child1, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
		EntityType: EntitiesArtifact,
		Name:       rand.RandomName(seed + 2),
		ProjectID:  proj.ID,
		ProviderID: prov.ID,
		OriginatedFrom: uuid.NullUUID{
			UUID:  parent.ID,
			Valid: true,
		},
	})
	require.NoError(t, err)

	child2, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
		EntityType: EntitiesPullRequest,
		Name:       rand.RandomName(seed + 3),
		ProjectID:  proj.ID,
		ProviderID: prov.ID,
		OriginatedFrom: uuid.NullUUID{
			UUID:  parent.ID,
			Valid: true,
		},
	})
	require.NoError(t, err)

	// child of a different parent — must not appear in results for parent
	_, err = testQueries.CreateEntity(context.Background(), CreateEntityParams{
		EntityType: EntitiesArtifact,
		Name:       rand.RandomName(seed + 4),
		ProjectID:  proj.ID,
		ProviderID: prov.ID,
		OriginatedFrom: uuid.NullUUID{
			UUID:  otherParent.ID,
			Valid: true,
		},
	})
	require.NoError(t, err)

	t.Run("returns all children when no type filter", func(t *testing.T) {
		t.Parallel()

		children, err := testQueries.ListEntitiesOriginatedFrom(context.Background(), ListEntitiesOriginatedFromParams{
			ParentID:   parent.ID,
			EntityType: NullEntities{},
		})
		require.NoError(t, err)
		require.Len(t, children, 2)

		ids := []uuid.UUID{children[0].ID, children[1].ID}
		require.Contains(t, ids, child1.ID)
		require.Contains(t, ids, child2.ID)
	})

	t.Run("filters by entity type", func(t *testing.T) {
		t.Parallel()

		children, err := testQueries.ListEntitiesOriginatedFrom(context.Background(), ListEntitiesOriginatedFromParams{
			ParentID:   parent.ID,
			EntityType: NullEntities{Entities: EntitiesArtifact, Valid: true},
		})
		require.NoError(t, err)
		require.Len(t, children, 1)
		require.Equal(t, child1.ID, children[0].ID)
	})

	t.Run("excludes children of a different parent", func(t *testing.T) {
		t.Parallel()

		children, err := testQueries.ListEntitiesOriginatedFrom(context.Background(), ListEntitiesOriginatedFromParams{
			ParentID:   parent.ID,
			EntityType: NullEntities{},
		})
		require.NoError(t, err)
		for _, c := range children {
			require.Equal(t, uuid.NullUUID{UUID: parent.ID, Valid: true}, c.OriginatedFrom)
		}
	})

	t.Run("returns empty for entity with no children", func(t *testing.T) {
		t.Parallel()

		childless, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
			EntityType:     EntitiesRepository,
			Name:           rand.RandomName(seed + 5),
			ProjectID:      proj.ID,
			ProviderID:     prov.ID,
			OriginatedFrom: uuid.NullUUID{},
		})
		require.NoError(t, err)

		children, err := testQueries.ListEntitiesOriginatedFrom(context.Background(), ListEntitiesOriginatedFromParams{
			ParentID:   childless.ID,
			EntityType: NullEntities{},
		})
		require.NoError(t, err)
		require.Empty(t, children)
	})
}

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

		entGet, err := testQueries.GetEntityByID(context.Background(), ent.ID)
		require.NoError(t, err)
		require.Equal(t, entGet, ent)

		err = testQueries.DeleteEntity(context.Background(), DeleteEntityParams{
			ID:        ent.ID,
			ProjectID: proj.ID,
		})
		require.NoError(t, err)

		entGet, err = testQueries.GetEntityByID(context.Background(), ent.ID)
		require.ErrorIs(t, err, sql.ErrNoRows)
		require.Empty(t, entGet)
	})

	t.Run("No such entity", func(t *testing.T) {
		t.Parallel()

		ent, err := testQueries.GetEntityByName(context.Background(), GetEntityByNameParams{
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
			ProviderID: prov.ID,
		})
		require.NoError(t, err)
		require.NotEmpty(t, getRepo)
		require.Equal(t, getRepo, entRepo)

		getArtifact, err := testQueries.GetEntityByName(context.Background(), GetEntityByNameParams{
			ProjectID:  proj.ID,
			Name:       testEntName,
			EntityType: EntitiesRepository,
			ProviderID: prov.ID,
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

		dbProp, err := testQueries.GetAllPropertiesForEntity(context.Background(), ent.ID)
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

		dbProp, err = testQueries.GetAllPropertiesForEntity(context.Background(), ent.ID)
		require.NoError(t, err)
		require.Len(t, dbProp, 2)
		require.Equal(t, "testvalue", propertyByKey(t, dbProp, "testkey"))
		require.Equal(t, "anothervalue", propertyByKey(t, dbProp, "anotherkey"))

		keyVal, err := testQueries.GetProperty(context.Background(), GetPropertyParams{
			EntityID: ent.ID,
			Key:      "testkey",
		})
		require.NoError(t, err)
		value, err := PropValueFromDbV1(keyVal.Value)
		require.NoError(t, err)
		require.Equal(t, "testvalue", value)

		anotherKeyVal, err := testQueries.GetProperty(context.Background(), GetPropertyParams{
			EntityID: ent.ID,
			Key:      "anotherkey",
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

		t.Log("Creating repository for GetTypedEntitiesByPropertyV1 test")
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

		t.Log("Creating artifact for GetTypedEntitiesByPropertyV1 test")
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

		t.Log("Get by shared key and repo should return the repository")
		getEnt, err := testQueries.GetTypedEntitiesByPropertyV1(
			context.Background(), EntitiesRepository, "sharedkey", "sharedvalue",
			GetTypedEntitiesOptions{
				ProjectID:  proj.ID,
				ProviderID: prov.ID,
			})
		require.NoError(t, err)
		require.Len(t, getEnt, 1)
		require.Equal(t, getEnt[0].ID, repo.ID)

		t.Log("Get by shared key and artifact should return the artifact")
		getEnt, err = testQueries.GetTypedEntitiesByPropertyV1(
			context.Background(), EntitiesArtifact, "sharedkey", "sharedvalue",
			GetTypedEntitiesOptions{
				ProjectID:  proj.ID,
				ProviderID: prov.ID,
			})
		require.NoError(t, err)
		require.Len(t, getEnt, 1)
		require.Equal(t, getEnt[0].ID, art.ID)

		t.Log("Get by repo key and value should return the repository")
		getEnt, err = testQueries.GetTypedEntitiesByPropertyV1(
			context.Background(), EntitiesRepository, "repokey", "repovalue",
			GetTypedEntitiesOptions{
				ProjectID:  proj.ID,
				ProviderID: prov.ID,
			})
		require.NoError(t, err)
		require.Len(t, getEnt, 1)
		require.Equal(t, getEnt[0].ID, repo.ID)

		t.Log("Get by repo key, value and provider should return the repository")
		getEnt, err = testQueries.GetTypedEntitiesByPropertyV1(
			context.Background(), EntitiesRepository, "repokey", "repovalue",
			GetTypedEntitiesOptions{
				ProviderID: prov.ID,
			})
		require.NoError(t, err)
		require.Len(t, getEnt, 1)
		require.Equal(t, getEnt[0].ID, repo.ID)

		t.Log("Get by repo key, value, project and provider should return the repository")
		getEnt, err = testQueries.GetTypedEntitiesByPropertyV1(
			context.Background(), EntitiesRepository, "repokey", "repovalue",
			GetTypedEntitiesOptions{
				ProjectID:  proj.ID,
				ProviderID: prov.ID,
			})
		require.NoError(t, err)
		require.Len(t, getEnt, 1)
		require.Equal(t, getEnt[0].ID, repo.ID)

		t.Log("Getting by key but with wrong provider should return nothing")
		getEnt, err = testQueries.GetTypedEntitiesByPropertyV1(
			context.Background(), EntitiesRepository, "repokey", "repovalue",
			GetTypedEntitiesOptions{
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

	entRepo, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
		EntityType:     EntitiesRepository,
		Name:           rand.RandomName(seed),
		ProjectID:      proj.ID,
		ProviderID:     prov.ID,
		OriginatedFrom: uuid.NullUUID{},
	})
	require.NoError(t, err)
	require.NotEmpty(t, entRepo)

	entPkg, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
		EntityType:     EntitiesRepository,
		Name:           rand.RandomName(seed),
		ProjectID:      proj2.ID,
		ProviderID:     prov.ID,
		OriginatedFrom: uuid.NullUUID{},
	})
	require.NoError(t, err)
	require.NotEmpty(t, entPkg)

	entPr, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
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

			ents, err := testQueries.GetEntitiesByProjectHierarchy(context.Background(), scenario.projects)
			require.NoError(t, err)
			require.Len(t, ents, scenario.expectedNumEnts)
		})
	}
}

func TestGetEntitiesByTypePaginated(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)
	prov := createRandomProvider(t, proj.ID)

	// Insert 30 repositories so we need 3 full pages at page size 10.
	const total = 30
	created := make([]uuid.UUID, total)
	for i := range created {
		repo := createRandomRepository(t, proj.ID, prov)
		created[i] = repo.ID
	}

	// Page through all results and collect every entity ID returned.
	seen := make(map[uuid.UUID]int)
	cursor := uuid.Nil
	// small page size to exercise multiple pages
	pageSize := int32(10)

	for {
		page, err := testQueries.GetEntitiesByTypePaginated(context.Background(),
			GetEntitiesByTypePaginatedParams{
				EntityType: EntitiesRepository,
				ProviderID: prov.ID,
				Projects:   []uuid.UUID{proj.ID},
				Cursor:     cursor,
				PageSize:   pageSize,
			})
		require.NoError(t, err)

		for _, e := range page {
			seen[e.ID]++
		}

		if len(page) < int(pageSize) {
			// last page
			break
		}
		cursor = page[len(page)-1].ID
	}

	// Every created repo must appear exactly once, no duplicates, no drops.
	for _, id := range created {
		require.Equal(t, 1, seen[id],
			"repo %s appeared %d times, expected exactly 1", id, seen[id])
	}

	// Total seen must equal total created.
	require.Len(t, seen, total,
		"expected %d unique repos, got %d", total, len(seen))
}
