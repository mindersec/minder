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
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/manager"
	"github.com/stacklok/minder/internal/providers/mock/fixtures"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

// Test both create by name/project, and create by ID together.
// This is because the test logic is basically identical.
func TestProviderFactory(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name              string
		Provider          db.Provider
		LookupType        lookupType
		RetrievalSucceeds bool
		ExpectedError     string
	}{
		{
			Name:              "InstantiateFromID returns error when DB lookup fails",
			LookupType:        byID,
			RetrievalSucceeds: false,
			ExpectedError:     "error retrieving db record",
		},
		{
			Name:              "InstantiateFromNameProject returns error when DB lookup fails",
			LookupType:        byName,
			RetrievalSucceeds: false,
			ExpectedError:     "error retrieving db record",
		},
		{
			Name:              "InstantiateFromID returns error when provider class has no associated manager",
			Provider:          providerWithClass(db.ProviderClassGithubApp),
			LookupType:        byID,
			RetrievalSucceeds: true,
			ExpectedError:     "unexpected provider class",
		},
		{
			Name:              "InstantiateFromNameProject returns error when provider class has no associated manager",
			Provider:          providerWithClass(db.ProviderClassGithubApp),
			LookupType:        byName,
			RetrievalSucceeds: true,
			ExpectedError:     "unexpected provider class",
		},
		{
			Name:              "InstantiateFromID calls manager and returns provider",
			Provider:          providerWithClass(db.ProviderClassGithub),
			LookupType:        byID,
			RetrievalSucceeds: true,
		},
		{
			Name:              "InstantiateFromNameProject calls manager and returns provider",
			Provider:          providerWithClass(db.ProviderClassGithub),
			LookupType:        byName,
			RetrievalSucceeds: true,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var opt func(mock fixtures.ProviderStoreMock)
			if scenario.LookupType == byName && scenario.RetrievalSucceeds {
				opt = fixtures.WithSuccessfulGetByName(&scenario.Provider)
			} else if scenario.LookupType == byID && scenario.RetrievalSucceeds {
				opt = fixtures.WithSuccessfulGetByID(&scenario.Provider)
			} else if scenario.LookupType == byID && !scenario.RetrievalSucceeds {
				opt = fixtures.WithFailedGetByID
			} else {
				opt = fixtures.WithFailedGetByName
			}

			store := fixtures.NewProviderStoreMock(opt)(ctrl)
			provFactory, err := manager.NewProviderManager(
				[]manager.ProviderClassManager{&mockClassManager{}},
				store,
			)
			require.NoError(t, err)

			if scenario.LookupType == byName {
				_, err = provFactory.InstantiateFromNameProject(ctx, scenario.Provider.Name, scenario.Provider.ProjectID)
			} else {
				_, err = provFactory.InstantiateFromID(ctx, scenario.Provider.ID)
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
)

func providerWithClass(class db.ProviderClass) db.Provider {
	newProvider := referenceProvider
	newProvider.Class = class
	return newProvider
}

type lookupType int

const (
	byID lookupType = iota
	byName
)

// Not using the mock generator because we probably won't need to stub this
// elsewhere, and the implementation is trivial.
type mockClassManager struct{}

func (_ *mockClassManager) Build(_ context.Context, _ *db.Provider) (v1.Provider, error) {
	return &mockProvider{}, nil
}

func (_ *mockClassManager) Delete(_ context.Context, _ *db.Provider) error {
	return nil
}

func (_ *mockClassManager) GetSupportedClasses() []db.ProviderClass {
	return []db.ProviderClass{db.ProviderClassGithub}
}

// TODO: we probably want to mock some of the provider traits in future using
// the mock generator.
type mockProvider struct{}

func (_ *mockProvider) CanImplement(_ minderv1.ProviderType) bool {
	return false
}
