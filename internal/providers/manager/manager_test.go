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
	"encoding/json"
	"errors"
	"fmt"
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

func TestProviderManager_CreateFromConfig(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name              string
		Provider          *db.Provider
		Config            json.RawMessage
		ExpectedError     string
		ValidateConfigErr bool
	}{
		{
			Name:          "CreateFromConfig returns error when provider class has no associated manager",
			Provider:      githubAppProvider,
			ExpectedError: "unexpected provider class",
		},
		{
			Name:     "CreateFromConfig creates a github provider with default configuration",
			Provider: providerWithClass(db.ProviderClassGithub),
			Config:   json.RawMessage(`{ github: {} }`),
		},
		{
			Name:     "CreateFromConfig creates a github provider with custom configuration",
			Provider: providerWithClass(db.ProviderClassGithub, providerWithConfig(json.RawMessage(`{ github: { key: value} }`))),
			Config:   json.RawMessage(`{ github: { key: value} }`),
		},
		{
			Name:              "CreateFromConfig returns an error when the config is invalid",
			Provider:          providerWithClass(db.ProviderClassGithub, providerWithConfig(json.RawMessage(`{ github: { key: value} }`))),
			Config:            json.RawMessage(`{ github: { key: value} }`),
			ExpectedError:     "invalid provider configuration",
			ValidateConfigErr: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			store := fixtures.NewProviderStoreMock()(ctrl)

			classManager := mockmanager.NewMockProviderClassManager(ctrl)
			classManager.EXPECT().GetSupportedClasses().Return([]db.ProviderClass{db.ProviderClassGithub}).MaxTimes(1)
			classManager.EXPECT().GetConfig(gomock.Any(), scenario.Provider.Class, gomock.Any()).Return(scenario.Config, nil).MaxTimes(1)
			if scenario.ValidateConfigErr {
				classManager.EXPECT().MarshallConfig(gomock.Any(), scenario.Provider.Class, scenario.Config).
					Return(nil, fmt.Errorf("invalid config")).MaxTimes(1)
			} else {
				classManager.EXPECT().MarshallConfig(gomock.Any(), scenario.Provider.Class, scenario.Config).
					Return(scenario.Config, nil).MaxTimes(1)
			}

			expectedProvider := providerWithClass(scenario.Provider.Class, providerWithConfig(scenario.Config))
			store.EXPECT().Create(gomock.Any(), scenario.Provider.Class, scenario.Provider.Name, scenario.Provider.ProjectID, scenario.Config).Return(expectedProvider, nil).MaxTimes(1)

			provManager, err := manager.NewProviderManager(store, classManager)
			require.NoError(t, err)

			newProv, err := provManager.CreateFromConfig(ctx, scenario.Provider.Class, scenario.Provider.ProjectID, scenario.Provider.Name, scenario.Config)
			if scenario.ExpectedError != "" {
				require.ErrorContains(t, err, scenario.ExpectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, newProv, expectedProvider)
			}
		})
	}
}

// Test both create by name/project, and create by ID together.
// This is because the test logic is basically identical.
func TestProviderManager_Instantiate(t *testing.T) {
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

func TestProviderManager_BulkInstantiateByTrait(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name               string
		Provider           *db.Provider
		ProviderStoreSetup fixtures.ProviderStoreMockBuilder
		InstantiationFails bool
		ExpectedError      string
	}{
		{
			Name:               "InstantiateFromID returns error when DB lookup fails",
			Provider:           githubProvider,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithFailedGetByTraitInHierarchy),
			ExpectedError:      "error retrieving db record",
		},
		{
			Name:               "BulkInstantiateByTrait returns name of provider which could not be instantiated",
			Provider:           githubAppProvider,
			InstantiationFails: true,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithSuccessfulGetByTraitInHierarchy(githubProvider)),
		},
		{
			Name:               "BulkInstantiateByTrait calls manager and returns provider",
			Provider:           githubProvider,
			InstantiationFails: false,
			ProviderStoreSetup: fixtures.NewProviderStoreMock(fixtures.WithSuccessfulGetByTraitInHierarchy(githubProvider)),
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
			if scenario.InstantiationFails {
				classManager.EXPECT().Build(gomock.Any(), gomock.Any()).Return(nil, errors.New("oh no"))
			} else {
				classManager.EXPECT().Build(gomock.Any(), gomock.Any()).Return(provider, nil).MaxTimes(1)
			}
			classManager.EXPECT().GetSupportedClasses().Return([]db.ProviderClass{db.ProviderClassGithub}).MaxTimes(1)
			provManager, err := manager.NewProviderManager(store, classManager)
			require.NoError(t, err)

			success, fail, err := provManager.BulkInstantiateByTrait(ctx, scenario.Provider.ProjectID, db.ProviderTypeRepoLister, "")
			if scenario.ExpectedError != "" {
				require.ErrorContains(t, err, scenario.ExpectedError)
			} else if scenario.InstantiationFails {
				require.Len(t, fail, 1)
				require.Empty(t, success)
				require.Equal(t, scenario.Provider.Name, fail[0])
			} else {
				require.Len(t, success, 1)
				require.Empty(t, fail)
				require.Equal(t, provider, success[scenario.Provider.Name])
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
		Name:       "test-provider",
		ID:         uuid.New(),
		ProjectID:  uuid.New(),
		Implements: []db.ProviderType{db.ProviderTypeRepoLister},
	}
	githubAppProvider = providerWithClass(db.ProviderClassGithubApp)
	githubProvider    = providerWithClass(db.ProviderClassGithub)
)

type createProviderOpt func(*db.Provider)

func providerWithConfig(config json.RawMessage) createProviderOpt {
	return func(p *db.Provider) {
		p.Definition = config
	}
}

func providerWithClass(class db.ProviderClass, opts ...createProviderOpt) *db.Provider {
	newProvider := referenceProvider
	newProvider.Class = class

	for _, opt := range opts {
		opt(&newProvider)
	}

	return &newProvider
}

type lookupType int

const (
	byID lookupType = iota
	byName
)
