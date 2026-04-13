// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	df "github.com/mindersec/minder/database/mock/fixtures"
	"github.com/mindersec/minder/internal/db"
)

func TestProcessInstallationRepositoriesAppEvent_BatchResilience(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	providerID := uuid.New()

	autoregEnabled := `{"github-app": {}, "auto_registration": {"entities": {"repository": {"enabled": true}}}}`

	validRepo := func(id int, name, fullName string) *repo {
		idVal := int64(id)
		return &repo{
			ID:       &idVal,
			Name:     &name,
			FullName: &fullName,
		}
	}

	invalidRepo := func() *repo {
		// name is empty, triggers repositoryAdded validation error
		emptyName := ""
		return &repo{
			Name: &emptyName,
		}
	}

	zeroIDRepo := func() *repo {
		// ID is 0, triggers repositoryRemoved validation error
		var zero int64
		name := "bad-repo"
		return &repo{
			ID:   &zero,
			Name: &name,
		}
	}

	mockInstallation := db.ProviderGithubAppInstallation{
		ProjectID:  uuid.NullUUID{UUID: projectID, Valid: true},
		ProviderID: uuid.NullUUID{UUID: providerID, Valid: true},
	}

	baseMocks := func(ctrl *gomock.Controller) db.Store {
		return df.NewMockStore(
			df.WithSuccessfulGetInstallationIDByAppID(mockInstallation, 54321),
			df.WithSuccessfulGetProviderByID(
				db.Provider{
					ID:         providerID,
					Definition: json.RawMessage(autoregEnabled),
				},
				providerID,
			),
		)(ctrl)
	}

	tests := []struct {
		name          string
		payload       *installationRepositoriesEvent
		expectedCount int
		expectErr     bool
	}{
		{
			name: "full batch success",
			payload: &installationRepositoriesEvent{
				Action:              strPtr("added"),
				RepositorySelection: strPtr("selected"),
				RepositoriesAdded: []*repo{
					validRepo(111, "repo-a", "org/repo-a"),
					validRepo(222, "repo-b", "org/repo-b"),
				},
				RepositoriesRemoved: []*repo{
					validRepo(333, "repo-c", "org/repo-c"),
				},
				Installation: &installation{ID: int64Ptr(54321)},
			},
			expectedCount: 3,
		},
		{
			name: "skip invalid added repo",
			payload: &installationRepositoriesEvent{
				Action:              strPtr("added"),
				RepositorySelection: strPtr("selected"),
				RepositoriesAdded: []*repo{
					validRepo(111, "repo-a", "org/repo-a"),
					invalidRepo(), // bad name → skipped
					validRepo(333, "repo-c", "org/repo-c"),
				},
				Installation: &installation{ID: int64Ptr(54321)},
			},
			expectedCount: 2,
		},
		{
			name: "skip invalid removed repo",
			payload: &installationRepositoriesEvent{
				Action:              strPtr("removed"),
				RepositorySelection: strPtr("selected"),
				RepositoriesRemoved: []*repo{
					validRepo(111, "repo-a", "org/repo-a"),
					zeroIDRepo(), // id=0 → skipped
					validRepo(333, "repo-c", "org/repo-c"),
				},
				Installation: &installation{ID: int64Ptr(54321)},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := baseMocks(ctrl)
			payload, err := json.Marshal(tt.payload)
			require.NoError(t, err)

			results, err := processInstallationRepositoriesAppEvent(
				context.Background(), store, payload,
			)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, results, tt.expectedCount)
		})
	}
}

func strPtr(s string) *string { return &s }
func int64Ptr(i int64) *int64 { return &i }
