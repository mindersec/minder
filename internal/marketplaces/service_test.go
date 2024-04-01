// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package marketplaces_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/stacklok/minder/internal/db"
	dbf "github.com/stacklok/minder/internal/db/fixtures"
	"github.com/stacklok/minder/internal/marketplaces"
	mockbundle "github.com/stacklok/minder/internal/marketplaces/bundles/mock"
	bsf "github.com/stacklok/minder/internal/marketplaces/bundles/mock/fixtures"
	"github.com/stacklok/minder/internal/marketplaces/subscriptions"
	ssf "github.com/stacklok/minder/internal/marketplaces/subscriptions/mock/fixtures"
	"github.com/stacklok/minder/internal/marketplaces/types"
	"github.com/stacklok/minder/pkg/mindpak"
	"github.com/stacklok/minder/pkg/mindpak/sources"
)

// scenario structure is same for both tests
type testScenario struct {
	Name              string
	SourceSetup       bsf.SourceMockBuilder
	SubscriptionSetup ssf.SubscriptionMockBuilder
	ExpectedError     string
}

func TestMarketplace_Subscribe(t *testing.T) {
	t.Parallel()
	testHarness(t, subscribe, []testScenario{
		{
			Name:          "Subscribe returns error when bundle does not exist in source",
			SourceSetup:   bsf.NewBundleSourceMock(bsf.WithFailedGetBundle),
			ExpectedError: "error while retrieving bundle",
		},
		{
			Name:              "Subscribe returns error when subscription cannot be created",
			SourceSetup:       bsf.NewBundleSourceMock(bsf.WithSuccessfulGetBundle(bundleReader)),
			SubscriptionSetup: ssf.NewSubscriptionServiceMock(ssf.WithFailedSubscribe),
			ExpectedError:     "error while creating subscription",
		},
		{
			Name:              "Subscribe subscribes the project to the bundle",
			SourceSetup:       bsf.NewBundleSourceMock(bsf.WithSuccessfulGetBundle(bundleReader)),
			SubscriptionSetup: ssf.NewSubscriptionServiceMock(ssf.WithSuccessfulSubscribe),
		},
	})
}

func TestMarketplace_AddProfile(t *testing.T) {
	t.Parallel()
	testHarness(t, createProfile, []testScenario{
		{
			Name:          "AddProfile returns error when bundle does not exist in source",
			SourceSetup:   bsf.NewBundleSourceMock(bsf.WithFailedGetBundle),
			ExpectedError: "error while retrieving bundle",
		},
		{
			Name:              "AddProfile returns error when profile cannot be created",
			SourceSetup:       bsf.NewBundleSourceMock(bsf.WithSuccessfulGetBundle(bundleReader)),
			SubscriptionSetup: ssf.NewSubscriptionServiceMock(ssf.WithFailedCreateProfile),
			ExpectedError:     "error while creating profile in project",
		},
		{
			Name:              "AddProfile subscribes the project to the bundle",
			SourceSetup:       bsf.NewBundleSourceMock(bsf.WithSuccessfulGetBundle(bundleReader)),
			SubscriptionSetup: ssf.NewSubscriptionServiceMock(ssf.WithSuccessfulCreateProfile),
		},
	})
}

func testHarness(t *testing.T, method testMethod, scenarios []testScenario) {
	t.Helper()
	for i := range scenarios {
		scenario := scenarios[i]
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var subSvc subscriptions.SubscriptionService
			if scenario.SubscriptionSetup != nil {
				subSvc = scenario.SubscriptionSetup(ctrl)
			}

			var source sources.BundleSource
			if scenario.SourceSetup != nil {
				mock := scenario.SourceSetup(ctrl)
				bsf.WithListBundles(bundleID)(mock)
				source = mock
			}

			store := dbf.NewDBMock()(ctrl)

			marketplace, err := marketplaces.NewMarketplace([]sources.BundleSource{source}, subSvc)
			require.NoError(t, err)

			switch method {
			case subscribe:
				err = marketplace.Subscribe(ctx, projectContext, bundleID, store)
			case createProfile:
				err = marketplace.AddProfile(ctx, projectContext, bundleID, profileName, store)
			default:
				t.Fatalf("unknown method %d", method)
			}

			if scenario.ExpectedError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}

// use nil controller - we do not need to mock any methods on this type
var (
	bundleReader   = mockbundle.NewMockBundleReader(nil)
	projectContext = types.ProjectContext{
		ID:       uuid.New(),
		Provider: &db.Provider{},
	}
	bundleID = mindpak.ID("stacklok", "healthcheck")
)

const (
	profileName = "stacklok/a-profile"
)

type testMethod int

const (
	subscribe testMethod = iota
	createProfile
)
