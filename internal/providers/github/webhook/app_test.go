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
	"github.com/mindersec/minder/internal/util/ptr"
)

func TestProcessInstallationRepositoriesAppEvent_BatchResilience(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	providerID := uuid.New()

	autoregEnabled := `{"github-app": {}, "auto_registration": {"entities": {"repository": {"enabled": true}}}}`

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
				Action:              ptr.Ptr("added"),
				RepositorySelection: ptr.Ptr("selected"),
				RepositoriesAdded: []*repo{
					newValidRepo(111, "repo-a", "org/repo-a"),
					newValidRepo(222, "repo-b", "org/repo-b"),
				},
				RepositoriesRemoved: []*repo{
					newValidRepo(333, "repo-c", "org/repo-c"),
				},
				Installation: &installation{ID: ptr.Ptr(int64(54321))},
			},
			expectedCount: 3,
		},
		{
			name: "skip invalid added repo",
			payload: &installationRepositoriesEvent{
				Action:              ptr.Ptr("added"),
				RepositorySelection: ptr.Ptr("selected"),
				RepositoriesAdded: []*repo{
					newValidRepo(111, "repo-a", "org/repo-a"),
					newInvalidRepo(), // empty name → skipped
					newValidRepo(333, "repo-c", "org/repo-c"),
				},
				Installation: &installation{ID: ptr.Ptr(int64(54321))},
			},
			expectedCount: 2,
		},
		{
			name: "skip invalid removed repo",
			payload: &installationRepositoriesEvent{
				Action:              ptr.Ptr("removed"),
				RepositorySelection: ptr.Ptr("selected"),
				RepositoriesRemoved: []*repo{
					newValidRepo(111, "repo-a", "org/repo-a"),
					newZeroIDRepo(), // id=0 → skipped
					newValidRepo(333, "repo-c", "org/repo-c"),
				},
				Installation: &installation{ID: ptr.Ptr(int64(54321))},
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

// newValidRepo constructs a repo with all required fields set.
func newValidRepo(id int64, name, fullName string) *repo {
	return &repo{
		ID:       ptr.Ptr(id),
		Name:     ptr.Ptr(name),
		FullName: ptr.Ptr(fullName),
	}
}

// newInvalidRepo constructs a repo with an empty name, which triggers
// a repositoryAdded validation error and causes the entry to be skipped.
func newInvalidRepo() *repo {
	return &repo{
		Name: ptr.Ptr(""),
	}
}

// newZeroIDRepo constructs a repo with ID=0, which triggers a
// repositoryRemoved validation error and causes the entry to be skipped.
func newZeroIDRepo() *repo {
	return &repo{
		ID:   ptr.Ptr(int64(0)),
		Name: ptr.Ptr("bad-repo"),
	}
}
