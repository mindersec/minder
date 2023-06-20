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
	"testing"
	"time"

	"github.com/stacklok/mediator/pkg/util"

	"github.com/stretchr/testify/require"
)

func stringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func createRandomUser(t *testing.T, org Organization) User {
	seed := time.Now().UnixNano()

	arg := CreateUserParams{
		OrganizationID: org.ID,
		Email:          stringToNullString(util.RandomEmail(seed)),
		Username:       util.RandomString(10, seed),
		Password:       util.RandomPassword(10, seed),
		FirstName:      stringToNullString(util.RandomName(seed)),
		LastName:       stringToNullString(util.RandomName(seed)),
	}

	user, err := testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Email, user.Email)
	require.Equal(t, arg.OrganizationID, user.OrganizationID)
	require.Equal(t, arg.Username, user.Username)
	require.Equal(t, arg.Password, user.Password)
	require.Equal(t, arg.FirstName, user.FirstName)
	require.Equal(t, arg.LastName, user.LastName)
	require.Equal(t, false, user.IsProtected)

	require.NotZero(t, user.ID)
	require.NotZero(t, user.CreatedAt)
	require.NotZero(t, user.UpdatedAt)

	return user
}

func TestUser(t *testing.T) {
	org := createRandomOrganization(t)
	createRandomUser(t, org)
}

func TestGetUser(t *testing.T) {
	org := createRandomOrganization(t)
	user1 := createRandomUser(t, org)

	user2, err := testQueries.GetUserByID(context.Background(), user1.ID)

	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.OrganizationID, user2.OrganizationID)
	require.Equal(t, user1.Email, user2.Email)
	require.Equal(t, user1.Username, user2.Username)
	require.Equal(t, user1.Password, user2.Password)
	require.Equal(t, user1.FirstName, user2.FirstName)
	require.Equal(t, user1.LastName, user2.LastName)
	require.Equal(t, user1.IsProtected, user2.IsProtected)

	require.NotZero(t, user2.CreatedAt)
	require.NotZero(t, user2.UpdatedAt)

	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
	require.WithinDuration(t, user1.UpdatedAt, user2.UpdatedAt, time.Second)
}

func TestUpdateUser(t *testing.T) {
	seed := time.Now().UnixNano()
	org := createRandomOrganization(t)
	user1 := createRandomUser(t, org)

	arg := UpdateUserParams{
		ID:        user1.ID,
		Email:     stringToNullString(util.RandomEmail(seed)),
		Username:  util.RandomString(10, seed),
		Password:  util.RandomString(10, seed),
		FirstName: stringToNullString(util.RandomName(seed)),
		LastName:  stringToNullString(util.RandomName(seed)),
	}

	user2, err := testQueries.UpdateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, arg.ID, user2.ID)
	require.Equal(t, arg.Email, user2.Email)
	require.Equal(t, arg.Username, user2.Username)
	require.Equal(t, arg.Password, user2.Password)
	require.Equal(t, arg.FirstName, user2.FirstName)
	require.Equal(t, arg.LastName, user2.LastName)
	require.Equal(t, arg.IsProtected, user2.IsProtected)

	require.NotZero(t, user2.CreatedAt)
	require.NotZero(t, user2.UpdatedAt)

	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
	require.WithinDuration(t, user1.UpdatedAt, user2.UpdatedAt, time.Second)
}
