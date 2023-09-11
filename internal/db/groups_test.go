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
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stacklok/mediator/internal/util"
)

func createRandomGroup(t *testing.T, org int32) Group {
	t.Helper()

	seed := time.Now().UnixNano()
	arg := CreateGroupParams{
		OrganizationID: org,
		Name:           util.RandomName(seed),
	}

	group, err := testQueries.CreateGroup(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, group)

	require.Equal(t, arg.OrganizationID, group.OrganizationID)
	require.Equal(t, arg.Name, group.Name)

	require.NotZero(t, group.ID)
	require.NotZero(t, group.CreatedAt)
	require.NotZero(t, group.UpdatedAt)

	return group
}

func TestGroup(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	createRandomGroup(t, org.ID)
}

func TestGetGroup(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	group1 := createRandomGroup(t, org.ID)

	group2, err := testQueries.GetGroupByID(context.Background(), group1.ID)

	require.NoError(t, err)
	require.NotEmpty(t, group2)

	require.Equal(t, group1.OrganizationID, group2.OrganizationID)
	require.Equal(t, group1.Name, group2.Name)

	require.NotZero(t, group2.ID)
	require.NotZero(t, group2.CreatedAt)
	require.NotZero(t, group2.UpdatedAt)
}

func TestListGroups(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)

	for i := 0; i < 10; i++ {
		createRandomGroup(t, org.ID)
	}

	arg := ListGroupsParams{
		OrganizationID: org.ID,
		Limit:          5,
		Offset:         5,
	}

	groups, err := testQueries.ListGroups(context.Background(), arg)

	require.NoError(t, err)
	require.Len(t, groups, 5)

	for _, group := range groups {
		require.NotEmpty(t, group)
	}
}

func TestUpdateGroup(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()
	org := createRandomOrganization(t)
	group1 := createRandomGroup(t, org.ID)

	arg := UpdateGroupParams{
		ID:             group1.ID,
		OrganizationID: org.ID,
		Name:           util.RandomName(seed),
	}

	group2, err := testQueries.UpdateGroup(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, group2)

	require.Equal(t, arg.OrganizationID, group2.OrganizationID)
	require.Equal(t, arg.Name, group2.Name)

	require.NotZero(t, group2.ID)
	require.NotZero(t, group2.CreatedAt)
	require.NotZero(t, group2.UpdatedAt)
}

func TestDeleteGroup(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	group1 := createRandomGroup(t, org.ID)

	err := testQueries.DeleteGroup(context.Background(), group1.ID)

	require.NoError(t, err)

	group2, err := testQueries.GetGroupByID(context.Background(), group1.ID)

	require.Error(t, err)
	require.Empty(t, group2)
}

func TestListGroupsByOrganization(t *testing.T) {
	t.Parallel()

	org1 := createRandomOrganization(t)
	org2 := createRandomOrganization(t)

	for i := 0; i < 10; i++ {
		createRandomGroup(t, org1.ID)
		createRandomGroup(t, org2.ID)
	}

	arg := ListGroupsParams{
		OrganizationID: org1.ID,
		Limit:          5,
		Offset:         5,
	}

	groups, err := testQueries.ListGroups(context.Background(), arg)

	require.NoError(t, err)
	require.Len(t, groups, 5)

	for _, group := range groups {
		require.NotEmpty(t, group)
		require.Equal(t, org1.ID, group.OrganizationID)
	}
}
