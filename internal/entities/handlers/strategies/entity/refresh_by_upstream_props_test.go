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

func TestRefreshEntityByUpstreamPropsStrategy(t *testing.T) {
	t.Parallel()

	targetEwp := &models.EntityWithProperties{
		Entity: models.EntityInstance{
			ID:   uuid.New(),
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
			name: "fails when resolving upstream properties errors",
			msg: &message.HandleEntityAndDoMessage{
				Entity: message.TypedProps{
					Type:       minderv1.Entity_ENTITY_REPOSITORIES,
					GetByProps: map[string]any{"repo_id": 123},
				},
			},
			setupMocks: func(p *mockPropSvc.MockPropertiesService, _ *mockProvMgr.MockProviderManager, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, nil)
				s.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(s)
				s.EXPECT().Rollback(gomock.Any()).Return(nil)

				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("upstream lookup failed"))
			},
			errContains: "error getting entity",
		},
		{
			name: "fails when retrieving all properties errors",
			msg: &message.HandleEntityAndDoMessage{
				Entity: message.TypedProps{
					Type:       minderv1.Entity_ENTITY_REPOSITORIES,
					GetByProps: map[string]any{"repo_id": 123},
				},
			},
			setupMocks: func(p *mockPropSvc.MockPropertiesService, m *mockProvMgr.MockProviderManager, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, nil)
				s.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(s)
				s.EXPECT().Rollback(gomock.Any()).Return(nil)

				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(targetEwp, nil)
				p.EXPECT().
					RetrieveAllPropertiesForEntity(gomock.Any(), targetEwp, m, gomock.Any()).
					Return(errors.New("network failure"))
			},
			errContains: "error fetching entity",
		},
		{
			name: "successfully refreshes entity by upstream props",
			msg: &message.HandleEntityAndDoMessage{
				Entity: message.TypedProps{
					Type:       minderv1.Entity_ENTITY_REPOSITORIES,
					GetByProps: map[string]any{"repo_id": 123},
				},
			},
			setupMocks: func(p *mockPropSvc.MockPropertiesService, m *mockProvMgr.MockProviderManager, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, nil)
				s.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(s)
				s.EXPECT().Rollback(gomock.Any()).Return(nil)
				s.EXPECT().Commit(gomock.Any()).Return(nil)

				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any(), gomock.Any()).
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

			strat := strategiesEntity.NewRefreshEntityByUpstreamPropsStrategy(mockP, mockM, mockS)
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
