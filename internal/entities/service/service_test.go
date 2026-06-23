// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/db/embedded"
	propService "github.com/mindersec/minder/internal/entities/properties/service"
	"github.com/mindersec/minder/internal/entities/service"
	mockprov "github.com/mindersec/minder/internal/providers/manager/mock"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

func setupEntityService(ctx context.Context, t *testing.T) (
	service.EntityService, db.Store, propService.PropertiesService, uuid.UUID, db.Provider, *mockprov.MockProviderManager,
) {
	t.Helper()

	store, cancel, err := embedded.GetFakeStore()
	require.NoError(t, err)
	t.Cleanup(cancel)

	propSvc := propService.NewPropertiesService(store)
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	providerManager := mockprov.NewMockProviderManager(ctrl)
	svc := service.NewEntityService(store, propSvc, providerManager)

	projectID := uuid.New()

	// Create project
	_, err = store.CreateProjectWithID(ctx, db.CreateProjectWithIDParams{
		ID:       projectID,
		Name:     "test-project-list",
		Metadata: []byte("{}"),
	})
	require.NoError(t, err)

	// Create provider
	provider, err := store.CreateProvider(ctx, db.CreateProviderParams{
		Name:       "github-list",
		ProjectID:  projectID,
		Class:      db.ProviderClassGithub,
		Implements: []db.ProviderType{db.ProviderTypeGithub, db.ProviderTypeRepoLister},
		Definition: []byte("{}"),
		AuthFlows:  []db.AuthorizationFlow{db.AuthorizationFlowOauth2AuthorizationCodeFlow},
	})
	require.NoError(t, err)

	return svc, store, propSvc, projectID, provider, providerManager
}

func TestEntityService_ListEntities(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	svc, store, propSvc, projectID, provider, _ := setupEntityService(ctx, t)

	// Seed some entities
	nRepos := 5
	for i := range nRepos {
		ei, err := store.CreateEntity(ctx, db.CreateEntityParams{
			EntityType: db.EntitiesRepository,
			Name:       fmt.Sprintf("repo-%d", i),
			ProjectID:  projectID,
			ProviderID: provider.ID,
		})
		require.NoError(t, err)

		// Add some properties
		props := properties.NewProperties(map[string]any{
			"name":              fmt.Sprintf("test-owner/repo-%d", i),
			"github/repo_id":    int64(i),
			"github/repo_name":  fmt.Sprintf("repo-%d", i),
			"github/repo_owner": "test-owner",
		})
		err = propSvc.SaveAllProperties(ctx, ei.ID, props, nil)
		require.NoError(t, err)
	}

	nArtifacts := 3
	for i := range nArtifacts {
		ei, err := store.CreateEntity(ctx, db.CreateEntityParams{
			EntityType: db.EntitiesArtifact,
			Name:       fmt.Sprintf("artifact-%d", i),
			ProjectID:  projectID,
			ProviderID: provider.ID,
		})
		require.NoError(t, err)

		// Add some properties
		props := properties.NewProperties(map[string]any{
			"name": fmt.Sprintf("test-owner/repo-1/artifact-%d", i),
		})
		err = propSvc.SaveAllProperties(ctx, ei.ID, props, nil)
		require.NoError(t, err)
	}

	t.Run("List all entities", func(t *testing.T) {
		t.Parallel()
		results, nextCursor, err := svc.ListEntities(ctx, projectID, provider.ID, pb.Entity_ENTITY_UNSPECIFIED, "", 0)
		assert.NoError(t, err)
		assert.Len(t, results, nRepos+nArtifacts)
		assert.Empty(t, nextCursor)
	})

	t.Run("List repositories only", func(t *testing.T) {
		t.Parallel()
		results, _, err := svc.ListEntities(ctx, projectID, provider.ID, pb.Entity_ENTITY_REPOSITORIES, "", 0)
		assert.NoError(t, err)
		assert.Len(t, results, nRepos)
		for _, r := range results {
			assert.Equal(t, pb.Entity_ENTITY_REPOSITORIES, r.Type)
		}
	})

	t.Run("List artifacts only", func(t *testing.T) {
		t.Parallel()
		results, _, err := svc.ListEntities(ctx, projectID, provider.ID, pb.Entity_ENTITY_ARTIFACTS, "", 0)
		assert.NoError(t, err)
		assert.Len(t, results, nArtifacts)
		for _, r := range results {
			assert.Equal(t, pb.Entity_ENTITY_ARTIFACTS, r.Type)
		}
	})

	t.Run("Pagination with limit", func(t *testing.T) {
		t.Parallel()
		limit := 3
		results, nextCursor, err := svc.ListEntities(ctx, projectID, provider.ID, pb.Entity_ENTITY_UNSPECIFIED, "", int64(limit))
		assert.NoError(t, err)
		assert.Len(t, results, limit)
		assert.NotEmpty(t, nextCursor)

		// Get next page
		results2, nextCursor2, err := svc.ListEntities(ctx, projectID, provider.ID, pb.Entity_ENTITY_UNSPECIFIED, nextCursor, int64(limit))
		assert.NoError(t, err)
		assert.Len(t, results2, limit)
		assert.NotEmpty(t, nextCursor2)

		// Get last page
		results3, nextCursor3, err := svc.ListEntities(ctx, projectID, provider.ID, pb.Entity_ENTITY_UNSPECIFIED, nextCursor2, int64(limit))
		assert.NoError(t, err)
		assert.Len(t, results3, (nRepos+nArtifacts)-(2*limit))
		assert.Empty(t, nextCursor3)
	})

	t.Run("Empty results for non-existent provider", func(t *testing.T) {
		t.Parallel()
		results, nextCursor, err := svc.ListEntities(ctx, projectID, uuid.New(), pb.Entity_ENTITY_UNSPECIFIED, "", 0)
		assert.NoError(t, err)
		assert.Empty(t, results)
		assert.Empty(t, nextCursor)
	})
}

func TestEntityService_GetEntity(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	svc, store, propSvc, projectID, provider, providerManager := setupEntityService(ctx, t)

	// Seed an entity
	name := "test-owner/repo-to-get"
	ei, err := store.CreateEntity(ctx, db.CreateEntityParams{
		EntityType: db.EntitiesRepository,
		Name:       name,
		ProjectID:  projectID,
		ProviderID: provider.ID,
	})
	require.NoError(t, err)

	// Add some properties
	props := properties.NewProperties(map[string]any{
		"name":              name,
		"github/repo_id":    123,
		"github/repo_name":  "repo-to-get",
		"github/repo_owner": "test-owner",
	})
	err = propSvc.SaveAllProperties(ctx, ei.ID, props, nil)
	require.NoError(t, err)

	t.Run("GetEntityByID", func(t *testing.T) {
		t.Parallel()
		providerManager.EXPECT().
			InstantiateFromID(gomock.Any(), provider.ID).
			Return(nil, nil)

		result, err := svc.GetEntityByID(ctx, ei.ID, projectID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, ei.ID.String(), result.Id)
		assert.Equal(t, name, result.Name)
	})

	t.Run("GetEntityByName", func(t *testing.T) {
		t.Parallel()
		providerManager.EXPECT().
			InstantiateFromID(gomock.Any(), provider.ID).
			Return(nil, nil)

		result, err := svc.GetEntityByName(ctx, name, projectID, provider.ID, pb.Entity_ENTITY_REPOSITORIES)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, ei.ID.String(), result.Id)
		assert.Equal(t, name, result.Name)
	})

	t.Run("GetEntityByID - Not Found", func(t *testing.T) {
		t.Parallel()
		result, err := svc.GetEntityByID(ctx, uuid.New(), projectID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("GetEntityByName - Not Found", func(t *testing.T) {
		t.Parallel()
		result, err := svc.GetEntityByName(ctx, "my-repo/non-existent", projectID, provider.ID, pb.Entity_ENTITY_REPOSITORIES)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestEntityService_DeleteEntity(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	svc, store, propSvc, projectID, provider, _ := setupEntityService(ctx, t)

	// Seed an entity
	name := "test-owner/repo-to-delete"
	ei, err := store.CreateEntity(ctx, db.CreateEntityParams{
		EntityType: db.EntitiesRepository,
		Name:       name,
		ProjectID:  projectID,
		ProviderID: provider.ID,
	})
	require.NoError(t, err)

	// Add some properties
	props := properties.NewProperties(map[string]any{
		"name":              name,
		"github/repo_id":    12345,
		"github/repo_name":  "repo-to-delete",
		"github/repo_owner": "test-owner",
	})
	err = propSvc.SaveAllProperties(ctx, ei.ID, props, nil)
	require.NoError(t, err)

	t.Run("DeleteEntityByID", func(t *testing.T) {
		t.Parallel()
		err := svc.DeleteEntityByID(ctx, ei.ID, projectID)
		assert.NoError(t, err)

		// Verify it's gone
		_, err = store.GetEntityByID(ctx, ei.ID)
		assert.Error(t, err)
	})

	t.Run("DeleteEntityByID - Not Found", func(t *testing.T) {
		t.Parallel()
		err := svc.DeleteEntityByID(ctx, uuid.New(), projectID)
		assert.Error(t, err)
	})
}
