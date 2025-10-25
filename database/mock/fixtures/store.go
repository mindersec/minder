// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package fixtures contains code for creating RepositoryService
// fixtures and is used in various parts of the code. For testing use
// only.
//
//nolint:all
package fixtures

import (
	"database/sql"
	"encoding/json"
	"slices"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/db"
)

type (
	MockStoreBuilder = func(*gomock.Controller) *mockdb.MockStore
)

func NewMockStore(
	funcs ...func(*mockdb.MockStore),
) func(*gomock.Controller) *mockdb.MockStore {
	return func(ctrl *gomock.Controller) *mockdb.MockStore {
		mockStore := mockdb.NewMockStore(ctrl)

		for _, fn := range funcs {
			fn(mockStore)
		}

		return mockStore
	}
}

func WithSuccessfulGetProviderByID(
	provider db.Provider,
	providerID uuid.UUID,
) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			GetProviderByID(gomock.Any(), providerID).
			Return(provider, nil)
	}
}

func WithFailedGetProviderByID(
	err error,
) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			GetProviderByID(gomock.Any(), gomock.Any()).
			Return(db.Provider{}, err)
	}
}

func WithSuccessfulGetInstallationIDByAppID(
	provider db.ProviderGithubAppInstallation,
	installationID int64,
) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			GetInstallationIDByAppID(gomock.Any(), installationID).
			Return(provider, nil)
	}
}

func WithFailedGetInstallationIDByAppID(
	err error,
) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			GetInstallationIDByAppID(gomock.Any(), gomock.Any()).
			Return(db.ProviderGithubAppInstallation{}, err)
	}
}

func WithSuccessfulGetFeatureInProject(
	active bool,
) func(*mockdb.MockStore) {
	if active {
		return func(mockStore *mockdb.MockStore) {
			mockStore.EXPECT().
				GetFeatureInProject(gomock.Any(), gomock.Any()).
				Return(json.RawMessage{}, nil)
		}
	}
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			GetFeatureInProject(gomock.Any(), gomock.Any()).
			Return(nil, sql.ErrNoRows)
	}
}

func WithSuccessfulUpsertPullRequest(
	instance db.EntityInstance,
) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			CreateOrEnsureEntityByID(gomock.Any(), gomock.Any()).
			Return(instance, nil)
	}
}

type createOrEnsureEntityByIDParamsMatcher struct {
	params db.CreateOrEnsureEntityByIDParams
}

func (m createOrEnsureEntityByIDParamsMatcher) String() string {
	return "matches CreateOrEnsureEntityByIDParams"
}

func (m createOrEnsureEntityByIDParamsMatcher) Matches(x interface{}) bool {
	actual, ok := x.(db.CreateOrEnsureEntityByIDParams)
	if !ok {
		return false
	}

	// Note we don't compare the ID because it might be
	// dynamically generated
	return m.params.EntityType == actual.EntityType &&
		m.params.Name == actual.Name &&
		m.params.ProjectID == actual.ProjectID &&
		m.params.ProviderID == actual.ProviderID &&
		m.params.OriginatedFrom == actual.OriginatedFrom
}

func WithSuccessfulUpsertPullRequestWithParams(
	instance db.EntityInstance,
	entParams db.CreateOrEnsureEntityByIDParams,
) func(*mockdb.MockStore) {
	coeebipMatcher := createOrEnsureEntityByIDParamsMatcher{params: entParams}
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			CreateOrEnsureEntityByID(gomock.Any(), coeebipMatcher).
			Return(instance, nil)
	}
}

func WithSuccessfulUpsertArtifact(
	instance db.EntityInstance,
) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			CreateOrEnsureEntityByID(gomock.Any(), gomock.Any()).
			Return(instance, nil)
	}
}

func WithTransaction() func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			BeginTransaction().
			Return(nil, nil)
		mockStore.EXPECT().
			GetQuerierWithTransaction(gomock.Any()).
			Return(mockStore)
		mockStore.EXPECT().
			Commit(gomock.Any()).
			Return(nil)
		mockStore.EXPECT().
			Rollback(gomock.Any()).
			Return(nil)
	}
}

func WithRollbackTransaction() func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			BeginTransaction().
			Return(nil, nil)
		mockStore.EXPECT().
			GetQuerierWithTransaction(gomock.Any()).
			Return(mockStore)
		mockStore.EXPECT().
			Rollback(gomock.Any()).
			Return(nil)
	}
}

func WithSuccessfullGetEntityByID(
	expID uuid.UUID,
	entity db.EntityInstance,
) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			GetEntityByID(gomock.Any(), expID).
			Return(entity, nil)
	}
}

func WithSuccessfulGetEntitiesByProjectHierarchy(
	entities []db.EntityInstance,
	expectedProjectIDs []uuid.UUID,
) func(*mockdb.MockStore) {
	isSubset := func(actualAny any) bool {
		actual, ok := actualAny.([]uuid.UUID)
		if !ok {
			return false
		}

		for _, e := range expectedProjectIDs {
			if !slices.Contains(actual, e) {
				return false
			}
		}
		return true
	}

	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			GetEntitiesByProjectHierarchy(
				gomock.Any(),
				gomock.Cond(isSubset)).
			Return(entities, nil)
	}
}

func WithSuccessfulDeleteEntity(entID, projectID uuid.UUID) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			DeleteEntity(gomock.Any(), db.DeleteEntityParams{
				ID:        entID,
				ProjectID: projectID,
			}).
			Return(nil)
	}
}

func WithFailedGetEntitiesByProjectHierarchy(
	err error,
) func(*mockdb.MockStore) {
	return func(mockStore *mockdb.MockStore) {
		mockStore.EXPECT().
			GetEntitiesByProjectHierarchy(gomock.Any(), gomock.Any()).
			Return(nil, err)
	}
}
