// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestListArtifacts_RepoFilter(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	providerID := uuid.New()
	providerName := "github"

	repoAID := uuid.New()
	repoBID := uuid.New()

	artifactFromRepoA := db.EntityInstance{
		ID:         uuid.New(),
		EntityType: db.EntitiesArtifact,
		Name:       "artifact-a",
		ProjectID:  projectID,
		ProviderID: providerID,
		CreatedAt:  time.Now(),
		OriginatedFrom: uuid.NullUUID{
			UUID:  repoAID,
			Valid: true,
		},
	}
	artifactFromRepoB := db.EntityInstance{
		ID:         uuid.New(),
		EntityType: db.EntitiesArtifact,
		Name:       "artifact-b",
		ProjectID:  projectID,
		ProviderID: providerID,
		CreatedAt:  time.Now(),
		OriginatedFrom: uuid.NullUUID{
			UUID:  repoBID,
			Valid: true,
		},
	}
	artifactNoRepo := db.EntityInstance{
		ID:             uuid.New(),
		EntityType:     db.EntitiesArtifact,
		Name:           "artifact-no-repo",
		ProjectID:      projectID,
		ProviderID:     providerID,
		CreatedAt:      time.Now(),
		OriginatedFrom: uuid.NullUUID{Valid: false},
	}

	allArtifacts := []db.EntityInstance{artifactFromRepoA, artifactFromRepoB, artifactNoRepo}

	tests := []struct {
		name          string
		from          string
		setupMocks    func(store *mockdb.MockStore)
		wantArtifacts []string
	}{
		{
			name: "no filter returns all artifacts",
			from: "",
			setupMocks: func(store *mockdb.MockStore) {
				store.EXPECT().GetProviderByName(gomock.Any(), db.GetProviderByNameParams{
					Name:     providerName,
					Projects: []uuid.UUID{projectID},
				}).Return(db.Provider{ID: providerID, Name: providerName}, nil)
				store.EXPECT().GetEntitiesByType(gomock.Any(), db.GetEntitiesByTypeParams{
					EntityType: db.EntitiesArtifact,
					ProviderID: providerID,
					Projects:   []uuid.UUID{projectID},
				}).Return(allArtifacts, nil)
			},
			wantArtifacts: []string{"artifact-a", "artifact-b", "artifact-no-repo"},
		},
		{
			name: "filter by repo-a returns only artifact-a",
			from: "repository=repo-a",
			setupMocks: func(store *mockdb.MockStore) {
				store.EXPECT().GetProviderByName(gomock.Any(), db.GetProviderByNameParams{
					Name:     providerName,
					Projects: []uuid.UUID{projectID},
				}).Return(db.Provider{ID: providerID, Name: providerName}, nil)
				store.EXPECT().GetEntitiesByType(gomock.Any(), db.GetEntitiesByTypeParams{
					EntityType: db.EntitiesArtifact,
					ProviderID: providerID,
					Projects:   []uuid.UUID{projectID},
				}).Return(allArtifacts, nil)
				store.EXPECT().GetEntityByName(gomock.Any(), db.GetEntityByNameParams{
					ProjectID:  projectID,
					EntityType: db.EntitiesRepository,
					Name:       "repo-a",
					ProviderID: providerID,
				}).Return(db.EntityInstance{ID: repoAID}, nil)
			},
			wantArtifacts: []string{"artifact-a"},
		},
		{
			name: "filter by unknown repo returns empty list",
			from: "repository=repo-unknown",
			setupMocks: func(store *mockdb.MockStore) {
				store.EXPECT().GetProviderByName(gomock.Any(), db.GetProviderByNameParams{
					Name:     providerName,
					Projects: []uuid.UUID{projectID},
				}).Return(db.Provider{ID: providerID, Name: providerName}, nil)
				store.EXPECT().GetEntitiesByType(gomock.Any(), db.GetEntitiesByTypeParams{
					EntityType: db.EntitiesArtifact,
					ProviderID: providerID,
					Projects:   []uuid.UUID{projectID},
				}).Return(allArtifacts, nil)
				store.EXPECT().GetEntityByName(gomock.Any(), db.GetEntityByNameParams{
					ProjectID:  projectID,
					EntityType: db.EntitiesRepository,
					Name:       "repo-unknown",
					ProviderID: providerID,
				}).Return(db.EntityInstance{}, sql.ErrNoRows)
			},
			wantArtifacts: []string{},
		},
		{
			name: "filter by repo-a,repo-b returns both artifacts",
			from: "repository=repo-a,repo-b",
			setupMocks: func(store *mockdb.MockStore) {
				store.EXPECT().GetProviderByName(gomock.Any(), db.GetProviderByNameParams{
					Name:     providerName,
					Projects: []uuid.UUID{projectID},
				}).Return(db.Provider{ID: providerID, Name: providerName}, nil)
				store.EXPECT().GetEntitiesByType(gomock.Any(), db.GetEntitiesByTypeParams{
					EntityType: db.EntitiesArtifact,
					ProviderID: providerID,
					Projects:   []uuid.UUID{projectID},
				}).Return(allArtifacts, nil)
				store.EXPECT().GetEntityByName(gomock.Any(), db.GetEntityByNameParams{
					ProjectID:  projectID,
					EntityType: db.EntitiesRepository,
					Name:       "repo-a",
					ProviderID: providerID,
				}).Return(db.EntityInstance{ID: repoAID}, nil)
				store.EXPECT().GetEntityByName(gomock.Any(), db.GetEntityByNameParams{
					ProjectID:  projectID,
					EntityType: db.EntitiesRepository,
					Name:       "repo-b",
					ProviderID: providerID,
				}).Return(db.EntityInstance{ID: repoBID}, nil)
			},
			wantArtifacts: []string{"artifact-a", "artifact-b"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.setupMocks(store)

			server := &Server{store: store}

			ctx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
				Project:  engcontext.Project{ID: projectID},
				Provider: engcontext.Provider{Name: providerName},
			})

			resp, err := server.ListArtifacts(ctx, &pb.ListArtifactsRequest{From: tc.from})
			require.NoError(t, err)

			var names []string
			for _, a := range resp.Results {
				names = append(names, a.Name)
			}
			if tc.wantArtifacts == nil {
				tc.wantArtifacts = []string{}
			}
			if len(tc.wantArtifacts) == 0 {
				assert.Empty(t, names)
			} else {
				assert.ElementsMatch(t, tc.wantArtifacts, names)
			}
		})
	}
}
