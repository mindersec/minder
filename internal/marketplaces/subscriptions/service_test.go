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

package subscriptions_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/stacklok/minder/internal/db"
	dbf "github.com/stacklok/minder/internal/db/fixtures"
	brf "github.com/stacklok/minder/internal/marketplaces/bundles/mock/fixtures"
	"github.com/stacklok/minder/internal/marketplaces/subscriptions"
	"github.com/stacklok/minder/internal/profiles"
	psf "github.com/stacklok/minder/internal/profiles/mock/fixtures"
	"github.com/stacklok/minder/internal/ruletypes"
	rsf "github.com/stacklok/minder/internal/ruletypes/mock/fixtures"
	"github.com/stacklok/minder/pkg/mindpak/reader"
)

func TestSubscriptionService_Subscribe(t *testing.T) {
	t.Parallel()
	scenarios := []struct {
		Name          string
		DBSetup       dbf.DBMockBuilder
		BundleSetup   brf.BundleMockBuilder
		RuleTypeSetup rsf.RuleTypeSvcMockBuilder
		ExpectedError string
	}{
		{
			Name:        "Subscribe is a no-op when the subscription already exists",
			BundleSetup: brf.NewBundleReaderMock(brf.WithMetadata),
			DBSetup:     dbf.NewDBMock(withSuccessfulFindSubscription),
		},
		{
			Name:          "Subscribe returns error when it cannot query for existing subscriptions",
			BundleSetup:   brf.NewBundleReaderMock(brf.WithMetadata),
			DBSetup:       dbf.NewDBMock(withFailedFindSubscription),
			ExpectedError: "error while querying subscriptions",
		},
		{
			Name:          "Subscribe returns error when bundle cannot be upserted",
			BundleSetup:   brf.NewBundleReaderMock(brf.WithMetadata),
			DBSetup:       dbf.NewDBMock(withNotFoundFindSubscription, withFailedBundleUpsert),
			ExpectedError: "error while ensuring bundle exists",
		},
		{
			Name:          "Subscribe returns error when subscription cannot be created",
			BundleSetup:   brf.NewBundleReaderMock(brf.WithMetadata),
			DBSetup:       dbf.NewDBMock(withNotFoundFindSubscription, withFailedCreateSubscription, withBundleUpsert),
			ExpectedError: "error while creating subscription",
		},
		{
			Name:          "Subscribe returns error if rules cannot be read from bundle",
			DBSetup:       dbf.NewDBMock(withNotFoundFindSubscription, withBundleUpsert, withSuccessfulCreateSubscription),
			BundleSetup:   brf.NewBundleReaderMock(brf.WithMetadata, brf.WithFailedForEachRuleType),
			ExpectedError: "error while creating rules in project",
		},
		{
			Name:          "Subscribe returns error if rules cannot be upserted into database",
			DBSetup:       dbf.NewDBMock(withNotFoundFindSubscription, withBundleUpsert, withSuccessfulCreateSubscription),
			BundleSetup:   brf.NewBundleReaderMock(brf.WithMetadata, brf.WithSuccessfulForEachRuleType),
			RuleTypeSetup: rsf.NewRuleTypeServiceMock(rsf.WithFailedUpsertRuleType),
			ExpectedError: "error while creating rules in project",
		},
		{
			Name:          "Subscribe creates subscription",
			DBSetup:       dbf.NewDBMock(withNotFoundFindSubscription, withSuccessfulCreateSubscription, withBundleUpsert),
			BundleSetup:   brf.NewBundleReaderMock(brf.WithMetadata, brf.WithSuccessfulForEachRuleType),
			RuleTypeSetup: rsf.NewRuleTypeServiceMock(rsf.WithSuccessfulUpsertRuleType),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var bundle reader.BundleReader
			if scenario.BundleSetup != nil {
				bundle = scenario.BundleSetup(ctrl)
			}

			querier := getQuerier(ctrl, scenario.DBSetup)

			svc := createService(ctrl, nil, scenario.RuleTypeSetup)
			err := svc.Subscribe(ctx, projectID, bundle, querier)
			if scenario.ExpectedError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}

func TestSubscriptionService_CreateProfile(t *testing.T) {
	t.Parallel()
	scenarios := []struct {
		Name          string
		DBSetup       dbf.DBMockBuilder
		BundleSetup   brf.BundleMockBuilder
		ProfileSetup  psf.ProfileSvcMockBuilder
		ExpectedError string
	}{
		{
			Name:          "CreateProfile returns error when project is not subscribed to bundle",
			DBSetup:       dbf.NewDBMock(withNotFoundFindSubscription),
			BundleSetup:   brf.NewBundleReaderMock(brf.WithMetadata),
			ExpectedError: "not subscribed to bundle",
		},
		{
			Name:          "CreateProfile returns error when it cannot query for existing subscriptions",
			DBSetup:       dbf.NewDBMock(withFailedFindSubscription),
			BundleSetup:   brf.NewBundleReaderMock(brf.WithMetadata),
			ExpectedError: "error while querying subscriptions",
		},
		{
			Name:          "CreateProfile returns error if profile does not exist in bundle",
			DBSetup:       dbf.NewDBMock(withSuccessfulFindSubscription),
			BundleSetup:   brf.NewBundleReaderMock(brf.WithMetadata, brf.WithFailedGetProfile),
			ExpectedError: "error while retrieving profile from bundle",
		},
		{
			Name:          "CreateProfile returns error if profile cannot be created in project",
			DBSetup:       dbf.NewDBMock(withSuccessfulFindSubscription),
			BundleSetup:   brf.NewBundleReaderMock(brf.WithMetadata, brf.WithSuccessfulGetProfile),
			ProfileSetup:  psf.NewProfileServiceMock(psf.WithFailedCreateSubscriptionProfile),
			ExpectedError: "error while creating profile in project",
		},
		{
			Name:         "CreateProfile creates profile in project",
			DBSetup:      dbf.NewDBMock(withSuccessfulFindSubscription),
			BundleSetup:  brf.NewBundleReaderMock(brf.WithMetadata, brf.WithSuccessfulGetProfile),
			ProfileSetup: psf.NewProfileServiceMock(psf.WithSuccessfulCreateSubscriptionProfile),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			bundle := scenario.BundleSetup(ctrl)
			querier := getQuerier(ctrl, scenario.DBSetup)

			svc := createService(ctrl, scenario.ProfileSetup, nil)
			err := svc.CreateProfile(ctx, projectID, bundle, profileName, querier)
			if scenario.ExpectedError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, scenario.ExpectedError)
			}
		})
	}
}

const (
	profileName = "my_profile"
)

var (
	errDefault     = errors.New("error during subscription operation")
	subscriptionID = uuid.New()
	projectID      = uuid.New()
	bundleID       = uuid.New()
)

func withNotFoundFindSubscription(mock dbf.DBMock) {
	mock.EXPECT().
		GetSubscriptionByProjectBundle(gomock.Any(), gomock.Any()).
		Return(db.Subscription{}, sql.ErrNoRows)
}

func withFailedFindSubscription(mock dbf.DBMock) {
	mock.EXPECT().
		GetSubscriptionByProjectBundle(gomock.Any(), gomock.Any()).
		Return(db.Subscription{}, errDefault)
}

func withSuccessfulFindSubscription(mock dbf.DBMock) {
	mock.EXPECT().
		GetSubscriptionByProjectBundle(gomock.Any(), gomock.Any()).
		Return(db.Subscription{ID: subscriptionID}, nil)
}

func withSuccessfulCreateSubscription(mock dbf.DBMock) {
	mock.EXPECT().
		CreateSubscription(gomock.Any(), gomock.Any()).
		Return(db.Subscription{}, nil)
}

func withFailedCreateSubscription(mock dbf.DBMock) {
	mock.EXPECT().
		CreateSubscription(gomock.Any(), gomock.Any()).
		Return(db.Subscription{}, errDefault)
}

func withBundleUpsert(mock dbf.DBMock) {
	mock.EXPECT().
		UpsertBundle(gomock.Any(), gomock.Any()).
		Return(nil)

	mock.EXPECT().
		GetBundle(gomock.Any(), gomock.Any()).
		Return(db.Bundle{ID: bundleID}, nil)
}

func withFailedBundleUpsert(mock dbf.DBMock) {
	mock.EXPECT().
		UpsertBundle(gomock.Any(), gomock.Any()).
		Return(errDefault)
}

func createService(
	ctrl *gomock.Controller,
	profileSetup psf.ProfileSvcMockBuilder,
	ruleTypeSetup rsf.RuleTypeSvcMockBuilder,
) subscriptions.SubscriptionService {
	var rules ruletypes.RuleTypeService
	if ruleTypeSetup != nil {
		rules = ruleTypeSetup(ctrl)
	}

	var profSvc profiles.ProfileService
	if profileSetup != nil {
		profSvc = profileSetup(ctrl)
	}

	return subscriptions.NewSubscriptionService(profSvc, rules)
}

func getQuerier(ctrl *gomock.Controller, dbSetup dbf.DBMockBuilder) db.ExtendQuerier {
	var store db.Store
	if dbSetup != nil {
		store = dbSetup(ctrl)
	}

	return store
}
