package db

// import (
// 	"context"
// 	"testing"

// 	"github.com/stretchr/testify/require"
// )

// func TestStoreTX(t *testing.T) {
// 	ctx := context.Background()
// 	store := NewStore(testDB)
// 	org1 := createRandomOrganisation(t)
// 	org2, err := store.GetOrganisation(ctx, org1.ID)
// 	require.NoError(t, err)

// 	n := 5
// 	results := make(chan Organisation)

// 	for i := 0; i < n; i++ {
// 		go func() {
// 			updatedOrg, err := store.GetOrganisation(ctx, org1.ID)
// 			require.NoError(t, err)
// 			results <- *updatedOrg
// 		}()
// 	}

// 	for i := 0; i < n; i++ {
// 		updatedOrg := <-results
// 		require.Equal(t, org2, &updatedOrg)
// 	}

// 	// require.Equal(t, org2, org1)

// }
