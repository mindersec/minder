//
// Copyright 2023 Stacklok, Inc.
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
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/util/rand"
)

func createRandomEntity(t *testing.T, project uuid.UUID, provider uuid.UUID, entType Entities) {
	t.Helper()

	seed := time.Now().UnixNano()

	ent, err := testQueries.CreateEntity(context.Background(), CreateEntityParams{
		EntityType:     entType,
		Name:           rand.RandomName(seed),
		ProjectID:      project,
		ProviderID:     provider,
		OriginatedFrom: uuid.NullUUID{},
	})
	require.NoError(t, err)

	prop, err := testQueries.UpsertPropertyValueV1(context.Background(), UpsertPropertyValueV1Params{
		EntityID: ent.ID,
		Key:      "testkey1",
		Value:    rand.RandomName(seed),
	})
	require.NoError(t, err)
	require.NotEmpty(t, prop)

	prop, err = testQueries.UpsertPropertyValueV1(context.Background(), UpsertPropertyValueV1Params{
		EntityID: ent.ID,
		Key:      "testkey1",
		Value:    rand.RandomName(seed),
	})
	require.NoError(t, err)
	require.NotEmpty(t, prop)

	prop, err = testQueries.UpsertPropertyValueV1(context.Background(), UpsertPropertyValueV1Params{
		EntityID: ent.ID,
		Key:      "upstream_id",
		Value:    rand.RandomName(seed),
	})
	require.NoError(t, err)
	require.NotEmpty(t, prop)
}

func createRandomProject(t *testing.T, orgID uuid.UUID) Project {
	t.Helper()

	seed := time.Now().UnixNano()
	arg := CreateProjectParams{
		Name: rand.RandomName(seed),
		ParentID: uuid.NullUUID{
			UUID:  orgID,
			Valid: true,
		},
		Metadata: json.RawMessage(`{"company": "stacklok"}`),
	}

	proj, err := testQueries.CreateProject(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, proj)

	require.Equal(t, arg.Name, proj.Name)
	require.Equal(t, arg.Metadata, proj.Metadata)

	require.NotZero(t, proj.ID)
	require.NotZero(t, proj.CreatedAt)
	require.NotZero(t, proj.UpdatedAt)

	return proj
}

// createRoot will create a root project and return it. It will also return a cleanup function to delete the root
func createRoot(t *testing.T) (Project, func()) {
	t.Helper()

	proj, err := testQueries.CreateProject(context.Background(), CreateProjectParams{
		Name:     "Root " + t.Name(),
		ParentID: uuid.NullUUID{},
		Metadata: json.RawMessage("{}"),
	})
	require.NoError(t, err, "error creating root project")

	// return first root project
	return proj, func() {
		_, err := testQueries.DeleteProject(context.Background(), proj.ID)
		require.NoError(t, err, "error deleting root project")
	}
}

// createProjectHierarchy will create the specified depth of projects starting from a new root directory.
func createProjectHierarchy(t *testing.T, root *Project, depth int) (first, last *Project) {
	t.Helper()

	last = root

	for i := 0; i < depth; i++ {
		n := t.Name() + strconv.Itoa(i)
		p := CreateProjectParams{
			Name: n,
			ParentID: uuid.NullUUID{
				UUID:  last.ID,
				Valid: true,
			},
			Metadata: json.RawMessage("{}"),
		}

		res, err := testQueries.CreateProject(context.Background(), p)
		require.NoError(t, err, "error creating project")

		if first == nil {
			first = &res
		}

		last = &res
	}

	return first, last
}

func TestGetRoot(t *testing.T) {
	t.Parallel()

	rp, cleanup := createRoot(t)
	t.Cleanup(cleanup)
	assert.False(t, rp.ParentID.Valid, "root project should not have parent")
}

func TestCreateandGetProject(t *testing.T) {
	t.Parallel()

	rp, cleanup := createRoot(t)
	t.Cleanup(cleanup)
	puuid := uuid.NullUUID{
		UUID:  rp.ID,
		Valid: true,
	}

	createdP, err := testQueries.CreateProject(context.Background(), CreateProjectParams{
		Name:     t.Name(),
		ParentID: puuid,
		Metadata: json.RawMessage("{}"),
	})
	require.NoError(t, err, "error creating project")
	t.Cleanup(func() {
		_, err := testQueries.DeleteProject(context.Background(), createdP.ID)
		require.NoError(t, err, "error deleting project")
	})

	gotP, err := testQueries.GetProjectByID(context.Background(), createdP.ID)
	require.NoError(t, err, "error getting project")

	assert.Equal(t, createdP.ID, gotP.ID, "project id should match")
	assert.Equal(t, createdP.Name, gotP.Name, "project name should match")
	assert.Equal(t, createdP.ParentID, gotP.ParentID, "project parent id should match")
	assert.Equal(t, createdP.Metadata, gotP.Metadata, "project metadata should match")
}

func TestCreateDirectoryWithParentThatDoesntExist(t *testing.T) {
	t.Parallel()

	_, err := testQueries.CreateProject(context.Background(), CreateProjectParams{
		Name:     t.Name(),
		ParentID: uuid.NullUUID{UUID: uuid.New(), Valid: true},
	})

	assert.Error(t, err, "should have errored")
}

func TestDeleteDirectoryWithoutChildren(t *testing.T) {
	t.Parallel()

	rp, cleanup := createRoot(t)
	t.Cleanup(cleanup)

	p, err := testQueries.CreateProject(context.Background(), CreateProjectParams{
		Name:     t.Name(),
		ParentID: uuid.NullUUID{UUID: rp.ID, Valid: true},
		Metadata: json.RawMessage("{}"),
	})
	require.NoError(t, err, "error creating project")

	// list projects to ensure it was created
	projects, err := testQueries.GetChildrenProjects(context.Background(), rp.ID)
	require.NoError(t, err, "error getting children")
	l := CalculateProjectHierarchyOffset(1)
	assert.Len(t, projects, l)
	assert.Equal(t, p.ID, projects[CalculateProjectHierarchyOffset(0)].ID, "project id should match")

	// delete project
	_, err = testQueries.DeleteProject(context.Background(), p.ID)
	require.NoError(t, err, "error deleting project")

	// list projects to ensure it was deleted
	projects, err = testQueries.GetChildrenProjects(context.Background(), rp.ID)
	require.NoError(t, err, "error getting children")
	l = CalculateProjectHierarchyOffset(0)
	assert.Len(t, projects, l)
}

func TestDeleteDirectoryWithChildren(t *testing.T) {
	t.Parallel()

	rp, cleanup := createRoot(t)
	t.Cleanup(cleanup)

	const depth = 5
	first, _ := createProjectHierarchy(t, &rp, depth)

	// list projects to ensure they were created
	projects, err := testQueries.GetChildrenProjects(context.Background(), rp.ID)
	require.NoError(t, err, "error getting children")

	l := CalculateProjectHierarchyOffset(depth)
	assert.Len(t, projects, l)

	// delete project
	_, err = testQueries.DeleteProject(context.Background(), first.ID)
	require.NoError(t, err, "error deleting project")

	// list projects to ensure they were all deleted
	projects, err = testQueries.GetChildrenProjects(context.Background(), rp.ID)
	require.NoError(t, err, "error getting children")

	l = CalculateProjectHierarchyOffset(0)
	assert.Len(t, projects, l)
}

func TestQueryUnknownDirectory(t *testing.T) {
	t.Parallel()

	_, err := testQueries.GetProjectByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, sql.ErrNoRows, "should have errored")
}

func TestGetSingleParent(t *testing.T) {
	t.Parallel()

	rp, cleanup := createRoot(t)
	t.Cleanup(cleanup)
	puuid := uuid.NullUUID{
		UUID:  rp.ID,
		Valid: true,
	}

	createdP, err := testQueries.CreateProject(context.Background(), CreateProjectParams{
		Name:     t.Name(),
		ParentID: puuid,
		Metadata: json.RawMessage("{}"),
	})
	require.NoError(t, err, "error creating project")
	t.Cleanup(func() {
		_, err := testQueries.DeleteProject(context.Background(), createdP.ID)
		require.NoError(t, err, "error deleting project")
	})

	// Get parent should be root
	parents, err := testQueries.GetParentProjects(context.Background(), createdP.ID)
	require.NoError(t, err, "error getting parents")

	// Should only have one parent
	l := CalculateProjectHierarchyOffset(1)
	assert.Len(t, parents, l)

	// Parent should be root
	assert.Equal(t, rp.ID, parents[CalculateProjectHierarchyOffset(0)], "parent should be root")
}

func TestGetParentFromRootDirShouldReturnEmpty(t *testing.T) {
	t.Parallel()

	rp, cleanup := createRoot(t)
	t.Cleanup(cleanup)

	parents, err := testQueries.GetParentProjects(context.Background(), rp.ID)
	require.NoError(t, err, "error getting parents")

	l := CalculateProjectHierarchyOffset(0)
	assert.Len(t, parents, l)
}

func TestGetParentsFromUnknownShouldReturnEmpty(t *testing.T) {
	t.Parallel()

	ps, err := testQueries.GetParentProjects(context.Background(), uuid.New())
	require.NoError(t, err, "error getting parents")

	require.Len(t, ps, 0)
}

func TestGetChildrenMayReturnEmptyAppropriately(t *testing.T) {
	t.Parallel()

	rp, cleanup := createRoot(t)
	t.Cleanup(cleanup)

	parents, err := testQueries.GetChildrenProjects(context.Background(), rp.ID)
	require.NoError(t, err, "error getting children")

	l := CalculateProjectHierarchyOffset(0)
	assert.Len(t, parents, l)
}
