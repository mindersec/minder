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
