// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package validators_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/entities/service/validators"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestRepositoryValidator_Validate(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()

	tests := []struct {
		name        string
		entityType  pb.Entity
		props       *properties.Properties
		setupMocks  func(*mockdb.MockStore)
		wantErr     bool
		errIs       error
		errContains string
	}{
		{
			name:       "allows valid public repository",
			entityType: pb.Entity_ENTITY_REPOSITORIES,
			props: properties.NewProperties(map[string]any{
				properties.RepoPropertyIsArchived: false,
				properties.RepoPropertyIsPrivate:  false,
			}),
			setupMocks: func(store *mockdb.MockStore) {
				// No feature flag check needed for public repos
			},
			wantErr: false,
		},
		// Note: Testing private repository feature flag logic is complex
		// as it involves multiple database calls. This is better tested
		// via integration tests.
		{
			name:       "rejects archived repository",
			entityType: pb.Entity_ENTITY_REPOSITORIES,
			props: properties.NewProperties(map[string]any{
				properties.RepoPropertyIsArchived: true,
				properties.RepoPropertyIsPrivate:  false,
			}),
			setupMocks: func(store *mockdb.MockStore) {},
			wantErr:    true,
			errIs:      validators.ErrArchivedRepoForbidden,
		},
		{
			name:       "skips validation for non-repository entities",
			entityType: pb.Entity_ENTITY_RELEASE,
			props: properties.NewProperties(map[string]any{
				"some_property": "value",
			}),
			setupMocks: func(store *mockdb.MockStore) {
				// No mocks needed - should return early
			},
			wantErr: false,
		},
		{
			name:       "skips validation for artifacts",
			entityType: pb.Entity_ENTITY_ARTIFACTS,
			props: properties.NewProperties(map[string]any{
				"name": "artifact",
			}),
			setupMocks: func(store *mockdb.MockStore) {},
			wantErr:    false,
		},
		{
			name:       "handles missing is_archived property gracefully",
			entityType: pb.Entity_ENTITY_REPOSITORIES,
			props: properties.NewProperties(map[string]any{
				properties.RepoPropertyIsPrivate: false,
				// is_archived missing
			}),
			setupMocks: func(store *mockdb.MockStore) {},
			wantErr:    true,
			errContains: "is_archived property",
		},
		{
			name:       "handles missing is_private property gracefully",
			entityType: pb.Entity_ENTITY_REPOSITORIES,
			props: properties.NewProperties(map[string]any{
				properties.RepoPropertyIsArchived: false,
				// is_private missing
			}),
			setupMocks: func(store *mockdb.MockStore) {},
			wantErr:    true,
			errContains: "is_private property",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tt.setupMocks(mockStore)

			validator := validators.NewRepositoryValidator(mockStore)

			err := validator.Validate(context.Background(), tt.entityType, tt.props, projectID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errIs != nil {
					assert.ErrorIs(t, err, tt.errIs)
				}
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
