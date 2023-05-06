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

// A helper function to create a random organisation
func createRandomOrganisation(t *testing.T) Organisation {
	seed := time.Now().UnixNano()
	arg := CreateOrganisationParams{
		Name:    util.RandomName(seed),
		Company: util.RandomName(seed),
	}

	organisation, err := testQueries.CreateOrganisation(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, organisation)

	require.Equal(t, arg.Name, organisation.Name)
	require.Equal(t, arg.Company, organisation.Company)

	require.NotZero(t, organisation.ID)
	require.NotZero(t, organisation.CreatedAt)
	require.NotZero(t, organisation.UpdatedAt)

	return organisation
}

// Create a random organisation
func TestOrganisation(t *testing.T) {
	createRandomOrganisation(t)
}

func TestGetOrganisation(t *testing.T) {
	organisation1 := createRandomOrganisation(t)

	organisation2, err := testQueries.GetOrganisation(context.Background(), organisation1.ID)

	require.NoError(t, err)
	require.NotEmpty(t, organisation2)

	require.Equal(t, organisation1.ID, organisation2.ID)
	require.Equal(t, organisation1.Name, organisation2.Name)
	require.Equal(t, organisation1.Company, organisation2.Company)

	require.NotZero(t, organisation2.CreatedAt)
	require.NotZero(t, organisation2.UpdatedAt)

	require.WithinDuration(t, organisation1.CreatedAt, organisation2.CreatedAt, time.Second)
	require.WithinDuration(t, organisation1.UpdatedAt, organisation2.UpdatedAt, time.Second)

}

func TestUpdateOrganisation(t *testing.T) {
	seed := time.Now().UnixNano()
	organisation1 := createRandomOrganisation(t)

	arg := UpdateOrganisationParams{
		ID:      organisation1.ID,
		Name:    util.RandomName(seed),
		Company: util.RandomName(seed),
	}

	organisation2, err := testQueries.UpdateOrganisation(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, organisation2)

	require.Equal(t, organisation1.ID, organisation2.ID)
	require.Equal(t, arg.Name, organisation2.Name)
	require.Equal(t, arg.Company, organisation2.Company)

	require.NotZero(t, organisation2.CreatedAt)
	require.NotZero(t, organisation2.UpdatedAt)

	require.WithinDuration(t, organisation1.CreatedAt, organisation2.CreatedAt, time.Second)
	require.WithinDuration(t, organisation1.UpdatedAt, organisation2.UpdatedAt, time.Second)
}

func TestDeleteOrganisation(t *testing.T) {
	organisation1 := createRandomOrganisation(t)

	err := testQueries.DeleteOrganisation(context.Background(), organisation1.ID)
	require.NoError(t, err)

	organisation2, err := testQueries.GetOrganisation(context.Background(), organisation1.ID)
	require.Error(t, err)
	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, organisation2)
}

func TestListOrganisations(t *testing.T) {
	for i := 0; i < 10; i++ {
		createRandomOrganisation(t)
	}

	arg := ListOrganisationsParams{
		Limit:  5,
		Offset: 5,
	}

	organisations, err := testQueries.ListOrganisations(context.Background(), arg)

	require.NoError(t, err)
	require.Len(t, organisations, 5)

	for _, organisation := range organisations {
		require.NotEmpty(t, organisation)
	}
}
