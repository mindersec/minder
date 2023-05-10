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
	"fmt"
	"testing"
	"time"

	"github.com/stacklok/mediator/pkg/util"

	"github.com/stretchr/testify/require"
)

func createRandomUser(t *testing.T, org Organisation) User {
	seed := time.Now().UnixNano()
	group := createRandomGroup(t, org.ID)

	arg := CreateUserParams{
		OrganisationID: sql.NullInt32{Int32: org.ID, Valid: true},
		GroupID:        sql.NullInt32{Int32: group.ID, Valid: true},
		Email:          util.RandomEmail(seed),
		Password:       util.RandomString(10, seed),
		Name:           util.RandomString(10, seed),
		AvatarUrl:      util.RandomURL(seed),
		ProviderID:     fmt.Sprintf("%d", util.RandomInt(1, 1000, seed)),
		IsAdmin:        true,
		IsSuperAdmin:   true,
	}

	user, err := testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.OrganisationID, user.OrganisationID)
	require.Equal(t, arg.GroupID, user.GroupID)
	require.Equal(t, arg.Email, user.Email)
	require.Equal(t, arg.Password, user.Password)
	require.Equal(t, arg.Name, user.Name)
	require.Equal(t, arg.AvatarUrl, user.AvatarUrl)
	require.Equal(t, arg.ProviderID, user.ProviderID)
	require.Equal(t, arg.IsAdmin, user.IsAdmin)
	require.Equal(t, arg.IsSuperAdmin, user.IsSuperAdmin)

	require.NotZero(t, user.ID)
	require.NotZero(t, user.CreatedAt)
	require.NotZero(t, user.UpdatedAt)

	return user
}

func TestUser(t *testing.T) {
	org := createRandomOrganisation(t)
	createRandomUser(t, org)
}

func TestGetUser(t *testing.T) {
	org := createRandomOrganisation(t)
	user1 := createRandomUser(t, org)

	user2, err := testQueries.GetUserByID(context.Background(), user1.ID)

	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.OrganisationID, user2.OrganisationID)
	require.Equal(t, user1.GroupID, user2.GroupID)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.Password, user2.Password)
	require.Equal(t, user1.Name, user2.Name)
	require.Equal(t, user1.AvatarUrl, user2.AvatarUrl)
	require.Equal(t, user1.ProviderID, user2.ProviderID)
	require.Equal(t, user1.IsAdmin, user2.IsAdmin)
	require.Equal(t, user1.IsSuperAdmin, user2.IsSuperAdmin)

	require.NotZero(t, user2.CreatedAt)
	require.NotZero(t, user2.UpdatedAt)

	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
	require.WithinDuration(t, user1.UpdatedAt, user2.UpdatedAt, time.Second)
}

func TestUpdateUser(t *testing.T) {
	seed := time.Now().UnixNano()
	org := createRandomOrganisation(t)
	user1 := createRandomUser(t, org)

	arg := UpdateUserParams{
		ID:             user1.ID,
		OrganisationID: sql.NullInt32{Int32: user1.OrganisationID.Int32, Valid: true},
		GroupID:        sql.NullInt32{Int32: user1.GroupID.Int32, Valid: true},
		Email:          util.RandomEmail(seed),
		Password:       util.RandomString(10, seed),
		Name:           util.RandomName(seed),
		AvatarUrl:      util.RandomURL(seed),
		ProviderID:     fmt.Sprintf("%d", util.RandomInt(1, 1000, seed)),
		IsAdmin:        true,
		IsSuperAdmin:   true,
	}

	user2, err := testQueries.UpdateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, arg.ID, user2.ID)
	require.Equal(t, arg.OrganisationID, user2.OrganisationID)
	require.Equal(t, arg.GroupID, user2.GroupID)
	require.Equal(t, arg.Email, user2.Email)
	require.Equal(t, arg.Password, user2.Password)
	require.Equal(t, arg.Name, user2.Name)
	require.Equal(t, arg.AvatarUrl, user2.AvatarUrl)
	require.Equal(t, arg.ProviderID, user2.ProviderID)
	require.Equal(t, arg.IsAdmin, user2.IsAdmin)
	require.Equal(t, arg.IsSuperAdmin, user2.IsSuperAdmin)

	require.NotZero(t, user2.CreatedAt)
	require.NotZero(t, user2.UpdatedAt)

	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
	require.WithinDuration(t, user1.UpdatedAt, user2.UpdatedAt, time.Second)
}
