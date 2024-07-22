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

func TestCreateSelector(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		org := createRandomOrganization(t)
		proj := createRandomProject(t, org.ID)
		prof := createRandomProfile(t, proj.ID, []string{})

		sel, err := testQueries.CreateSelector(context.Background(), CreateSelectorParams{
			ProfileID: prof.ID,
			Entity:    NullEntities{},
			Selector:  "entity.name.startsWith(\"foo\") && !entity.name.startsWith(\"foobar\")",
			Comment:   "no foobar allowed, only foo",
		})
		require.NoError(t, err)
		require.NotEmpty(t, sel)
		require.NotEqual(t, sel.ID, uuid.Nil)
		require.Equal(t, sel.ProfileID, prof.ID)
		require.Equal(t, sel.Entity, NullEntities{})
		require.Equal(t, sel.Selector, "entity.name.startsWith(\"foo\") && !entity.name.startsWith(\"foobar\")")
		require.Equal(t, sel.Comment, "no foobar allowed, only foo")
	})

	t.Run("No such project", func(t *testing.T) {
		t.Parallel()

		uuid, err := uuid.NewRandom()
		require.NoError(t, err)

		sel, err := testQueries.CreateSelector(context.Background(), CreateSelectorParams{
			ProfileID: uuid,
			Entity:    NullEntities{},
			Selector:  "entity.name.startsWith(\"foo\") && !entity.name.startsWith(\"foobar\")",
			Comment:   "no foobar allowed, only foo",
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "violates foreign key constraint")
		require.Empty(t, sel)
	})
}

func TestUpdateSelector(t *testing.T) {
	t.Parallel()

	t.Run("Update existing", func(t *testing.T) {
		t.Parallel()

		org := createRandomOrganization(t)
		proj := createRandomProject(t, org.ID)
		prof := createRandomProfile(t, proj.ID, []string{})

		sel, err := testQueries.CreateSelector(context.Background(), CreateSelectorParams{
			ProfileID: prof.ID,
			Entity:    NullEntities{},
			Selector:  "entity.name.startsWith(\"foo\") && !entity.name.startsWith(\"foobar\")",
			Comment:   "no foobar allowed, only foo",
		})
		require.NoError(t, err)
		require.NotEmpty(t, sel)

		updatedSel, err := testQueries.UpdateSelector(context.Background(), UpdateSelectorParams{
			ID: sel.ID,
			Entity: NullEntities{
				Entities: EntitiesRepository,
				Valid:    true,
			},
			Selector: "entity.name.startsWith(\"bar\") && !entity.name.startsWith(\"barfoo\")",
			Comment:  "no barfoo allowed, only bar",
		})
		require.NoError(t, err)
		require.NotEmpty(t, updatedSel)
		require.Equal(t, sel.ID, updatedSel.ID)
		require.Equal(t, sel.ProfileID, updatedSel.ProfileID)
		require.Equal(t, updatedSel.Entity, NullEntities{Entities: EntitiesRepository, Valid: true})
		require.Equal(t, updatedSel.Selector, "entity.name.startsWith(\"bar\") && !entity.name.startsWith(\"barfoo\")")
		require.Equal(t, updatedSel.Comment, "no barfoo allowed, only bar")
	})

	t.Run("Attempt to update non-existing", func(t *testing.T) {
		t.Parallel()

		randomID, err := uuid.NewRandom()
		require.NoError(t, err)

		updatedSel, err := testQueries.UpdateSelector(context.Background(), UpdateSelectorParams{
			ID: randomID,
			Entity: NullEntities{
				Entities: EntitiesRepository,
				Valid:    true,
			},
			Selector: "entity.name.startsWith(\"bar\") && !entity.name.startsWith(\"barfoo\")",
			Comment:  "no barfoo allowed, only bar",
		})
		require.Error(t, err)
		require.ErrorIs(t, err, sql.ErrNoRows)
		require.Empty(t, updatedSel)
	})
}

func TestDeleteSelector(t *testing.T) {
	t.Parallel()

	t.Run("Delete existing", func(t *testing.T) {
		t.Parallel()

		org := createRandomOrganization(t)
		proj := createRandomProject(t, org.ID)
		prof := createRandomProfile(t, proj.ID, []string{})

		sel, err := testQueries.CreateSelector(context.Background(), CreateSelectorParams{
			ProfileID: prof.ID,
			Entity:    NullEntities{},
			Selector:  "entity.name.startsWith(\"foo\") && !entity.name.startsWith(\"foobar\")",
			Comment:   "no foobar allowed, only foo",
		})
		require.NoError(t, err)
		require.NotEmpty(t, sel)

		err = testQueries.DeleteSelector(context.Background(), sel.ID)
		require.NoError(t, err)

		fetchedSel, err := testQueries.GetSelectorByID(context.Background(), sel.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, sql.ErrNoRows)
		require.Empty(t, fetchedSel)
	})

	t.Run("Attempt to delete non-existing", func(t *testing.T) {
		t.Parallel()

		randomID, err := uuid.NewRandom()
		require.NoError(t, err)

		err = testQueries.DeleteSelector(context.Background(), randomID)
		require.NoError(t, err)
	})
}

func TestGetSelectorByID(t *testing.T) {
	t.Parallel()

	t.Run("Get existing", func(t *testing.T) {
		t.Parallel()

		org := createRandomOrganization(t)
		proj := createRandomProject(t, org.ID)
		prof := createRandomProfile(t, proj.ID, []string{})

		sel, err := testQueries.CreateSelector(context.Background(), CreateSelectorParams{
			ProfileID: prof.ID,
			Entity:    NullEntities{},
			Selector:  "entity.name.startsWith(\"foo\") && !entity.name.startsWith(\"foobar\")",
			Comment:   "no foobar allowed, only foo",
		})
		require.NoError(t, err)
		require.NotEmpty(t, sel)

		fetchedSel, err := testQueries.GetSelectorByID(context.Background(), sel.ID)
		require.NoError(t, err)
		require.NotEmpty(t, fetchedSel)
		require.Equal(t, sel.ID, fetchedSel.ID)
		require.Equal(t, sel.ProfileID, fetchedSel.ProfileID)
		require.Equal(t, sel.Entity, fetchedSel.Entity)
		require.Equal(t, sel.Selector, fetchedSel.Selector)
		require.Equal(t, sel.Comment, fetchedSel.Comment)
	})

	t.Run("Get non-existing", func(t *testing.T) {
		t.Parallel()

		randomID, err := uuid.NewRandom()
		require.NoError(t, err)

		fetchedSel, err := testQueries.GetSelectorByID(context.Background(), randomID)
		require.Error(t, err)
		require.ErrorIs(t, err, sql.ErrNoRows)
		require.Empty(t, fetchedSel)
	})
}

func TestQueries_GetSelectorsByProfileID(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)

	t.Run("Get a list of existing", func(t *testing.T) {
		t.Parallel()

		prof := createRandomProfile(t, proj.ID, []string{})

		sel1, err := testQueries.CreateSelector(context.Background(), CreateSelectorParams{
			ProfileID: prof.ID,
			Entity:    NullEntities{},
			Selector:  "entity.name.startsWith(\"foo\") && !entity.name.startsWith(\"foobar\")",
			Comment:   "no foobar allowed, only foo",
		})
		require.NoError(t, err)
		require.NotEmpty(t, sel1)

		sel2, err := testQueries.CreateSelector(context.Background(), CreateSelectorParams{
			ProfileID: prof.ID,
			Entity: NullEntities{
				Entities: EntitiesRepository,
				Valid:    true,
			},
			Selector: "entity.name.startsWith(\"bar\") && !entity.name.startsWith(\"barfoo\")",
			Comment:  "no barfoo allowed, only bar",
		})
		require.NoError(t, err)
		require.NotEmpty(t, sel2)

		selectors, err := testQueries.GetSelectorsByProfileID(context.Background(), prof.ID)
		require.NoError(t, err)
		require.NotEmpty(t, selectors)
		require.Len(t, selectors, 2)
		require.Contains(t, selectors, sel1)
		require.Contains(t, selectors, sel2)
	})

	t.Run("Get an empty list", func(t *testing.T) {
		t.Parallel()

		prof := createRandomProfile(t, proj.ID, []string{})

		selectors, err := testQueries.GetSelectorsByProfileID(context.Background(), prof.ID)
		require.NoError(t, err)
		require.Empty(t, selectors)
	})

	t.Run("Non-existing profile", func(t *testing.T) {
		t.Parallel()

		randomID, err := uuid.NewRandom()
		require.NoError(t, err)

		selectors, err := testQueries.GetSelectorsByProfileID(context.Background(), randomID)
		require.NoError(t, err)
		require.Empty(t, selectors)
	})
}

func TestQueries_DeleteSelectorsByProfileID(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	proj := createRandomProject(t, org.ID)

	t.Run("Delete all profile selectors", func(t *testing.T) {
		t.Parallel()

		prof := createRandomProfile(t, proj.ID, []string{})

		sel1, err := testQueries.CreateSelector(context.Background(), CreateSelectorParams{
			ProfileID: prof.ID,
			Entity:    NullEntities{},
			Selector:  "entity.name.startsWith(\"foo\") && !entity.name.startsWith(\"foobar\")",
			Comment:   "no foobar allowed, only foo",
		})
		require.NoError(t, err)
		require.NotEmpty(t, sel1)

		sel2, err := testQueries.CreateSelector(context.Background(), CreateSelectorParams{
			ProfileID: prof.ID,
			Entity: NullEntities{
				Entities: EntitiesRepository,
				Valid:    true,
			},
			Selector: "entity.name.startsWith(\"bar\") && !entity.name.startsWith(\"barfoo\")",
			Comment:  "no barfoo allowed, only bar",
		})
		require.NoError(t, err)
		require.NotEmpty(t, sel2)

		selectors, err := testQueries.GetSelectorsByProfileID(context.Background(), prof.ID)
		require.NoError(t, err)
		require.NotEmpty(t, selectors)
		require.Len(t, selectors, 2)

		err = testQueries.DeleteSelectorsByProfileID(context.Background(), prof.ID)
		require.NoError(t, err)

		selectors, err = testQueries.GetSelectorsByProfileID(context.Background(), prof.ID)
		require.NoError(t, err)
		require.Empty(t, selectors)
	})

	t.Run("Profile with no selectors", func(t *testing.T) {
		t.Parallel()

		prof := createRandomProfile(t, proj.ID, []string{})

		selectors, err := testQueries.GetSelectorsByProfileID(context.Background(), prof.ID)
		require.NoError(t, err)
		require.Empty(t, selectors)
	})
}
