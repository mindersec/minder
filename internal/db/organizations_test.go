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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/util/rand"
)

// A helper function to create a random organization
func createRandomOrganization(t *testing.T) Project {
	t.Helper()

	seed := time.Now().UnixNano()
	arg := CreateOrganizationParams{
		Name:     rand.RandomName(seed),
		Metadata: json.RawMessage(`{"company": "stacklok"}`),
	}

	organization, err := testQueries.CreateOrganization(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, organization)

	require.Equal(t, arg.Name, organization.Name)
	require.Equal(t, arg.Metadata, organization.Metadata)

	require.NotZero(t, organization.ID)
	require.NotZero(t, organization.CreatedAt)
	require.NotZero(t, organization.UpdatedAt)

	return organization
}

// Create a random organization
func TestOrganization(t *testing.T) {
	t.Parallel()

	createRandomOrganization(t)
}

func TestGetOrganization(t *testing.T) {
	t.Parallel()

	organization1 := createRandomOrganization(t)

	organization2, err := testQueries.GetOrganization(context.Background(), organization1.ID)

	require.NoError(t, err)
	require.NotEmpty(t, organization2)

	require.Equal(t, organization1.ID, organization2.ID)
	require.Equal(t, organization1.Name, organization2.Name)
	require.Equal(t, organization1.Metadata, organization2.Metadata)

	require.NotZero(t, organization2.CreatedAt)
	require.NotZero(t, organization2.UpdatedAt)

	require.WithinDuration(t, organization1.CreatedAt, organization2.CreatedAt, time.Second)
	require.WithinDuration(t, organization1.UpdatedAt, organization2.UpdatedAt, time.Second)

}

func TestUpdateOrganization(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()
	organization1 := createRandomOrganization(t)

	cmpName := rand.RandomName(seed)
	arg := UpdateOrganizationParams{
		ID:       organization1.ID,
		Name:     rand.RandomName(seed),
		Metadata: json.RawMessage(fmt.Sprintf(`{"company": "%s"}`, cmpName)),
	}

	organization2, err := testQueries.UpdateOrganization(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, organization2)

	require.Equal(t, organization1.ID, organization2.ID)
	require.Equal(t, arg.Name, organization2.Name)
	require.Equal(t, arg.Metadata, organization2.Metadata)

	require.NotZero(t, organization2.CreatedAt)
	require.NotZero(t, organization2.UpdatedAt)

	require.WithinDuration(t, organization1.CreatedAt, organization2.CreatedAt, time.Second)
	require.WithinDuration(t, organization1.UpdatedAt, organization2.UpdatedAt, time.Second)
}

func TestDeleteOrganization(t *testing.T) {
	t.Parallel()

	organization1 := createRandomOrganization(t)

	err := testQueries.DeleteOrganization(context.Background(), organization1.ID)
	require.NoError(t, err)

	organization2, err := testQueries.GetOrganization(context.Background(), organization1.ID)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, organization2)
}

func TestListOrganizations(t *testing.T) {
	t.Parallel()

	for i := 0; i < 10; i++ {
		createRandomOrganization(t)
	}

	arg := ListOrganizationsParams{
		Limit:  5,
		Offset: 5,
	}

	organizations, err := testQueries.ListOrganizations(context.Background(), arg)

	require.NoError(t, err)
	require.Len(t, organizations, 5)

	for _, organization := range organizations {
		require.NotEmpty(t, organization)
	}
}
