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
	"github.com/mindersec/minder/internal/entities/handlers/message"
	strategiesEntity "github.com/mindersec/minder/internal/entities/handlers/strategies/entity"
	"github.com/mindersec/minder/internal/entities/models"
	mockPropSvc "github.com/mindersec/minder/internal/entities/properties/service/mock"
	mockProvMgr "github.com/mindersec/minder/internal/providers/manager/mock"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestRefreshEntityByIDStrategy(t *testing.T) {
	t.Parallel()

	targetID := uuid.New()
	targetEwp := &models.EntityWithProperties{
		Entity: models.EntityInstance{
			ID:   targetID,
			Type: minderv1.Entity_ENTITY_REPOSITORIES,
		},
	}

	testCases := []struct {
		name        string
		msg         *message.HandleEntityAndDoMessage
		setupMocks  func(p *mockPropSvc.MockPropertiesService, m *mockProvMgr.MockProviderManager, s *mockdb.MockStore)
		errContains string
	}{
		{
			name: "returns error when entity id is nil",
			msg: &message.HandleEntityAndDoMessage{
				Entity: message.TypedProps{
					EntityID: uuid.Nil,
				},
			},
			setupMocks:  func(_ *mockPropSvc.MockPropertiesService, _ *mockProvMgr.MockProviderManager, _ *mockdb.MockStore) {},
			errContains: "entity id is nil",
		},
		{
			name: "fails when entity lookup by ID errors and triggers rollback",
			msg: &message.HandleEntityAndDoMessage{
				Entity: message.TypedProps{
					EntityID: targetID,
				},
			},
			setupMocks: func(p *mockPropSvc.MockPropertiesService, _ *mockProvMgr.MockProviderManager, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, nil)
				s.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(s)
				s.EXPECT().Rollback(gomock.Any()).Return(nil)

				p.EXPECT().
					EntityWithPropertiesByID(gomock.Any(), targetID, gomock.Any()).
					Return(nil, errors.New("db not found"))
			},
			errContains: "error getting entity",
		},
		{
			name: "fails when retrieving all properties errors",
			msg: &message.HandleEntityAndDoMessage{
				Entity: message.TypedProps{
					EntityID: targetID,
				},
			},
			setupMocks: func(p *mockPropSvc.MockPropertiesService, m *mockProvMgr.MockProviderManager, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, nil)
				s.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(s)
				s.EXPECT().Rollback(gomock.Any()).Return(nil)

				p.EXPECT().
					EntityWithPropertiesByID(gomock.Any(), targetID, gomock.Any()).
					Return(targetEwp, nil)
				p.EXPECT().
					RetrieveAllPropertiesForEntity(gomock.Any(), targetEwp, m, gomock.Any()).
					Return(errors.New("fetch failed"))
			},
			errContains: "error retrieving properties for entity",
		},
		{
			name: "successfully refreshes entity properties by id",
			msg: &message.HandleEntityAndDoMessage{
				Entity: message.TypedProps{
					EntityID: targetID,
				},
			},
			setupMocks: func(p *mockPropSvc.MockPropertiesService, m *mockProvMgr.MockProviderManager, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, nil)
				s.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(s)
				s.EXPECT().Rollback(gomock.Any()).Return(nil)
				s.EXPECT().Commit(gomock.Any()).Return(nil)

				p.EXPECT().
					EntityWithPropertiesByID(gomock.Any(), targetID, gomock.Any()).
					Return(targetEwp, nil)
				p.EXPECT().
					RetrieveAllPropertiesForEntity(gomock.Any(), targetEwp, m, gomock.Any()).
					Return(nil)
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

			tc.setupMocks(mockP, mockM, mockS)

			strat := strategiesEntity.NewRefreshEntityByIDStrategy(mockP, mockM, mockS)
			res, err := strat.GetEntity(context.Background(), tc.msg)

			if tc.errContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.Equal(t, targetEwp, res)
			}
		})
	}
}
