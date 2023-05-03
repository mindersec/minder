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
