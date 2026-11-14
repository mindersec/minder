// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/entities/handlers/message"
	strategiesEntity "github.com/mindersec/minder/internal/entities/handlers/strategies/entity"
	"github.com/mindersec/minder/internal/entities/models"
	mockPropSvc "github.com/mindersec/minder/internal/entities/properties/service/mock"
	mockCreator "github.com/mindersec/minder/internal/entities/service/mock"
	mockProvMgr "github.com/mindersec/minder/internal/providers/manager/mock"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestAddOriginatingEntityStrategy(t *testing.T) {
	t.Parallel()

	provID := uuid.New()
	projID := uuid.New()
	parentID := uuid.New()
	childID := uuid.New()

	parentEwp := &models.EntityWithProperties{
		Entity: models.EntityInstance{
			ID:         parentID,
			Type:       minderv1.Entity_ENTITY_REPOSITORIES,
			ProviderID: provID,
			ProjectID:  projID,
		},
	}

	provider := db.Provider{ID: provID, Name: "github"}

	childEwp := &models.EntityWithProperties{
		Entity: models.EntityInstance{
			ID:         childID,
			Type:       minderv1.Entity_ENTITY_PULL_REQUESTS,
			ProviderID: provID,
			ProjectID:  projID,
		},
		Properties: properties.NewProperties(map[string]any{"number": 42}),
	}

	baseMsg := &message.HandleEntityAndDoMessage{
		Originator: message.TypedProps{
			Type:       minderv1.Entity_ENTITY_REPOSITORIES,
			GetByProps: map[string]any{"repo_id": 100},
		},
		Entity: message.TypedProps{
			Type:       minderv1.Entity_ENTITY_PULL_REQUESTS,
			GetByProps: map[string]any{"number": 42},
		},
	}

	testCases := []struct {
		name        string
		setupMocks  func(p *mockPropSvc.MockPropertiesService, s *mockdb.MockStore, c *mockCreator.MockEntityCreator)
		errContains string
	}{
		{
			name: "fails when resolving parent entity errors",
			setupMocks: func(p *mockPropSvc.MockPropertiesService, _ *mockdb.MockStore, _ *mockCreator.MockEntityCreator) {
				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("parent entity lookup failed"))
			},
			errContains: "error getting parent entity",
		},
		{
			name: "fails when resolving provider from store errors",
			setupMocks: func(p *mockPropSvc.MockPropertiesService, s *mockdb.MockStore, _ *mockCreator.MockEntityCreator) {
				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(parentEwp, nil)
				s.EXPECT().
					GetProviderByID(gomock.Any(), provID).
					Return(db.Provider{}, errors.New("provider not found in DB"))
			},
			errContains: "error getting provider",
		},
		{
			name: "fails when EntityCreator.CreateEntity fails",
			setupMocks: func(p *mockPropSvc.MockPropertiesService, s *mockdb.MockStore, c *mockCreator.MockEntityCreator) {
				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(parentEwp, nil)
				s.EXPECT().
					GetProviderByID(gomock.Any(), provID).
					Return(provider, nil)
				c.EXPECT().
					CreateEntity(gomock.Any(), &provider, projID, minderv1.Entity_ENTITY_PULL_REQUESTS, gomock.Any(), gomock.Any()).
					Return(nil, errors.New("entity registration failure"))
			},
			errContains: "error creating entity",
		},
		{
			name: "successfully adds child entity under parent originator",
			setupMocks: func(p *mockPropSvc.MockPropertiesService, s *mockdb.MockStore, c *mockCreator.MockEntityCreator) {
				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(parentEwp, nil)
				s.EXPECT().
					GetProviderByID(gomock.Any(), provID).
					Return(provider, nil)
				c.EXPECT().
					CreateEntity(gomock.Any(), &provider, projID, minderv1.Entity_ENTITY_PULL_REQUESTS, gomock.Any(), gomock.Any()).
					Return(childEwp, nil)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockP := mockPropSvc.NewMockPropertiesService(ctrl)
			mockM := mockProvMgr.NewMockProviderManager(ctrl)
			mockS := mockdb.NewMockStore(ctrl)
			mockC := mockCreator.NewMockEntityCreator(ctrl)

			tc.setupMocks(mockP, mockS, mockC)

			strat := strategiesEntity.NewAddOriginatingEntityStrategy(mockP, mockM, mockS, mockC)
			res, err := strat.GetEntity(context.Background(), baseMsg)

			if tc.errContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.Equal(t, childEwp, res)
			}
		})
	}
}
