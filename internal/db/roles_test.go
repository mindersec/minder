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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/util/rand"
)

// A helper function to create a random role
func createRandomRole(t *testing.T, org uuid.UUID) Role {
	t.Helper()

	seed := time.Now().UnixNano()
	arg := CreateRoleParams{
		OrganizationID: org,
		Name:           rand.RandomName(seed),
	}

	role, err := testQueries.CreateRole(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, role)

	require.Equal(t, arg.Name, role.Name)

	require.NotZero(t, role.ID)
	require.NotZero(t, role.OrganizationID)
	require.NotZero(t, role.CreatedAt)
	require.NotZero(t, role.UpdatedAt)

	return role
}

func TestRole(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	createRandomRole(t, org.ID)
}

func TestGetRole(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	role1 := createRandomRole(t, org.ID)

	role2, err := testQueries.GetRoleByID(context.Background(), role1.ID)

	require.NoError(t, err)
	require.NotEmpty(t, role2)

	require.Equal(t, role1.Name, role2.Name)
	require.Equal(t, role1.ProjectID, role2.ProjectID)

	require.NotZero(t, role2.ID)
	require.NotZero(t, role2.CreatedAt)
	require.NotZero(t, role2.UpdatedAt)
	require.False(t, role2.IsAdmin)
}

func TestUpdateRole(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()
	org := createRandomOrganization(t)
	role1 := createRandomRole(t, org.ID)

	arg := UpdateRoleParams{
		ID:             role1.ID,
		OrganizationID: org.ID,
		Name:           rand.RandomName(seed),
		IsAdmin:        true,
	}

	role2, err := testQueries.UpdateRole(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, role2)

	require.Equal(t, role1.ID, role2.ID)
	require.Equal(t, role1.ProjectID, role2.ProjectID)
	require.Equal(t, arg.Name, role2.Name)

	require.NotZero(t, role2.CreatedAt)
	require.NotZero(t, role2.UpdatedAt)
	require.True(t, role2.IsAdmin)
}

func TestDeleteRole(t *testing.T) {
	t.Parallel()

	org := createRandomOrganization(t)
	role1 := createRandomRole(t, org.ID)

	err := testQueries.DeleteRole(context.Background(), role1.ID)

	require.NoError(t, err)

	role2, err := testQueries.GetRoleByID(context.Background(), role1.ID)

	require.Error(t, err)
	require.Empty(t, role2)
}
