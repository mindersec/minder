// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	stubeventer "github.com/stacklok/minder/internal/events/stubs"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestServer_CreateRepositoryReconciliationTask(t *testing.T) {
	t.Parallel()

	ghProvider := "github"
	repoUuid := uuid.New()
	tests := []struct {
		name          string
		input         *pb.CreateEntityReconciliationTaskRequest
		entityContext *engine.EntityContext
		setup         func(store *mockdb.MockStore, entityContext *engine.EntityContext)
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
			entityContext: &engine.EntityContext{
				Project: engine.Project{
					ID: uuid.New(),
				},
				Provider: engine.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engine.EntityContext) {
				projId := entityContext.Project.ID
				prov := entityContext.Provider.Name
				setupTestingEntityContextValidation(store, projId, prov)
				store.EXPECT().
					GetRepositoryByIDAndProject(gomock.Any(), db.GetRepositoryByIDAndProjectParams{
						ID:        repoUuid,
						ProjectID: projId,
					}).
					Return(db.Repository{
						ID:        repoUuid,
						Provider:  prov,
						ProjectID: projId,
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
			entityContext: &engine.EntityContext{
				Project: engine.Project{
					ID: uuid.New(),
				},
				Provider: engine.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engine.EntityContext) {
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
			entityContext: &engine.EntityContext{
				Project: engine.Project{
					ID: uuid.New(),
				},
				Provider: engine.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engine.EntityContext) {
				projId := entityContext.Project.ID
				prov := entityContext.Provider.Name
				setupTestingEntityContextValidation(store, projId, prov)
				store.EXPECT().
					GetRepositoryByIDAndProject(gomock.Any(), db.GetRepositoryByIDAndProjectParams{
						ID:        repoUuid,
						ProjectID: projId,
					}).
					Return(db.Repository{}, sql.ErrNoRows)
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
			entityContext: &engine.EntityContext{
				Project: engine.Project{
					ID: uuid.New(),
				},
				Provider: engine.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engine.EntityContext) {
				projId := entityContext.Project.ID
				prov := entityContext.Provider.Name
				setupTestingEntityContextValidation(store, projId, prov)
				store.EXPECT().
					GetRepositoryByIDAndProject(gomock.Any(), db.GetRepositoryByIDAndProjectParams{
						ID:        repoUuid,
						ProjectID: projId,
					}).
					Return(db.Repository{}, sql.ErrConnDone)
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
			entityContext: &engine.EntityContext{
				Project: engine.Project{
					ID: uuid.New(),
				},
				Provider: engine.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engine.EntityContext) {
				projId := entityContext.Project.ID
				prov := entityContext.Provider.Name
				setupTestingEntityContextValidation(store, projId, prov)
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
			entityContext: &engine.EntityContext{
				Project: engine.Project{
					ID: uuid.New(),
				},
				Provider: engine.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engine.EntityContext) {
				projId := entityContext.Project.ID
				prov := entityContext.Provider.Name
				setupTestingEntityContextValidation(store, projId, prov)
			},
			err: "entity type ENTITY_UNSPECIFIED is not supported",
		},
		{
			name: "nil entity",
			input: &pb.CreateEntityReconciliationTaskRequest{
				Entity: nil,
			},
			entityContext: &engine.EntityContext{
				Project: engine.Project{
					ID: uuid.New(),
				},
				Provider: engine.Provider{
					Name: ghProvider,
				},
			},
			setup: func(store *mockdb.MockStore, entityContext *engine.EntityContext) {
				projId := entityContext.Project.ID
				prov := entityContext.Provider.Name
				setupTestingEntityContextValidation(store, projId, prov)
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
				store: mockStore,
				evt:   &stubEventer,
			}

			ctx := engine.WithEntityContext(context.Background(), tt.entityContext)
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

func setupTestingEntityContextValidation(store *mockdb.MockStore, projId uuid.UUID, prov string) {
	store.EXPECT().
		GetProjectByID(gomock.Any(), projId).
		Return(db.Project{ID: projId}, nil)
	store.EXPECT().
		GetParentProjects(gomock.Any(), projId).
		Return([]uuid.UUID{projId}, nil)
	store.EXPECT().
		GetProviderByName(gomock.Any(), db.GetProviderByNameParams{
			Name:     prov,
			Projects: []uuid.UUID{projId},
		}).Return(db.Provider{Name: prov}, nil)
}
