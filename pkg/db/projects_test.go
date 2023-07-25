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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(jaosorior): Currently we have the caveat that GetChildrenProjects and GetParentProjects
// will also return the calling project. I didn't quite figure out how to
// filter this out with the CTE query. I think it's possible, but I'm not
// sure how to do it. For now, we'll just filter it out in the code.
// Once we figure out how to do it in the query, we can remove the filtering
// in the code and remove the +1 in the hierarchy offset and set it to 0.
const hierarchyOffset = 1

func calculateHierarchyOffset(hierarchy int) int {
	return hierarchy + hierarchyOffset
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
	l := calculateHierarchyOffset(1)
	assert.Len(t, projects, l)
	assert.Equal(t, p.ID, projects[calculateHierarchyOffset(0)], "project id should match")

	// delete project
	_, err = testQueries.DeleteProject(context.Background(), p.ID)
	require.NoError(t, err, "error deleting project")

	// list projects to ensure it was deleted
	projects, err = testQueries.GetChildrenProjects(context.Background(), rp.ID)
	require.NoError(t, err, "error getting children")
	l = calculateHierarchyOffset(0)
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

	l := calculateHierarchyOffset(depth)
	assert.Len(t, projects, l)

	// delete project
	_, err = testQueries.DeleteProject(context.Background(), first.ID)
	require.NoError(t, err, "error deleting project")

	// list projects to ensure they were all deleted
	projects, err = testQueries.GetChildrenProjects(context.Background(), rp.ID)
	require.NoError(t, err, "error getting children")

	l = calculateHierarchyOffset(0)
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
	l := calculateHierarchyOffset(1)
	assert.Len(t, parents, l)

	// Parent should be root
	assert.Equal(t, rp.ID, parents[calculateHierarchyOffset(0)], "parent should be root")
}

func TestGetParentFromRootDirShouldReturnEmpty(t *testing.T) {
	t.Parallel()

	rp, cleanup := createRoot(t)
	t.Cleanup(cleanup)

	parents, err := testQueries.GetParentProjects(context.Background(), rp.ID)
	require.NoError(t, err, "error getting parents")

	l := calculateHierarchyOffset(0)
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

	l := calculateHierarchyOffset(0)
	assert.Len(t, parents, l)
}
