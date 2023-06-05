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

	"github.com/stacklok/mediator/pkg/util"
	"github.com/stretchr/testify/require"
)

// A helper function to create a random role
func createRandomRole(t *testing.T, group int32) Role {
	seed := time.Now().UnixNano()
	arg := CreateRoleParams{
		GroupID: group,
		Name:    util.RandomName(seed),
	}

	role, err := testQueries.CreateRole(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, role)

	require.Equal(t, arg.Name, role.Name)

	require.NotZero(t, role.ID)
	require.NotZero(t, role.GroupID)
	require.NotZero(t, role.CreatedAt)
	require.NotZero(t, role.UpdatedAt)

	return role
}

func TestRole(t *testing.T) {
	org := createRandomOrganization(t)
	group := createRandomGroup(t, org.ID)
	createRandomRole(t, group.ID)
}

func TestGetRole(t *testing.T) {
	org := createRandomOrganization(t)
	group := createRandomGroup(t, org.ID)
	role1 := createRandomRole(t, group.ID)

	role2, err := testQueries.GetRoleByID(context.Background(), role1.ID)

	require.NoError(t, err)
	require.NotEmpty(t, role2)

	require.Equal(t, role1.Name, role2.Name)
	require.Equal(t, role1.GroupID, role2.GroupID)

	require.NotZero(t, role2.ID)
	require.NotZero(t, role2.CreatedAt)
	require.NotZero(t, role2.UpdatedAt)
	require.False(t, role2.IsAdmin)
	require.False(t, role2.IsProtected)
}

func TestUpdateRole(t *testing.T) {
	seed := time.Now().UnixNano()
	org := createRandomOrganization(t)
	group := createRandomGroup(t, org.ID)
	role1 := createRandomRole(t, group.ID)

	arg := UpdateRoleParams{
		ID:      role1.ID,
		GroupID: group.ID,
		Name:    util.RandomName(seed),
		IsAdmin: true,
	}

	role2, err := testQueries.UpdateRole(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, role2)

	require.Equal(t, role1.ID, role2.ID)
	require.Equal(t, role1.GroupID, role2.GroupID)
	require.Equal(t, arg.Name, role2.Name)

	require.NotZero(t, role2.CreatedAt)
	require.NotZero(t, role2.UpdatedAt)
	require.True(t, role2.IsAdmin)
}

func TestDeleteRole(t *testing.T) {
	org := createRandomOrganization(t)
	role1 := createRandomRole(t, org.ID)

	err := testQueries.DeleteRole(context.Background(), role1.ID)

	require.NoError(t, err)

	role2, err := testQueries.GetRoleByID(context.Background(), role1.ID)

	require.Error(t, err)
	require.Empty(t, role2)
}
