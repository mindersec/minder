// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/pkg/util/rand"
)

func createRandomUser(t *testing.T) User {
	t.Helper()

	seed := time.Now().UnixNano()

	sub := rand.RandomString(10, seed)

	user, err := testQueries.CreateUser(context.Background(), sub)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, sub, user.IdentitySubject)

	require.NotZero(t, user.ID)
	require.NotZero(t, user.CreatedAt)
	require.NotZero(t, user.UpdatedAt)

	return user
}

func TestUser(t *testing.T) {
	t.Parallel()

	createRandomUser(t)
}

func TestGetUser(t *testing.T) {
	t.Parallel()

	user1 := createRandomUser(t)

	user2, err := testQueries.GetUserByID(context.Background(), user1.ID)

	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.ID, user2.ID)
	require.Equal(t, user1.IdentitySubject, user2.IdentitySubject)

	require.NotZero(t, user2.CreatedAt)
	require.NotZero(t, user2.UpdatedAt)

	require.WithinDuration(t, user1.CreatedAt, user2.CreatedAt, time.Second)
	require.WithinDuration(t, user1.UpdatedAt, user2.UpdatedAt, time.Second)
}
