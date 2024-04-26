// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manager_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/stacklok/minder/internal/db"
	mockgithub "github.com/stacklok/minder/internal/providers/github/mock"
	"github.com/stacklok/minder/internal/providers/manager"
	mockmanager "github.com/stacklok/minder/internal/providers/manager/mock"
	"github.com/stacklok/minder/internal/providers/mock/fixtures"
)

// Test both create by name/project, and create by ID together.
// This is because the test logic is basically identical.
func TestProviderManager_Build(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name               string
		Provider           *db.Provider
		ProviderStoreSetup fixtures.ProviderStoreMockBuilder
		LookupType         lookupType
		ExpectedError      string
	}{
		{
			Name:               "InstantiateFromID returns error when DB lookup fails",
			Provider:           githubProvider,
			LookupType:         byID,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithFailedGetByID),
			ExpectedError:      "error retrieving db record",
		},
		{
			Name:               "InstantiateFromNameProject returns error when DB lookup fails",
			Provider:           githubProvider,
			LookupType:         byName,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithFailedGetByName),
			ExpectedError:      "error retrieving db record",
		},
		{
			Name:               "InstantiateFromID returns error when provider class has no associated manager",
			Provider:           githubAppProvider,
			LookupType:         byID,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithSuccessfulGetByID(githubAppProvider)),
			ExpectedError:      "unexpected provider class",
		},
		{
			Name:               "InstantiateFromNameProject returns error when provider class has no associated manager",
			Provider:           githubAppProvider,
			LookupType:         byName,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithSuccessfulGetByName(githubAppProvider)),
			ExpectedError:      "unexpected provider class",
		},
		{
			Name:               "InstantiateFromID calls manager and returns provider",
			Provider:           githubProvider,
			LookupType:         byID,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithSuccessfulGetByID(githubProvider)),
		},
		{
			Name:               "InstantiateFromNameProject calls manager and returns provider",
			Provider:           githubProvider,
			LookupType:         byName,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithSuccessfulGetByName(githubProvider)),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			store := scenario.ProviderStoreSetup(ctrl)
			classManager := mockmanager.NewMockProviderClassManager(ctrl)
			provider := mockgithub.NewMockGitHub(ctrl)
			classManager.EXPECT().Build(gomock.Any(), gomock.Any()).Return(provider, nil).MaxTimes(1)
			classManager.EXPECT().GetSupportedClasses().Return([]db.ProviderClass{db.ProviderClassGithub}).MaxTimes(1)
			provManager, err := manager.NewProviderManager(store, classManager)
			require.NoError(t, err)

			if scenario.LookupType == byName {
				_, err = provManager.InstantiateFromNameProject(ctx, scenario.Provider.Name, scenario.Provider.ProjectID)
			} else {
				_, err = provManager.InstantiateFromID(ctx, scenario.Provider.ID)
			}

			if scenario.ExpectedError != "" {
				require.ErrorContains(t, err, scenario.ExpectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test both delete by name/project, and create by ID together.
// This is because the test logic is basically identical.
func TestProviderManager_Delete(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name               string
		Provider           *db.Provider
		ProviderStoreSetup fixtures.ProviderStoreMockBuilder
		CleanupSucceeds    bool
		LookupType         lookupType
		ExpectedError      string
	}{
		{
			Name:               "DeleteByID returns error when DB lookup fails",
			Provider:           githubProvider,
			LookupType:         byID,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithFailedGetByIDProject),
			ExpectedError:      "error retrieving db record",
		},
		{
			Name:               "DeleteByName returns error when DB lookup fails",
			Provider:           githubProvider,
			LookupType:         byName,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithFailedGetByNameInSpecificProject),
			ExpectedError:      "error retrieving db record",
		},
		{
			Name:               "DeleteByID returns error when provider class has no associated manager",
			Provider:           githubAppProvider,
			LookupType:         byID,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithSuccessfulGetByIDProject(githubAppProvider)),
			ExpectedError:      "unexpected provider class",
		},
		{
			Name:               "DeleteByName returns error when provider class has no associated manager",
			Provider:           githubAppProvider,
			LookupType:         byName,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithSuccessfulGetByNameInSpecificProject(githubAppProvider)),
			ExpectedError:      "unexpected provider class",
		},
		{
			Name:               "DeleteByID returns error when provider-specific cleanup fails",
			Provider:           githubProvider,
			LookupType:         byID,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithSuccessfulGetByIDProject(githubProvider)),
			CleanupSucceeds:    false,
			ExpectedError:      "error while cleaning up provider",
		},
		{
			Name:               "DeleteByName returns error when provider-specific cleanup fails",
			Provider:           githubProvider,
			LookupType:         byName,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithSuccessfulGetByNameInSpecificProject(githubProvider)),
			CleanupSucceeds:    false,
			ExpectedError:      "error while cleaning up provider",
		},
		{
			Name:            "DeleteByID returns error when provider cannot be deleted from the database",
			Provider:        githubProvider,
			LookupType:      byID,
			CleanupSucceeds: true,
			ExpectedError:   "error while deleting provider from DB",
			ProviderStoreSetup: fixtures.NewProviderStoreMock(
				fixtures.WithSuccessfulGetByIDProject(githubProvider),
				fixtures.WithFailedDelete,
			),
		},
		{
			Name:            "DeleteByName returns error when provider cannot be deleted from the database",
			Provider:        githubProvider,
			LookupType:      byName,
			CleanupSucceeds: true,
			ExpectedError:   "error while deleting provider from DB",
			ProviderStoreSetup: fixtures.NewProviderStoreMock(
				fixtures.WithSuccessfulGetByNameInSpecificProject(githubProvider),
				fixtures.WithFailedDelete,
			),
		},
		{
			Name:            "DeleteByID calls manager and returns provider",
			Provider:        githubProvider,
			LookupType:      byID,
			CleanupSucceeds: true,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(
				fixtures.WithSuccessfulGetByIDProject(githubProvider),
				fixtures.WithSuccessfulDelete(githubProvider),
			),
		},
		{
			Name:            "DeleteByName calls manager and returns provider",
			Provider:        githubProvider,
			LookupType:      byName,
			CleanupSucceeds: true,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(
				fixtures.WithSuccessfulGetByNameInSpecificProject(githubProvider),
				fixtures.WithSuccessfulDelete(githubProvider),
			),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			store := scenario.ProviderStoreSetup(ctrl)
			classManager := mockmanager.NewMockProviderClassManager(ctrl)
			classManager.EXPECT().GetSupportedClasses().Return([]db.ProviderClass{db.ProviderClassGithub}).MaxTimes(1)
			if scenario.CleanupSucceeds {
				classManager.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil)
			} else {
				classManager.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(errors.New("oh no")).MaxTimes(1)
			}
			provManager, err := manager.NewProviderManager(store, classManager)
			require.NoError(t, err)

			if scenario.LookupType == byName {
				err = provManager.DeleteByName(ctx, scenario.Provider.Name, scenario.Provider.ProjectID)
			} else {
				err = provManager.DeleteByID(ctx, scenario.Provider.ID, scenario.Provider.ProjectID)
			}

			if scenario.ExpectedError != "" {
				require.ErrorContains(t, err, scenario.ExpectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

var (
	referenceProvider = db.Provider{
		Name:      "test-provider",
		ID:        uuid.New(),
		ProjectID: uuid.New(),
	}
	githubAppProvider = providerWithClass(db.ProviderClassGithubApp)
	githubProvider    = providerWithClass(db.ProviderClassGithub)
)

func providerWithClass(class db.ProviderClass) *db.Provider {
	newProvider := referenceProvider
	newProvider.Class = class
	return &newProvider
}

type lookupType int

const (
	byID lookupType = iota
	byName
)
