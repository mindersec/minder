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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	entmodels "github.com/mindersec/minder/internal/entities/models"
	mockpropssvc "github.com/mindersec/minder/internal/entities/properties/service/mock"
	ghprops "github.com/mindersec/minder/internal/providers/github/properties"
	mockproviders "github.com/mindersec/minder/internal/providers/mock"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
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

func TestGetArtifactByName(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	providerID := uuid.New()
	artifactID := uuid.New()
	providerName := "github"

	baseCtx := engcontext.WithEntityContext(context.Background(), &engcontext.EntityContext{
		Project:  engcontext.Project{ID: projectID},
		Provider: engcontext.Provider{Name: providerName},
	})

	validProvider := &db.Provider{ID: providerID, Name: providerName}

	makeEWP := func(repoFullName string) *entmodels.EntityWithProperties {
		propsObj := properties.NewProperties(map[string]any{
			ghprops.ArtifactPropertyRepo: repoFullName,
		})
		return entmodels.NewEntityWithPropertiesFromInstance(
			entmodels.EntityInstance{
				ID:        artifactID,
				Type:      pb.Entity_ENTITY_ARTIFACTS,
				ProjectID: projectID,
			}, propsObj)
	}

	tests := []struct {
		name        string
		artifactRef string
		setupMocks  func(*mockdb.MockStore, *mockproviders.MockProviderStore, *mockpropssvc.MockPropertiesService)
		wantCode    codes.Code
	}{
		{
			name:        "invalid name — too few parts",
			artifactRef: "myorg/myartifact",
			setupMocks:  func(_ *mockdb.MockStore, _ *mockproviders.MockProviderStore, _ *mockpropssvc.MockPropertiesService) {},
			wantCode:    codes.InvalidArgument,
		},
		{
			name:        "provider not found",
			artifactRef: "myorg/myrepo/myartifact",
			setupMocks: func(_ *mockdb.MockStore, ps *mockproviders.MockProviderStore, _ *mockpropssvc.MockPropertiesService) {
				ps.EXPECT().GetByName(gomock.Any(), projectID, providerName).Return(nil, sql.ErrNoRows)
			},
			wantCode: codes.NotFound,
		},
		{
			name:        "artifact not found in DB",
			artifactRef: "myorg/myrepo/myartifact",
			setupMocks: func(store *mockdb.MockStore, ps *mockproviders.MockProviderStore, _ *mockpropssvc.MockPropertiesService) {
				ps.EXPECT().GetByName(gomock.Any(), projectID, providerName).Return(validProvider, nil)
				store.EXPECT().GetTypedEntitiesByPropertyV1(
					gomock.Any(), db.EntitiesArtifact, properties.PropertyName, "myorg/myartifact",
					db.GetTypedEntitiesOptions{ProjectID: projectID, ProviderID: providerID},
				).Return(nil, nil)
			},
			wantCode: codes.NotFound,
		},
		{
			name:        "repo mismatch — wrong repo in name",
			artifactRef: "myorg/wrong-repo/myartifact",
			setupMocks: func(store *mockdb.MockStore, ps *mockproviders.MockProviderStore, props *mockpropssvc.MockPropertiesService) {
				ps.EXPECT().GetByName(gomock.Any(), projectID, providerName).Return(validProvider, nil)
				store.EXPECT().GetTypedEntitiesByPropertyV1(
					gomock.Any(), db.EntitiesArtifact, properties.PropertyName, "myorg/myartifact",
					db.GetTypedEntitiesOptions{ProjectID: projectID, ProviderID: providerID},
				).Return([]db.EntityInstance{{ID: artifactID}}, nil)
				props.EXPECT().EntityWithPropertiesByID(gomock.Any(), artifactID, gomock.Any()).
					Return(makeEWP("myorg/real-repo"), nil)
			},
			wantCode: codes.NotFound,
		},
		{
			name:        "success — owner and repo match",
			artifactRef: "myorg/myrepo/myartifact",
			setupMocks: func(store *mockdb.MockStore, ps *mockproviders.MockProviderStore, props *mockpropssvc.MockPropertiesService) {
				ps.EXPECT().GetByName(gomock.Any(), projectID, providerName).Return(validProvider, nil)
				store.EXPECT().GetTypedEntitiesByPropertyV1(
					gomock.Any(), db.EntitiesArtifact, properties.PropertyName, "myorg/myartifact",
					db.GetTypedEntitiesOptions{ProjectID: projectID, ProviderID: providerID},
				).Return([]db.EntityInstance{{ID: artifactID}}, nil)
				ewp := makeEWP("myorg/myrepo")
				props.EXPECT().EntityWithPropertiesByID(gomock.Any(), artifactID, gomock.Any()).Return(ewp, nil)
				props.EXPECT().RetrieveAllPropertiesForEntity(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				props.EXPECT().EntityWithPropertiesAsProto(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&pb.Artifact{}, nil)
			},
			wantCode: codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			mockProvStore := mockproviders.NewMockProviderStore(ctrl)
			mockProps := mockpropssvc.NewMockPropertiesService(ctrl)

			tt.setupMocks(mockStore, mockProvStore, mockProps)

			server := Server{
				store:         mockStore,
				providerStore: mockProvStore,
				props:         mockProps,
			}

			resp, err := server.GetArtifactByName(baseCtx, &pb.GetArtifactByNameRequest{
				Name: tt.artifactRef,
			})

			if tt.wantCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, resp)
			} else {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tt.wantCode, st.Code())
			}
		})
	}
}
