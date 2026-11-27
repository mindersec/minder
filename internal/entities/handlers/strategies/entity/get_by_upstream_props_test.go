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

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/entities/handlers/message"
	strategiesEntity "github.com/mindersec/minder/internal/entities/handlers/strategies/entity"
	"github.com/mindersec/minder/internal/entities/models"
	propertyService "github.com/mindersec/minder/internal/entities/properties/service"
	mockPropSvc "github.com/mindersec/minder/internal/entities/properties/service/mock"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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

			if tc.errContains != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errContains)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.Equal(t, expectedEwp, res)
			}
		})
	}
}
