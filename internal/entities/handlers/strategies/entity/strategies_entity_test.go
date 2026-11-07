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
	propertyService "github.com/mindersec/minder/internal/entities/properties/service"
	mockPropSvc "github.com/mindersec/minder/internal/entities/properties/service/mock"
	mockCreator "github.com/mindersec/minder/internal/entities/service/mock"
	mockProvMgr "github.com/mindersec/minder/internal/providers/manager/mock"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestGetEntityByUpstreamIDStrategy(t *testing.T) {
	t.Parallel()

	entID := uuid.New()
	projID := uuid.New()
	provID := uuid.New()

	expectedEwp := &models.EntityWithProperties{
		Entity: models.EntityInstance{
			ID:         entID,
			Type:       minderv1.Entity_ENTITY_REPOSITORIES,
			Name:       "owner/repo",
			ProjectID:  projID,
			ProviderID: provID,
		},
	}

	testCases := []struct {
		name        string
		msg         *message.HandleEntityAndDoMessage
		setupMock   func(p *mockPropSvc.MockPropertiesService)
		expectedRes *models.EntityWithProperties
		expectErr   bool
		errContains string
	}{
		{
			name: "successfully retrieves entity with provider implements hint",
			msg: &message.HandleEntityAndDoMessage{
				Entity: message.TypedProps{
					Type:       minderv1.Entity_ENTITY_REPOSITORIES,
					GetByProps: map[string]any{"repo_id": 12345},
				},
				Hint: message.EntityHint{
					ProviderImplementsHint: "github",
				},
			},
			setupMock: func(p *mockPropSvc.MockPropertiesService) {
				expectedHint := propertyService.ByUpstreamHint{
					ProviderImplements: db.NullProviderType{
						ProviderType: db.ProviderTypeGithub,
						Valid:        true,
					},
				}
				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), expectedHint, gomock.Any()).
					Return(expectedEwp, nil)
			},
			expectedRes: expectedEwp,
			expectErr:   false,
		},
		{
			name: "successfully retrieves entity with provider class hint",
			msg: &message.HandleEntityAndDoMessage{
				Entity: message.TypedProps{
					Type:       minderv1.Entity_ENTITY_REPOSITORIES,
					GetByProps: map[string]any{"repo_id": 12345},
				},
				Hint: message.EntityHint{
					ProviderClassHint: "github-app",
				},
			},
			setupMock: func(p *mockPropSvc.MockPropertiesService) {
				expectedHint := propertyService.ByUpstreamHint{
					ProviderClass: db.NullProviderClass{
						ProviderClass: db.ProviderClassGithubApp,
						Valid:         true,
					},
				}
				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), expectedHint, gomock.Any()).
					Return(expectedEwp, nil)
			},
			expectedRes: expectedEwp,
			expectErr:   false,
		},
		{
			name: "propagates error when properties service fails",
			msg: &message.HandleEntityAndDoMessage{
				Entity: message.TypedProps{
					Type:       minderv1.Entity_ENTITY_REPOSITORIES,
					GetByProps: map[string]any{"repo_id": 99999},
				},
			},
			setupMock: func(p *mockPropSvc.MockPropertiesService) {
				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("lookup timeout"))
			},
			expectedRes: nil,
			expectErr:   true,
			errContains: "error searching entity by ID",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			mockP := mockPropSvc.NewMockPropertiesService(ctrl)
			tc.setupMock(mockP)

			strat := strategiesEntity.NewGetEntityByUpstreamIDStrategy(mockP)
			res, err := strat.GetEntity(context.Background(), tc.msg)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedRes, res)
			}
		})
	}
}

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
		expectedRes *models.EntityWithProperties
		expectErr   bool
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
			expectedRes: nil,
			expectErr:   true,
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
			expectedRes: nil,
			expectErr:   true,
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
			expectedRes: nil,
			expectErr:   true,
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
			expectedRes: targetEwp,
			expectErr:   false,
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

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedRes, res)
			}
		})
	}
}

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
		expectedRes *models.EntityWithProperties
		expectErr   bool
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
			expectedRes: nil,
			expectErr:   true,
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
			expectedRes: nil,
			expectErr:   true,
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
			expectedRes: targetEwp,
			expectErr:   false,
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

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedRes, res)
			}
		})
	}
}

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
		expectedRes *models.EntityWithProperties
		expectErr   bool
		errContains string
	}{
		{
			name: "fails when resolving parent entity errors",
			setupMocks: func(p *mockPropSvc.MockPropertiesService, _ *mockdb.MockStore, _ *mockCreator.MockEntityCreator) {
				p.EXPECT().
					EntityWithPropertiesByUpstreamHint(gomock.Any(), minderv1.Entity_ENTITY_REPOSITORIES, gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("parent entity lookup failed"))
			},
			expectedRes: nil,
			expectErr:   true,
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
			expectedRes: nil,
			expectErr:   true,
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
			expectedRes: nil,
			expectErr:   true,
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
			expectedRes: childEwp,
			expectErr:   false,
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

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedRes, res)
			}
		})
	}
}

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
		expectErr   bool
		errContains string
		errIs       error
	}{
		{
			name: "fails when BeginTransaction errors",
			setupMocks: func(_ *mockPropSvc.MockPropertiesService, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, errors.New("db pool exhausted"))
			},
			expectErr:   true,
			errContains: "error starting transaction",
		},
		{
			name: "fails when GetQuerierWithTransaction returns nil",
			setupMocks: func(_ *mockPropSvc.MockPropertiesService, s *mockdb.MockStore) {
				s.EXPECT().BeginTransaction().Return(nil, nil)
				s.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(nil)
				s.EXPECT().Rollback(gomock.Any()).Return(nil)
			},
			expectErr:   true,
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
			expectErr:   true,
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
			expectErr:   true,
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
			expectErr:   true,
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
			expectErr: false,
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
			expectErr:   true,
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
			expectErr: false,
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

			if tc.expectErr {
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
