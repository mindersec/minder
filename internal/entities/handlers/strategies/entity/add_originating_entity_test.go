// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/entities/handlers/message"
	"github.com/mindersec/minder/internal/entities/models"
	mockPropSvc "github.com/mindersec/minder/internal/entities/properties/service/mock"
	mockEntSvc "github.com/mindersec/minder/internal/entities/service/mock"
	mockManager "github.com/mindersec/minder/internal/providers/manager/mock"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestAddOriginatingEntityStrategy_AlreadyExists(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	providerID := uuid.New()
	projectID := uuid.New()

	parentEwp := &models.EntityWithProperties{
		Entity: models.EntityInstance{
			ID:         uuid.New(),
			Type:       minderv1.Entity_ENTITY_REPOSITORIES,
			ProviderID: providerID,
			ProjectID:  projectID,
		},
	}
	existingEwp := &models.EntityWithProperties{
		Entity: models.EntityInstance{
			ID:         uuid.New(),
			Type:       minderv1.Entity_ENTITY_ARTIFACTS,
			ProviderID: providerID,
			ProjectID:  projectID,
		},
	}

	store := mockdb.NewMockStore(ctrl)
	propSvc := mockPropSvc.NewMockPropertiesService(ctrl)
	provMgr := mockManager.NewMockProviderManager(ctrl)
	entityCreator := mockEntSvc.NewMockEntityCreator(ctrl)

	gomock.InOrder(
		propSvc.EXPECT().
			EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES,
				gomock.Any(), gomock.Any(), gomock.Any()).
			Return(parentEwp, nil),
		propSvc.EXPECT().
			EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_ARTIFACTS,
				gomock.Any(), gomock.Any(), gomock.Any()).
			Return(existingEwp, nil),
	)
	store.EXPECT().
		GetProviderByID(gomock.Any(), providerID).
		Return(db.Provider{ID: providerID}, nil)
	entityCreator.EXPECT().
		CreateEntity(gomock.Any(), gomock.Any(), projectID, minderv1.Entity_ENTITY_ARTIFACTS,
			gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("error creating entity: %w", &pq.Error{Code: pqerror.UniqueViolation}))
	propSvc.EXPECT().
		RetrieveAllPropertiesForEntity(gomock.Any(), existingEwp, provMgr, gomock.Any()).
		Return(nil)

	strategy := NewAddOriginatingEntityStrategy(propSvc, provMgr, store, entityCreator)

	entMsg := message.NewEntityRefreshAndDoMessage().
		WithEntity(minderv1.Entity_ENTITY_ARTIFACTS, properties.NewProperties(map[string]any{
			properties.PropertyUpstreamID: "123",
		})).
		WithOriginator(minderv1.Entity_ENTITY_REPOSITORIES, properties.NewProperties(map[string]any{
			properties.PropertyUpstreamID: "456",
		}))

	ewp, err := strategy.GetEntity(context.Background(), entMsg)
	require.NoError(t, err)
	require.Equal(t, existingEwp, ewp)
}
