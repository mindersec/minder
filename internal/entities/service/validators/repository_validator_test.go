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
	"github.com/mindersec/minder/pkg/entities/properties"
)

func TestRepositoryValidator_Validate(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()

	tests := []struct {
		name        string
		props       *properties.Properties
		setupMocks  func(*mockdb.MockStore)
		wantErr     bool
		errIs       error
		errContains string
	}{
		{
			name: "allows valid public repository",
			props: properties.NewProperties(map[string]any{
				properties.RepoPropertyIsArchived: false,
				properties.RepoPropertyIsPrivate:  false,
			}),
			// No feature flag check needed for public repos
			wantErr: false,
		},
		// Note: Testing private repository feature flag logic is complex
		// as it involves multiple database calls. This is better tested
		// via integration tests.
		{
			name: "rejects archived repository",
			props: properties.NewProperties(map[string]any{
				properties.RepoPropertyIsArchived: true,
				properties.RepoPropertyIsPrivate:  false,
			}),
			wantErr: true,
			errIs:   validators.ErrArchivedRepoForbidden,
		},
		{
			name: "handles missing is_archived property gracefully",
			props: properties.NewProperties(map[string]any{
				properties.RepoPropertyIsPrivate: false,
				// is_archived missing
			}),
			wantErr:     true,
			errContains: "is_archived property",
		},
		{
			name: "handles missing is_private property gracefully",
			props: properties.NewProperties(map[string]any{
				properties.RepoPropertyIsArchived: false,
				// is_private missing
			}),
			wantErr:     true,
			errContains: "is_private property",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			if tt.setupMocks != nil {
				tt.setupMocks(mockStore)
			}

			validator := validators.NewRepositoryValidator(mockStore)

			// Note: RepositoryValidator is now registered for ENTITY_REPOSITORIES
			// via the ValidatorRegistry, so entity type is not passed to Validate()
			err := validator.Validate(context.Background(), tt.props, projectID)

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
