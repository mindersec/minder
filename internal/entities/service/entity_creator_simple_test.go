// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package service_test contains tests for the entity service layer.
// These tests focus on provider validation and error handling paths.
package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/db/embedded"
	propService "github.com/mindersec/minder/internal/entities/properties/service"
	mockprop "github.com/mindersec/minder/internal/entities/properties/service/mock"
	"github.com/mindersec/minder/internal/entities/service"
	"github.com/mindersec/minder/internal/entities/service/validators"
	mockprov "github.com/mindersec/minder/internal/providers/manager/mock"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	mockevents "github.com/mindersec/minder/pkg/eventer/interfaces/mock"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
	mockprovidersv1 "github.com/mindersec/minder/pkg/providers/v1/mock"
)

// TestEntityCreator_ProviderValidation tests provider-related validation
func TestEntityCreator_ProviderValidation(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	providerID := uuid.New()
	testProvider := &db.Provider{
		ID:        providerID,
		Name:      "test-provider",
		ProjectID: projectID,
	}

	identifyingProps := properties.NewProperties(map[string]any{
		"upstream_id": "12345",
	})

	t.Run("fails when provider cannot be instantiated", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockdb.NewMockStore(ctrl)
		mockPropSvc := mockprop.NewMockPropertiesService(ctrl)
		mockProvMgr := mockprov.NewMockProviderManager(ctrl)
		mockEvt := mockevents.NewMockInterface(ctrl)

		mockProvMgr.EXPECT().
			InstantiateFromID(gomock.Any(), providerID).
			Return(nil, errors.New("provider error"))

		registry := validators.NewValidatorRegistry()
		creator := service.NewEntityCreator(mockStore, mockPropSvc, mockProvMgr, mockEvt, registry)

		_, err := creator.CreateEntity(context.Background(), testProvider, projectID,
			pb.Entity_ENTITY_REPOSITORIES, identifyingProps, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error instantiating provider")
	})

	t.Run("fails when provider doesn't support entity type", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockdb.NewMockStore(ctrl)
		mockPropSvc := mockprop.NewMockPropertiesService(ctrl)
		mockProvMgr := mockprov.NewMockProviderManager(ctrl)
		mockProv := mockprovidersv1.NewMockGitHub(ctrl)
		mockEvt := mockevents.NewMockInterface(ctrl)

		mockProvMgr.EXPECT().
			InstantiateFromID(gomock.Any(), providerID).
			Return(mockProv, nil)

		mockProv.EXPECT().
			CreationOptions(pb.Entity_ENTITY_REPOSITORIES).
			Return(nil) // Returns nil to indicate entity type is not supported

		registry := validators.NewValidatorRegistry()
		creator := service.NewEntityCreator(mockStore, mockPropSvc, mockProvMgr, mockEvt, registry)

		_, err := creator.CreateEntity(context.Background(), testProvider, projectID,
			pb.Entity_ENTITY_REPOSITORIES, identifyingProps, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not support entity type")
	})

	t.Run("fails when property fetching fails", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockdb.NewMockStore(ctrl)
		mockPropSvc := mockprop.NewMockPropertiesService(ctrl)
		mockProvMgr := mockprov.NewMockProviderManager(ctrl)
		mockProv := mockprovidersv1.NewMockGitHub(ctrl)
		mockEvt := mockevents.NewMockInterface(ctrl)

		mockProvMgr.EXPECT().
			InstantiateFromID(gomock.Any(), providerID).
			Return(mockProv, nil)

		mockProv.EXPECT().
			CreationOptions(pb.Entity_ENTITY_REPOSITORIES).
			Return(&provifv1.EntityCreationOptions{
				RegisterWithProvider:       true,
				PublishReconciliationEvent: true,
			})

		mockProv.EXPECT().
			FetchAllProperties(gomock.Any(), identifyingProps, pb.Entity_ENTITY_REPOSITORIES, nil).
			Return(nil, errors.New("API error"))

		registry := validators.NewValidatorRegistry()
		creator := service.NewEntityCreator(mockStore, mockPropSvc, mockProvMgr, mockEvt, registry)

		_, err := creator.CreateEntity(context.Background(), testProvider, projectID,
			pb.Entity_ENTITY_REPOSITORIES, identifyingProps, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error fetching properties")
	})
}

// TestEntityCreator_ValidationFlow tests validator integration
func TestEntityCreator_ValidationFlow(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	providerID := uuid.New()
	testProvider := &db.Provider{
		ID:        providerID,
		Name:      "test-provider",
		ProjectID: projectID,
	}

	identifyingProps := properties.NewProperties(map[string]any{
		"upstream_id": "12345",
	})

	archivedProps := properties.NewProperties(map[string]any{
		"is_archived": true,
	})

	t.Run("runs validators and fails on validation error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStore := mockdb.NewMockStore(ctrl)
		mockPropSvc := mockprop.NewMockPropertiesService(ctrl)
		mockProvMgr := mockprov.NewMockProviderManager(ctrl)
		mockProv := mockprovidersv1.NewMockGitHub(ctrl)
		mockEvt := mockevents.NewMockInterface(ctrl)

		mockProvMgr.EXPECT().InstantiateFromID(gomock.Any(), providerID).Return(mockProv, nil)
		mockProv.EXPECT().
			CreationOptions(pb.Entity_ENTITY_REPOSITORIES).
			Return(&provifv1.EntityCreationOptions{
				RegisterWithProvider:       true,
				PublishReconciliationEvent: true,
			})
		mockProv.EXPECT().
			FetchAllProperties(gomock.Any(), identifyingProps, pb.Entity_ENTITY_REPOSITORIES, nil).
			Return(archivedProps, nil)

		// Create registry with a validator that rejects archived repos
		registry := validators.NewValidatorRegistry()
		testValidator := &testEntityValidator{
			shouldFail: true,
			failError:  errors.New("validation failed"),
		}
		registry.AddValidator(pb.Entity_ENTITY_REPOSITORIES, testValidator)

		creator := service.NewEntityCreator(mockStore, mockPropSvc, mockProvMgr, mockEvt, registry)

		_, err := creator.CreateEntity(context.Background(), testProvider, projectID,
			pb.Entity_ENTITY_REPOSITORIES, identifyingProps, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})
}

// testEntityValidator is a simple test validator that implements validators.Validator
type testEntityValidator struct {
	shouldFail bool
	failError  error
}

func (v *testEntityValidator) Validate(_ context.Context, _ *properties.Properties, _ uuid.UUID) error {
	if v.shouldFail {
		return v.failError
	}
	return nil
}

// helper to seed a valid project and provider to satisfy foreign key constraint
func seedProjectAndProvider(ctx context.Context, t *testing.T, store db.Store) (db.Project, db.Provider) {
	t.Helper()

	project, err := store.CreateProject(ctx, db.CreateProjectParams{
		Name:     "test-project-" + uuid.NewString(),
		Metadata: []byte(`{}`),
	})
	require.NoError(t, err)

	provider, err := store.CreateProvider(ctx, db.CreateProviderParams{
		Name:       "github-" + uuid.NewString(),
		ProjectID:  project.ID,
		Class:      db.ProviderClassGithub,
		Implements: []db.ProviderType{db.ProviderTypeGithub},
		AuthFlows:  []db.AuthorizationFlow{},
		Definition: []byte(`{}`),
	})

	require.NoError(t, err)
	return project, provider
}

func TestEntityCreator_Integration_HappyPath(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	realStore, cleanup, err := embedded.GetFakeStore()
	require.NoError(t, err)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	project, dbProvider := seedProjectAndProvider(ctx, t, realStore)
	realPropSvc := propService.NewPropertiesService(realStore)
	mockProvMgr := mockprov.NewMockProviderManager(ctrl)
	mockProv := mockprovidersv1.NewMockGitHub(ctrl)
	mockEvt := mockevents.NewMockInterface(ctrl)

	mockProvMgr.EXPECT().InstantiateFromID(ctx, dbProvider.ID).Return(mockProv, nil)
	mockProv.EXPECT().CreationOptions(pb.Entity_ENTITY_REPOSITORIES).Return(&provifv1.EntityCreationOptions{
		RegisterWithProvider: true,
	})

	fetchedProps := properties.NewProperties(map[string]any{"name": "my-test-repo"})
	mockProv.EXPECT().FetchAllProperties(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchedProps, nil)
	mockProv.EXPECT().GetEntityName(gomock.Any(), gomock.Any()).Return("my-test-repo", nil)
	mockProv.EXPECT().RegisterEntity(gomock.Any(), gomock.Any(), gomock.Any()).Return(fetchedProps, nil)

	creator := service.NewEntityCreator(realStore, realPropSvc, mockProvMgr, mockEvt, validators.NewValidatorRegistry())

	res, err := creator.CreateEntity(ctx, &dbProvider, project.ID, pb.Entity_ENTITY_REPOSITORIES, nil, nil)

	require.NoError(t, err)
	dbEnt, err := realStore.GetEntityByID(ctx, db.GetEntityByIDParams{
		ID:         res.Entity.ID,
		ProjectID:  project.ID,
		ProviderID: dbProvider.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, db.EntitiesRepository, dbEnt.EntityType)

	savedEntity, _ := realPropSvc.EntityWithPropertiesByID(ctx, res.Entity.ID, project.ID, dbProvider.ID, nil)
	assert.Contains(t, fmt.Sprintf("%+v", savedEntity.Properties), "my-test-repo")
}

func TestEntityCreator_Integration_WithOriginator(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	realStore, cleanup, err := embedded.GetFakeStore()
	require.NoError(t, err)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	project, dbProvider := seedProjectAndProvider(ctx, t, realStore)
	realPropSvc := propService.NewPropertiesService(realStore)
	mockProvMgr := mockprov.NewMockProviderManager(ctrl)
	mockProv := mockprovidersv1.NewMockGitHub(ctrl)
	mockEvt := mockevents.NewMockInterface(ctrl)

	// seed parent entity
	parentID := uuid.New()
	_, err = realStore.CreateOrEnsureEntityByID(ctx, db.CreateOrEnsureEntityByIDParams{
		ID:         parentID,
		EntityType: db.EntitiesRepository,
		Name:       "parent-repo",
		ProjectID:  project.ID,
		ProviderID: dbProvider.ID,
	})
	require.NoError(t, err)

	mockProvMgr.EXPECT().InstantiateFromID(gomock.Any(), dbProvider.ID).Return(mockProv, nil)
	mockProv.EXPECT().CreationOptions(gomock.Any()).Return(&provifv1.EntityCreationOptions{})
	mockProv.EXPECT().FetchAllProperties(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(properties.NewProperties(map[string]any{"name": "child-artifact"}), nil)
	mockProv.EXPECT().GetEntityName(gomock.Any(), gomock.Any()).Return("child-artifact", nil)

	creator := service.NewEntityCreator(realStore, realPropSvc, mockProvMgr, mockEvt, validators.NewValidatorRegistry())

	// execute with originatingEntityID
	res, err := creator.CreateEntity(ctx, &dbProvider, project.ID, pb.Entity_ENTITY_ARTIFACTS, nil, &service.EntityCreationOptions{
		OriginatingEntityID: &parentID,
	})

	require.NoError(t, err)
	dbEnt, _ := realStore.GetEntityByID(ctx, db.GetEntityByIDParams{
		ID:         res.Entity.ID,
		ProjectID:  project.ID,
		ProviderID: dbProvider.ID,
	})
	assert.Equal(t, parentID, dbEnt.OriginatedFrom.UUID)
}

func TestEntityCreator_Integration_RollbackCleanup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	realStore, cleanup, err := embedded.GetFakeStore()
	require.NoError(t, err)
	defer cleanup()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	project, dbProvider := seedProjectAndProvider(ctx, t, realStore)
	mockPropSvc := mockprop.NewMockPropertiesService(ctrl)
	mockProvMgr := mockprov.NewMockProviderManager(ctrl)
	mockProv := mockprovidersv1.NewMockGitHub(ctrl)
	mockEvt := mockevents.NewMockInterface(ctrl)

	// setup props to be returned by registration
	testProps := properties.NewProperties(map[string]any{"name": "fail-repo"})

	mockProvMgr.EXPECT().InstantiateFromID(gomock.Any(), dbProvider.ID).Return(mockProv, nil)
	mockProv.EXPECT().CreationOptions(gomock.Any()).Return(&provifv1.EntityCreationOptions{RegisterWithProvider: true})
	mockProv.EXPECT().FetchAllProperties(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(testProps, nil)
	mockProv.EXPECT().GetEntityName(gomock.Any(), gomock.Any()).Return("fail-repo", nil)

	// return the props here so the cleanup logic has something to work with
	mockProv.EXPECT().RegisterEntity(gomock.Any(), gomock.Any(), gomock.Any()).Return(testProps, nil)

	// db save fails
	mockPropSvc.EXPECT().ReplaceAllProperties(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(errors.New("database explosion"))

	// ensure we match the props being deregistered
	mockProv.EXPECT().DeregisterEntity(gomock.Any(), pb.Entity_ENTITY_REPOSITORIES, testProps).Return(nil)

	creator := service.NewEntityCreator(realStore, mockPropSvc, mockProvMgr, mockEvt, validators.NewValidatorRegistry())

	_, err = creator.CreateEntity(ctx, &dbProvider, project.ID, pb.Entity_ENTITY_REPOSITORIES, nil, nil)
	assert.ErrorContains(t, err, "database explosion")
}
