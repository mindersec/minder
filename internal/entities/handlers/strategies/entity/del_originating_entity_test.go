// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity_test

import (
	"context"
	"database/sql"
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
	mockProvMgr "github.com/mindersec/minder/internal/providers/manager/mock"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestDelOriginatingEntityStrategy(t *testing.T) {
	t.Parallel()

	childID := uuid.New()
	projID := uuid.New()

	baseMsg := &message.HandleEntityAndDoMessage{
		Originator: message.TypedProps{
			Type: minderv1.Entity_ENTITY_REPOSITORIES,
		},
		Entity: message.TypedProps{
			Type: minderv1.Entity_ENTITY_PULL_REQUESTS,
		},
	}

	parentEwp := &models.EntityWithProperties{}
	childEwp := &models.EntityWithProperties{
		Entity: models.EntityInstance{
			ID:        childID,
			ProjectID: projID,
		},
	}

	testCases := []struct {
		name        string
		setupMocks  func(p *mockPropSvc.MockPropertiesService, s *mockdb.MockStore)
		errContains string
	}{
		{
			name: "fails when BeginTransaction errors",
			setupMocks: func(_ *mockPropSvc.MockPropertiesService, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, errors.New("db pool exhausted"))
			},
			errContains: "error starting transaction",
		},
		{
			name: "fails when GetQuerierWithTransaction returns nil",
			setupMocks: func(_ *mockPropSvc.MockPropertiesService, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, nil)
				s.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(nil)
				s.EXPECT().Rollback(gomock.Any()).Return(nil)
			},
			errContains: "error getting querier",
		},
		{
			name: "fails when getting parent entity errors",
			setupMocks: func(p *mockPropSvc.MockPropertiesService, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, nil)
				s.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(s)
				s.EXPECT().Rollback(gomock.Any()).Return(nil)

				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("parent lookup failed"))
			},
			errContains: "error getting parent entity",
		},
		{
			name: "fails when getting child entity errors",
			setupMocks: func(p *mockPropSvc.MockPropertiesService, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, nil)
				s.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(s)
				s.EXPECT().Rollback(gomock.Any()).Return(nil)

				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(parentEwp, nil)
				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_PULL_REQUESTS, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("child lookup failed"))
			},
			errContains: "error getting parent entity",
		},
		{
			name: "fails when DeleteEntity returns non-sql.ErrNoRows error",
			setupMocks: func(p *mockPropSvc.MockPropertiesService, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, nil)
				s.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(s)
				s.EXPECT().Rollback(gomock.Any()).Return(nil)

				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(parentEwp, nil)
				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_PULL_REQUESTS, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(childEwp, nil)

				s.EXPECT().
					DeleteEntity(gomock.Any(), db.DeleteEntityParams{
						ID:        childID,
						ProjectID: projID,
					}).
					Return(errors.New("foreign key violation"))
			},
			errContains: "foreign key violation",
		},
		{
			name: "handles sql.ErrNoRows idempotently and succeeds commit",
			setupMocks: func(p *mockPropSvc.MockPropertiesService, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, nil)
				s.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(s)
				s.EXPECT().Rollback(gomock.Any()).Return(nil)
				s.EXPECT().Commit(gomock.Any()).Return(nil)

				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(parentEwp, nil)
				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_PULL_REQUESTS, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(childEwp, nil)

				s.EXPECT().
					DeleteEntity(gomock.Any(), db.DeleteEntityParams{
						ID:        childID,
						ProjectID: projID,
					}).
					Return(sql.ErrNoRows)
			},
		},
		{
			name: "fails when transaction commit errors",
			setupMocks: func(p *mockPropSvc.MockPropertiesService, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, nil)
				s.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(s)
				s.EXPECT().Rollback(gomock.Any()).Return(nil)

				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(parentEwp, nil)
				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_PULL_REQUESTS, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(childEwp, nil)

				s.EXPECT().
					DeleteEntity(gomock.Any(), db.DeleteEntityParams{
						ID:        childID,
						ProjectID: projID,
					}).
					Return(nil)

				s.EXPECT().
					Commit(gomock.Any()).
					Return(errors.New("commit deadlock"))
			},
			errContains: "error committing transaction",
		},
		{
			name: "successfully deletes child entity and commits transaction",
			setupMocks: func(p *mockPropSvc.MockPropertiesService, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, nil)
				s.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(s)
				s.EXPECT().Rollback(gomock.Any()).Return(nil)
				s.EXPECT().Commit(gomock.Any()).Return(nil)

				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(parentEwp, nil)
				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_PULL_REQUESTS, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(childEwp, nil)

				s.EXPECT().
					DeleteEntity(gomock.Any(), db.DeleteEntityParams{
						ID:        childID,
						ProjectID: projID,
					}).
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

			tc.setupMocks(mockP, mockS)

			strat := strategiesEntity.NewDelOriginatingEntityStrategy(mockP, mockM, mockS)
			res, err := strat.GetEntity(context.Background(), baseMsg)

			if tc.errContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.Nil(t, res)
			}
		})
	}
}
