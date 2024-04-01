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

	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/util/rand"
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
