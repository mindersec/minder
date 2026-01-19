// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package service_test contains tests for the entity service layer.
// These tests focus on provider validation and error handling paths.
// TODO: Add integration tests with real database for happy path scenarios
package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	mockprop "github.com/mindersec/minder/internal/entities/properties/service/mock"
	"github.com/mindersec/minder/internal/entities/service"
	"github.com/mindersec/minder/internal/entities/service/validators"
	mockprov "github.com/mindersec/minder/internal/providers/manager/mock"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	mockevents "github.com/mindersec/minder/pkg/eventer/interfaces/mock"
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
			SupportsEntity(pb.Entity_ENTITY_REPOSITORIES).
			Return(false)

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
			SupportsEntity(pb.Entity_ENTITY_REPOSITORIES).
			Return(true)

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
		mockProv.EXPECT().SupportsEntity(pb.Entity_ENTITY_REPOSITORIES).Return(true)
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
