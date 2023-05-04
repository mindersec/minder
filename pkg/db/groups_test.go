package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stacklok/mediator/pkg/util"

	"github.com/stretchr/testify/require"
)

func createRandomGroup(t *testing.T, org int32) Group {
	arg := CreateGroupParams{
		OrganisationID: sql.NullInt32{Int32: org, Valid: true},
		Name:           util.RandomName(),
	}

	group, err := testQueries.CreateGroup(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, group)

	require.Equal(t, arg.OrganisationID, group.OrganisationID)
	require.Equal(t, arg.Name, group.Name)

	require.NotZero(t, group.ID)
	require.NotZero(t, group.CreatedAt)
	require.NotZero(t, group.UpdatedAt)

	return group
}

func TestGroup(t *testing.T) {
	org := createRandomOrganisation(t)
	createRandomGroup(t, org.ID)
}

func TestGetGroup(t *testing.T) {
	org := createRandomOrganisation(t)
	group1 := createRandomGroup(t, org.ID)

	group2, err := testQueries.GetGroupByID(context.Background(), group1.ID)

	require.NoError(t, err)
	require.NotEmpty(t, group2)

	require.Equal(t, group1.OrganisationID, group2.OrganisationID)
	require.Equal(t, group1.Name, group2.Name)

	require.NotZero(t, group2.ID)
	require.NotZero(t, group2.CreatedAt)
	require.NotZero(t, group2.UpdatedAt)
}

func TestListGroups(t *testing.T) {
	org := createRandomOrganisation(t)

	for i := 0; i < 10; i++ {
		createRandomGroup(t, org.ID)
	}

	arg := ListGroupsParams{
		OrganisationID: sql.NullInt32{Int32: org.ID, Valid: true},
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
	org := createRandomOrganisation(t)
	group1 := createRandomGroup(t, org.ID)

	arg := UpdateGroupParams{
		ID:             group1.ID,
		OrganisationID: sql.NullInt32{Int32: org.ID, Valid: true},
		Name:           util.RandomName(),
	}

	group2, err := testQueries.UpdateGroup(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, group2)

	require.Equal(t, arg.OrganisationID, group2.OrganisationID)
	require.Equal(t, arg.Name, group2.Name)

	require.NotZero(t, group2.ID)
	require.NotZero(t, group2.CreatedAt)
	require.NotZero(t, group2.UpdatedAt)
}

func TestDeleteGroup(t *testing.T) {
	org := createRandomOrganisation(t)
	group1 := createRandomGroup(t, org.ID)

	err := testQueries.DeleteGroup(context.Background(), group1.ID)

	require.NoError(t, err)

	group2, err := testQueries.GetGroupByID(context.Background(), group1.ID)

	require.Error(t, err)
	require.Empty(t, group2)
}

func TestListGroupsByOrganisation(t *testing.T) {
	org1 := createRandomOrganisation(t)
	org2 := createRandomOrganisation(t)

	for i := 0; i < 10; i++ {
		createRandomGroup(t, org1.ID)
		createRandomGroup(t, org2.ID)
	}

	arg := ListGroupsParams{
		OrganisationID: sql.NullInt32{Int32: org1.ID, Valid: true},
		Limit:          5,
		Offset:         5,
	}

	groups, err := testQueries.ListGroups(context.Background(), arg)

	require.NoError(t, err)
	require.Len(t, groups, 5)

	for _, group := range groups {
		require.NotEmpty(t, group)
		require.Equal(t, org1.ID, group.OrganisationID.Int32)
	}
}
