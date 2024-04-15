//
// Copyright 2024 Stacklok, Inc.
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

package projects_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/authz"
	"github.com/stacklok/minder/internal/authz/mock"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/projects"
	mockprofsvc "github.com/stacklok/minder/internal/providers/mock"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestDeleteProjectOneProjectWithNoParents(t *testing.T) {
	t.Parallel()

	proj := uuid.New()

	authzClient := &mock.SimpleClient{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().GetProjectByID(gomock.Any(), proj).Return(
		db.Project{ID: proj}, nil)
	mockStore.EXPECT().DeleteProject(gomock.Any(), proj).
		Return([]db.DeleteProjectRow{
			{ID: proj},
		}, nil)
	mockStore.EXPECT().ListProvidersByProjectID(gomock.Any(), []uuid.UUID{proj}).
		Return([]db.Provider{}, nil)

	mockProviderService := mockprofsvc.NewMockProviderService(ctrl)

	ctx := context.Background()

	tl := zerolog.NewTestWriter(t)
	l := zerolog.New(tl)
	err := projects.DeleteProject(ctx, proj, mockStore, authzClient, mockProviderService, l)
	assert.NoError(t, err)

	// Ensure there are no calls to the orphan cleanup function
	assert.Equal(t, int32(0), authzClient.OrphanCalls.Load())
}

func TestDeleteProjectWithOneParent(t *testing.T) {
	t.Parallel()

	proj := uuid.New()
	parent := uuid.New()

	authzClient := &mock.SimpleClient{
		Adoptions: map[uuid.UUID]uuid.UUID{
			proj: parent,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().GetProjectByID(gomock.Any(), proj).Return(
		db.Project{
			ID: proj,
			ParentID: uuid.NullUUID{
				UUID:  parent,
				Valid: true,
			},
		}, nil)
	mockStore.EXPECT().ListProvidersByProjectID(gomock.Any(), []uuid.UUID{proj}).
		Return([]db.Provider{}, nil)
	mockStore.EXPECT().DeleteProject(gomock.Any(), proj).
		Return([]db.DeleteProjectRow{
			{
				ID: proj,
				ParentID: uuid.NullUUID{
					UUID:  parent,
					Valid: true,
				},
			},
		}, nil)

	mockProviderService := mockprofsvc.NewMockProviderService(ctrl)

	ctx := context.Background()

	tl := zerolog.NewTestWriter(t)
	l := zerolog.New(tl)
	err := projects.DeleteProject(ctx, proj, mockStore, authzClient, mockProviderService, l)
	assert.NoError(t, err)

	// Ensure there is one call to the orphan cleanup function
	assert.Equal(t, int32(1), authzClient.OrphanCalls.Load())
}

func TestDeleteProjectProjectInThreeNodeHierarchy(t *testing.T) {
	t.Parallel()

	proj := uuid.New()
	parent := uuid.New()
	grandparent := uuid.New()

	authzClient := &mock.SimpleClient{
		Adoptions: map[uuid.UUID]uuid.UUID{
			proj:   parent,
			parent: grandparent,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().GetProjectByID(gomock.Any(), proj).Return(
		db.Project{
			ID: proj,
			ParentID: uuid.NullUUID{
				UUID:  parent,
				Valid: true,
			},
		}, nil)
	mockStore.EXPECT().ListProvidersByProjectID(gomock.Any(), []uuid.UUID{proj}).
		Return([]db.Provider{}, nil)
	mockStore.EXPECT().DeleteProject(gomock.Any(), proj).
		Return([]db.DeleteProjectRow{
			{
				ID: proj,
				ParentID: uuid.NullUUID{
					UUID:  parent,
					Valid: true,
				},
			},
		}, nil)

	mockProviderService := mockprofsvc.NewMockProviderService(ctrl)

	ctx := context.Background()

	tl := zerolog.NewTestWriter(t)
	l := zerolog.New(tl)
	err := projects.DeleteProject(ctx, proj, mockStore, authzClient, mockProviderService, l)
	assert.NoError(t, err)

	// Ensure there is one call to the orphan cleanup function
	assert.Equal(t, int32(1), authzClient.OrphanCalls.Load())

	// Ensure the adoption relationship was removed
	assert.NotContains(t, authzClient.Adoptions, proj)

	// Ensure the grandparent relationship was not removed
	assert.Contains(t, authzClient.Adoptions, parent)
}

func TestDeleteMiddleProjectInThreeNodeHierarchy(t *testing.T) {
	t.Parallel()

	child := uuid.New()
	proj := uuid.New()
	parent := uuid.New()

	authzClient := &mock.SimpleClient{
		Adoptions: map[uuid.UUID]uuid.UUID{
			child: proj,
			proj:  parent,
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().GetProjectByID(gomock.Any(), proj).Return(
		db.Project{
			ID: proj,
			ParentID: uuid.NullUUID{
				UUID:  parent,
				Valid: true,
			},
		}, nil)
	mockStore.EXPECT().ListProvidersByProjectID(gomock.Any(), []uuid.UUID{proj}).
		Return([]db.Provider{}, nil)
	mockStore.EXPECT().DeleteProject(gomock.Any(), proj).
		Return([]db.DeleteProjectRow{
			{
				ID: proj,
				ParentID: uuid.NullUUID{
					UUID:  parent,
					Valid: true,
				},
			},
			{
				ID: child,
				ParentID: uuid.NullUUID{
					UUID:  proj,
					Valid: true,
				},
			},
		}, nil)

	mockProviderService := mockprofsvc.NewMockProviderService(ctrl)

	ctx := context.Background()

	tl := zerolog.NewTestWriter(t)
	l := zerolog.New(tl)
	err := projects.DeleteProject(ctx, proj, mockStore, authzClient, mockProviderService, l)
	assert.NoError(t, err)

	// Ensure there are two calls to the orphan cleanup function
	assert.Equal(t, int32(2), authzClient.OrphanCalls.Load())

	// Ensure the adoption relationships were removed
	assert.NotContains(t, authzClient.Adoptions, proj)
	assert.NotContains(t, authzClient.Adoptions, child)
	assert.Len(t, authzClient.Adoptions, 0)
}

func TestDeleteProjectWithProvider(t *testing.T) {
	t.Parallel()

	proj := uuid.New()

	authzClient := &mock.SimpleClient{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().GetProjectByID(gomock.Any(), proj).Return(
		db.Project{ID: proj}, nil)
	mockStore.EXPECT().DeleteProject(gomock.Any(), proj).
		Return([]db.DeleteProjectRow{
			{ID: proj},
		}, nil)
	mockStore.EXPECT().ListProvidersByProjectID(gomock.Any(), []uuid.UUID{proj}).
		Return([]db.Provider{
			{ID: uuid.UUID{}},
		}, nil)

	mockProviderService := mockprofsvc.NewMockProviderService(ctrl)
	mockProviderService.EXPECT().DeleteProvider(gomock.Any(), gomock.Any()).Return(nil)

	ctx := context.Background()

	tl := zerolog.NewTestWriter(t)
	l := zerolog.New(tl)
	err := projects.DeleteProject(ctx, proj, mockStore, authzClient, mockProviderService, l)
	assert.NoError(t, err)

	// Ensure there are no calls to the orphan cleanup function
	assert.Equal(t, int32(0), authzClient.OrphanCalls.Load())
}

func TestCleanupUnmanaged(t *testing.T) {
	t.Parallel()

	projOne := uuid.New()
	projChild := uuid.New()
	projTwo := uuid.New()
	projThree := uuid.New()

	authzClient := &mock.SimpleClient{
		Assignments: map[uuid.UUID][]*minderv1.RoleAssignment{
			projTwo: {{
				Role:    authz.AuthzRoleAdmin.String(),
				Subject: "user1",
			}},
			projThree: {{
				Role:    authz.AuthzRoleViewer.String(),
				Subject: "user1",
			}},
		},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	mockStore.EXPECT().GetProjectByID(gomock.Any(), projChild).Return(
		db.Project{ID: projChild, ParentID: uuid.NullUUID{UUID: projOne, Valid: true}}, nil)
	mockStore.EXPECT().GetProjectByID(gomock.Any(), projTwo).Return(
		db.Project{ID: projTwo}, nil)
	mockStore.EXPECT().GetProjectByID(gomock.Any(), projThree).Return(
		db.Project{ID: projThree}, nil).Times(2)
	// Project 3 has no other admins, so it will be deleted.
	mockStore.EXPECT().DeleteProject(gomock.Any(), projThree).Return(
		[]db.DeleteProjectRow{
			{ID: projThree},
		}, nil)
	mockStore.EXPECT().ListProvidersByProjectID(gomock.Any(), []uuid.UUID{projThree}).
		Return([]db.Provider{
			{ID: uuid.UUID{}},
		}, nil)

	mockProviderService := mockprofsvc.NewMockProviderService(ctrl)
	mockProviderService.EXPECT().DeleteProvider(gomock.Any(), gomock.Any()).Return(nil)

	err := projects.CleanUpUnmanagedProjects(context.Background(), projChild, mockStore, authzClient, mockProviderService)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = projects.CleanUpUnmanagedProjects(context.Background(), projTwo, mockStore, authzClient, mockProviderService)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = projects.CleanUpUnmanagedProjects(context.Background(), projThree, mockStore, authzClient, mockProviderService)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
