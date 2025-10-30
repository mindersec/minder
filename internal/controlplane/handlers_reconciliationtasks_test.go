// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	stubeventer "github.com/mindersec/minder/internal/events/stubs"
	"github.com/mindersec/minder/internal/providers"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

const ghProvider = "github"

func TestServer_CreateRepositoryReconciliationTask(t *testing.T) {
	t.Parallel()

	repoUuid := uuid.New()
	tests := []struct {
		name          string
		input         *pb.CreateEntityReconciliationTaskRequest
		entityContext *engcontext.EntityContext
		setup         func(store *mockdb.MockStore, entityContext *engcontext.EntityContext)
		err           string
	}{
		{
			name: "reconciliation task created",
			input: &pb.CreateEntityReconciliationTaskRequest{
				Entity: &pb.EntityTypedId{
					Type: pb.Entity_ENTITY_REPOSITORIES,
					Id:   repoUuid.String(),
				},
			},
			entityContext: &engcontext.EntityContext{
				Project: engcontext.Project{
					ID: uuid.New(),
				},
				Provider: engcontext.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engcontext.EntityContext) {
				projId := entityContext.Project.ID
				prov := entityContext.Provider.Name
				setupTestingEntityContextValidation(store, projId, prov, uuid.New())
				store.EXPECT().
					GetEntityByID(gomock.Any(), repoUuid).
					Return(db.EntityInstance{
						ID:         repoUuid,
						EntityType: db.EntitiesRepository,
						ProviderID: uuid.New(),
						ProjectID:  projId,
					}, nil)
			},
			err: "",
		},
		{
			name: "create reconciliation by name",
			input: &pb.CreateEntityReconciliationTaskRequest{
				Entity: &pb.EntityTypedId{
					Type: pb.Entity_ENTITY_REPOSITORIES,
					Name: "my/repo",
				},
			},
			entityContext: &engcontext.EntityContext{
				Project: engcontext.Project{
					ID: uuid.New(),
				},
				Provider: engcontext.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engcontext.EntityContext) {
				projId := entityContext.Project.ID
				prov := entityContext.Provider.Name
				provId := uuid.New()
				setupTestingEntityContextValidation(store, projId, prov, provId)
				store.EXPECT().
					GetEntityByName(gomock.Any(), db.GetEntityByNameParams{
						ProjectID:  projId,
						EntityType: "repository",
						Name:       "my/repo",
						ProviderID: provId,
					}).
					Return(db.EntityInstance{ID: repoUuid}, nil)
				store.EXPECT().
					GetEntityByID(gomock.Any(), repoUuid).
					Return(db.EntityInstance{
						ID:         repoUuid,
						EntityType: db.EntitiesRepository,
						ProviderID: uuid.New(),
						ProjectID:  projId,
					}, nil)
			},
			err: "",
		},
		{
			name: "no provider succeeds",
			input: &pb.CreateEntityReconciliationTaskRequest{
				Entity: &pb.EntityTypedId{
					Type: pb.Entity_ENTITY_REPOSITORIES,
					Id:   repoUuid.String(),
				},
			},
			entityContext: &engcontext.EntityContext{
				Project: engcontext.Project{
					ID: uuid.New(),
				},
				Provider: engcontext.Provider{},
			},
			setup: func(store *mockdb.MockStore, entityContext *engcontext.EntityContext) {
				projId := entityContext.Project.ID
				store.EXPECT().
					GetProjectByID(gomock.Any(), projId).
					Return(db.Project{ID: projId}, nil)
				store.EXPECT().
					GetParentProjects(gomock.Any(), projId).
					Return([]uuid.UUID{projId}, nil)
				store.EXPECT().
					FindProviders(gomock.Any(), db.FindProvidersParams{
						Name:     sql.NullString{String: "", Valid: false},
						Projects: []uuid.UUID{projId},
						Trait:    db.NullProviderType{},
					}).Return([]db.Provider{{Name: ghProvider}}, nil)
				store.EXPECT().
					GetEntityByID(gomock.Any(), repoUuid).
					Return(db.EntityInstance{
						ID:         repoUuid,
						EntityType: db.EntitiesRepository,
						ProviderID: uuid.New(),
						ProjectID:  projId,
					}, nil)
			},
			err: "",
		},
		{
			name: "invalid entity context",
			input: &pb.CreateEntityReconciliationTaskRequest{
				Entity: &pb.EntityTypedId{
					Type: pb.Entity_ENTITY_REPOSITORIES,
					Id:   repoUuid.String(),
				},
			},
			entityContext: &engcontext.EntityContext{
				Project: engcontext.Project{
					ID: uuid.New(),
				},
				Provider: engcontext.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engcontext.EntityContext) {
				projId := entityContext.Project.ID
				store.EXPECT().
					GetProjectByID(gomock.Any(), projId).
					Return(db.Project{}, sql.ErrNoRows)
			},
			err: sql.ErrNoRows.Error(),
		},
		{
			name: "repository not found",
			input: &pb.CreateEntityReconciliationTaskRequest{
				Entity: &pb.EntityTypedId{
					Type: pb.Entity_ENTITY_REPOSITORIES,
					Id:   repoUuid.String(),
				},
			},
			entityContext: &engcontext.EntityContext{
				Project: engcontext.Project{
					ID: uuid.New(),
				},
				Provider: engcontext.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engcontext.EntityContext) {
				projId := entityContext.Project.ID
				prov := entityContext.Provider.Name
				setupTestingEntityContextValidation(store, projId, prov, uuid.New())
				store.EXPECT().
					GetEntityByID(gomock.Any(), repoUuid).
					Return(db.EntityInstance{}, sql.ErrNoRows)
			},
			err: "repository not found",
		},
		{
			name: "sql conn done",
			input: &pb.CreateEntityReconciliationTaskRequest{
				Entity: &pb.EntityTypedId{
					Type: pb.Entity_ENTITY_REPOSITORIES,
					Id:   repoUuid.String(),
				},
			},
			entityContext: &engcontext.EntityContext{
				Project: engcontext.Project{
					ID: uuid.New(),
				},
				Provider: engcontext.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engcontext.EntityContext) {
				projId := entityContext.Project.ID
				prov := entityContext.Provider.Name
				setupTestingEntityContextValidation(store, projId, prov, uuid.New())
				store.EXPECT().
					GetEntityByID(gomock.Any(), repoUuid).
					Return(db.EntityInstance{}, sql.ErrConnDone)
			},
			err: sql.ErrConnDone.Error(),
		},
		{
			name: "invalid repo id",
			input: &pb.CreateEntityReconciliationTaskRequest{
				Entity: &pb.EntityTypedId{
					Type: pb.Entity_ENTITY_REPOSITORIES,
					Id:   "foo",
				},
			},
			entityContext: &engcontext.EntityContext{
				Project: engcontext.Project{
					ID: uuid.New(),
				},
				Provider: engcontext.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engcontext.EntityContext) {
				projId := entityContext.Project.ID
				prov := entityContext.Provider.Name
				setupTestingEntityContextValidation(store, projId, prov, uuid.New())
			},
			err: "error parsing repository id",
		},
		{
			name: "invalid entity type",
			input: &pb.CreateEntityReconciliationTaskRequest{
				Entity: &pb.EntityTypedId{
					Type: pb.Entity_ENTITY_UNSPECIFIED,
				},
			},
			entityContext: &engcontext.EntityContext{
				Project: engcontext.Project{
					ID: uuid.New(),
				},
				Provider: engcontext.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engcontext.EntityContext) {
				projId := entityContext.Project.ID
				prov := entityContext.Provider.Name
				setupTestingEntityContextValidation(store, projId, prov, uuid.New())
			},
			err: "entity type ENTITY_UNSPECIFIED is not supported",
		},
		{
			name: "nil entity",
			input: &pb.CreateEntityReconciliationTaskRequest{
				Entity: nil,
			},
			entityContext: &engcontext.EntityContext{
				Project: engcontext.Project{
					ID: uuid.New(),
				},
				Provider: engcontext.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engcontext.EntityContext) {
				projId := entityContext.Project.ID
				prov := entityContext.Provider.Name
				setupTestingEntityContextValidation(store, projId, prov, uuid.New())
			},
			err: "entity is required",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mockdb.NewMockStore(ctrl)
			tt.setup(mockStore, tt.entityContext)

			stubEventer := stubeventer.StubEventer{}

			s := &Server{
				store:         mockStore,
				evt:           &stubEventer,
				providerStore: providers.NewProviderStore(mockStore),
			}

			ctx := engcontext.WithEntityContext(context.Background(), tt.entityContext)
			_, err := s.CreateEntityReconciliationTask(ctx, tt.input)
			if tt.err != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.err)
				return
			}

			require.NoError(t, err)

			messagesSent := 1

			// Length one indicates that the message was sent
			require.Equal(t, messagesSent, len(stubEventer.Sent))
		})
	}
}

func setupTestingEntityContextValidation(store *mockdb.MockStore, projId uuid.UUID, prov string, provId uuid.UUID) {
	store.EXPECT().
		GetProjectByID(gomock.Any(), projId).
		Return(db.Project{ID: projId}, nil)
	store.EXPECT().
		GetParentProjects(gomock.Any(), projId).
		Return([]uuid.UUID{projId}, nil)
	store.EXPECT().
		FindProviders(gomock.Any(), db.FindProvidersParams{
			Name:     sql.NullString{String: prov, Valid: true},
			Projects: []uuid.UUID{projId},
			Trait:    db.NullProviderType{},
		}).Return([]db.Provider{{Name: prov, ID: provId}}, nil)
}
