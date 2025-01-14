// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package subscriptions_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	datasourceservice "github.com/mindersec/minder/internal/datasources/service"
	dsf "github.com/mindersec/minder/internal/datasources/service/mock/fixtures"
	"github.com/mindersec/minder/internal/db"
	dbf "github.com/mindersec/minder/internal/db/fixtures"
	brf "github.com/mindersec/minder/internal/marketplaces/bundles/mock/fixtures"
	"github.com/mindersec/minder/internal/marketplaces/subscriptions"
	"github.com/mindersec/minder/pkg/mindpak/reader"
	"github.com/mindersec/minder/pkg/profiles"
	psf "github.com/mindersec/minder/pkg/profiles/mock/fixtures"
	"github.com/mindersec/minder/pkg/ruletypes"
	rsf "github.com/mindersec/minder/pkg/ruletypes/mock/fixtures"
)

func TestSubscriptionService_Subscribe(t *testing.T) {
	t.Parallel()
	scenarios := []struct {
		Name            string
		DBSetup         dbf.DBMockBuilder
		BundleSetup     brf.BundleMockBuilder
		RuleTypeSetup   rsf.RuleTypeSvcMockBuilder
		DataSourceSetup dsf.DataSourcesSvcMockBuilder
		ExpectedError   string
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
			Name:            "Subscribe returns error if rules cannot be read from bundle",
			DBSetup:         dbf.NewDBMock(withNotFoundFindSubscription, withBundleUpsert, withSuccessfulCreateSubscription),
			BundleSetup:     brf.NewBundleReaderMock(brf.WithMetadata, brf.WithFailedForEachRuleType, brf.WithSuccessfulForEachDataSource),
			DataSourceSetup: dsf.NewDataSourcesServiceMock(dsf.WithSuccessfulUpsertDataSource),
			ExpectedError:   "error while creating rules in project",
		},
		{
			Name:            "Subscribe returns error if rules cannot be upserted into database",
			DBSetup:         dbf.NewDBMock(withNotFoundFindSubscription, withBundleUpsert, withSuccessfulCreateSubscription),
			BundleSetup:     brf.NewBundleReaderMock(brf.WithMetadata, brf.WithSuccessfulForEachRuleType, brf.WithSuccessfulForEachDataSource),
			DataSourceSetup: dsf.NewDataSourcesServiceMock(dsf.WithSuccessfulUpsertDataSource),
			RuleTypeSetup:   rsf.NewRuleTypeServiceMock(rsf.WithFailedUpsertRuleType),
			ExpectedError:   "error while creating rules in project",
		},
		{
			Name:          "Subscribe returns error if data sources cannot be read from bundle",
			DBSetup:       dbf.NewDBMock(withNotFoundFindSubscription, withBundleUpsert, withSuccessfulCreateSubscription),
			BundleSetup:   brf.NewBundleReaderMock(brf.WithMetadata, brf.WithFailedForEachDataSource),
			ExpectedError: "error while creating data sources in project",
		},
		{
			Name:            "Subscribe creates subscription",
			DBSetup:         dbf.NewDBMock(withNotFoundFindSubscription, withSuccessfulCreateSubscription, withBundleUpsert),
			BundleSetup:     brf.NewBundleReaderMock(brf.WithMetadata, brf.WithSuccessfulForEachRuleType, brf.WithSuccessfulForEachDataSource),
			RuleTypeSetup:   rsf.NewRuleTypeServiceMock(rsf.WithSuccessfulUpsertRuleType),
			DataSourceSetup: dsf.NewDataSourcesServiceMock(dsf.WithSuccessfulUpsertDataSource),
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

			svc := createService(ctrl, nil, scenario.RuleTypeSetup, scenario.DataSourceSetup)
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

			svc := createService(ctrl, scenario.ProfileSetup, nil, nil)
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
	dataSourceSetup dsf.DataSourcesSvcMockBuilder,
) subscriptions.SubscriptionService {
	var rules ruletypes.RuleTypeService
	if ruleTypeSetup != nil {
		rules = ruleTypeSetup(ctrl)
	}

	var profSvc profiles.ProfileService
	if profileSetup != nil {
		profSvc = profileSetup(ctrl)
	}

	var dataSources datasourceservice.DataSourcesService
	if dataSourceSetup != nil {
		dataSources = dataSourceSetup(ctrl)
	}

	return subscriptions.NewSubscriptionService(profSvc, rules, dataSources)
}

func getQuerier(ctrl *gomock.Controller, dbSetup dbf.DBMockBuilder) db.ExtendQuerier {
	var store db.Store
	if dbSetup != nil {
		store = dbSetup(ctrl)
	}

	return store
}
