// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package repositories_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/entities/models"
	mock_propservice "github.com/mindersec/minder/internal/entities/properties/service/mock"
	mock_entityservice "github.com/mindersec/minder/internal/entities/service/mock"
	"github.com/mindersec/minder/internal/entities/service/validators"
	"github.com/mindersec/minder/internal/repositories"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	mockevents "github.com/mindersec/minder/pkg/eventer/interfaces/mock"
)

// TestRepositoryService_CreateRepository_Integration tests that RepositoryService
// correctly delegates to EntityCreator
func TestRepositoryService_CreateRepository_Integration(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	providerID := uuid.New()
	entityID := uuid.New()

	testProvider := &db.Provider{
		ID:        providerID,
		Name:      "github",
		ProjectID: projectID,
	}

	fetchByProps := properties.NewProperties(map[string]any{
		"github/repo_owner": "test-owner",
		"github/repo_name":  "test-repo",
	})

	tests := []struct {
		name           string
		setupMocks     func(*mock_entityservice.MockEntityCreator, *mock_propservice.MockPropertiesService)
		wantErr        bool
		errIs          error
		validateResult func(*testing.T, *pb.Repository)
	}{
		{
			name: "successfully creates repository",
			setupMocks: func(creator *mock_entityservice.MockEntityCreator, propSvc *mock_propservice.MockPropertiesService) {
				// EntityCreator should be called with correct parameters
				ewp := &models.EntityWithProperties{
					Entity: models.EntityInstance{
						ID:         entityID,
						Type:       pb.Entity_ENTITY_REPOSITORIES,
						Name:       "test-owner/test-repo",
						ProjectID:  projectID,
						ProviderID: providerID,
					},
					Properties: fetchByProps,
				}

				creator.EXPECT().
					CreateEntity(gomock.Any(), testProvider, projectID, pb.Entity_ENTITY_REPOSITORIES, fetchByProps, gomock.Any()).
					Return(ewp, nil)

				// Should convert to protobuf
				idStr := entityID.String()
				propSvc.EXPECT().
					EntityWithPropertiesAsProto(gomock.Any(), ewp, gomock.Any()).
					Return(&pb.Repository{
						Id:        &idStr,
						Name:      "test-repo",
						Owner:     "test-owner",
						RepoId:    12345,
						IsPrivate: false,
					}, nil)
			},
			wantErr: false,
			validateResult: func(t *testing.T, repo *pb.Repository) {
				t.Helper()
				require.NotNil(t, repo)
				assert.NotNil(t, repo.Id)
				assert.Equal(t, "test-repo", repo.Name)
				assert.Equal(t, "test-owner", repo.Owner)
			},
		},
		{
			name: "returns archived error from EntityCreator",
			setupMocks: func(creator *mock_entityservice.MockEntityCreator, _ *mock_propservice.MockPropertiesService) {
				creator.EXPECT().
					CreateEntity(gomock.Any(), testProvider, projectID, pb.Entity_ENTITY_REPOSITORIES, fetchByProps, gomock.Any()).
					Return(nil, validators.ErrArchivedRepoForbidden)
			},
			wantErr: true,
			errIs:   repositories.ErrArchivedRepoForbidden,
		},
		{
			name: "returns private repo error from EntityCreator",
			setupMocks: func(creator *mock_entityservice.MockEntityCreator, _ *mock_propservice.MockPropertiesService) {
				creator.EXPECT().
					CreateEntity(gomock.Any(), testProvider, projectID, pb.Entity_ENTITY_REPOSITORIES, fetchByProps, gomock.Any()).
					Return(nil, validators.ErrPrivateRepoForbidden)
			},
			wantErr: true,
			errIs:   repositories.ErrPrivateRepoForbidden,
		},
		{
			name: "wraps generic errors from EntityCreator",
			setupMocks: func(creator *mock_entityservice.MockEntityCreator, _ *mock_propservice.MockPropertiesService) {
				creator.EXPECT().
					CreateEntity(gomock.Any(), testProvider, projectID, pb.Entity_ENTITY_REPOSITORIES, fetchByProps, gomock.Any()).
					Return(nil, errors.New("some internal error"))
			},
			wantErr: true,
		},
		{
			name: "fails when proto conversion fails",
			setupMocks: func(creator *mock_entityservice.MockEntityCreator, propSvc *mock_propservice.MockPropertiesService) {
				creator.EXPECT().
					CreateEntity(gomock.Any(), testProvider, projectID, pb.Entity_ENTITY_REPOSITORIES, fetchByProps, gomock.Any()).
					Return(&models.EntityWithProperties{
						Entity: models.EntityInstance{
							ID:         entityID,
							Type:       pb.Entity_ENTITY_REPOSITORIES,
							Name:       "test-owner/test-repo",
							ProjectID:  projectID,
							ProviderID: providerID,
						},
						Properties: fetchByProps,
					}, nil)

				propSvc.EXPECT().
					EntityWithPropertiesAsProto(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("proto conversion error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockEntityCreator := mock_entityservice.NewMockEntityCreator(ctrl)
			mockPropSvc := mock_propservice.NewMockPropertiesService(ctrl)
			mockEvents := mockevents.NewMockInterface(ctrl)

			// Events setup (not used in current implementation but required by constructor)
			mockEvents.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			tt.setupMocks(mockEntityCreator, mockPropSvc)

			svc := repositories.NewRepositoryService(
				nil, // store not used directly in CreateRepository anymore
				mockPropSvc,
				mockEvents,
				nil, // providerManager not used directly anymore
				mockEntityCreator,
			)

			repo, err := svc.CreateRepository(context.Background(), testProvider, projectID, fetchByProps)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errIs != nil {
					assert.ErrorIs(t, err, tt.errIs)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, repo)
				if tt.validateResult != nil {
					tt.validateResult(t, repo)
				}
			}
		})
	}
}
