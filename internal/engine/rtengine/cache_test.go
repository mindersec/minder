// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rtengine

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdssvc "github.com/mindersec/minder/internal/datasources/service/mock"
	"github.com/mindersec/minder/internal/db"
	dbf "github.com/mindersec/minder/internal/db/fixtures"
	"github.com/mindersec/minder/internal/engine/ingestcache"
	"github.com/mindersec/minder/internal/providers/testproviders"
	v1datasources "github.com/mindersec/minder/pkg/datasources/v1"
	rtengine2 "github.com/mindersec/minder/pkg/engine/v1/rtengine"
)

func TestNewRuleTypeEngineCacheConstructor(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name           string
		DBSetup        dbf.DBMockBuilder
		DSServiceSetup func(service *mockdssvc.MockDataSourcesService)
		ExpectedError  string
	}{
		{
			Name: "Returns error when getting parent projects fails",
			DBSetup: dbf.NewDBMock(func(mock dbf.DBMock) {
				mock.EXPECT().GetParentProjects(gomock.Any(), gomock.Any()).Return(nil, errTest)
			}),
			ExpectedError: "error getting parent projects",
		},
		{
			Name: "Returns error when getting rule types fails",
			DBSetup: dbf.NewDBMock(func(mock dbf.DBMock) {
				mock.EXPECT().GetParentProjects(gomock.Any(), gomock.Any()).
					Return([]uuid.UUID{uuid.New()}, nil)
				mock.EXPECT().GetRuleTypesByEntityInHierarchy(gomock.Any(), gomock.Any()).
					Return(nil, errTest)
			}),
			ExpectedError: "error while retrieving rule types",
		},
		{
			Name: "Returns error when getting rule type with no def",
			DBSetup: dbf.NewDBMock(func(mock dbf.DBMock) {
				mock.EXPECT().GetParentProjects(gomock.Any(), gomock.Any()).
					Return([]uuid.UUID{uuid.New()}, nil)
				mock.EXPECT().GetRuleTypesByEntityInHierarchy(gomock.Any(), gomock.Any()).
					Return([]db.RuleType{{ID: uuid.New()}}, nil)
			}),
			ExpectedError: "cannot unmarshal rule type definition",
		},
		{
			Name: "Returns error when building data source registry fails",
			DBSetup: dbf.NewDBMock(func(mock dbf.DBMock) {
				hierarchy := []uuid.UUID{uuid.New(), uuid.New()}
				// Calls from the engine builder itself
				mock.EXPECT().GetParentProjects(gomock.Any(), gomock.Any()).
					Return(hierarchy, nil)
				mock.EXPECT().GetRuleTypesByEntityInHierarchy(gomock.Any(), gomock.Any()).
					Return([]db.RuleType{{
						ID:         uuid.New(),
						ProjectID:  hierarchy[0],
						Definition: []byte(ruleDefJSON),
					}}, nil)
			}),
			DSServiceSetup: func(service *mockdssvc.MockDataSourcesService) {
				service.EXPECT().BuildDataSourceRegistry(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errTest)
			},
			ExpectedError: errTest.Error(),
		},
		{
			Name: "Creates rule engine cache",
			DBSetup: dbf.NewDBMock(func(mock dbf.DBMock) {
				hierarchy := []uuid.UUID{uuid.New(), uuid.New()}
				// Calls from the engine builder itself
				mock.EXPECT().GetParentProjects(gomock.Any(), gomock.Any()).
					Return(hierarchy, nil)
				mock.EXPECT().GetRuleTypesByEntityInHierarchy(gomock.Any(), gomock.Any()).
					Return([]db.RuleType{{
						ID:         uuid.New(),
						ProjectID:  hierarchy[0],
						Definition: []byte(ruleDefJSON),
					}}, nil)
			}),
			DSServiceSetup: func(service *mockdssvc.MockDataSourcesService) {
				service.EXPECT().BuildDataSourceRegistry(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(v1datasources.NewDataSourceRegistry(), nil)
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			dssvc := mockdssvc.NewMockDataSourcesService(ctrl)

			var store db.Store
			if scenario.DBSetup != nil {
				store = scenario.DBSetup(ctrl)
			}

			if scenario.DSServiceSetup != nil {
				scenario.DSServiceSetup(dssvc)
			}

			cache, err := NewRuleEngineCache(
				ctx, store, db.EntitiesRepository, uuid.New(),
				testproviders.NewGitProvider(nil), ingestcache.NewNoopCache(),
				dssvc)
			if scenario.ExpectedError != "" {
				require.ErrorContains(t, err, scenario.ExpectedError)
				require.Nil(t, cache)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cache)

				// Ensure members are not null so we don't fall on the same issue
				// we had of not initializing them.
				impl, ok := cache.(*ruleEngineCache)
				require.True(t, ok)
				require.NotNil(t, impl.store)
				require.NotNil(t, impl.provider)
				require.NotNil(t, impl.ingestCache)
				require.NotNil(t, impl.engines)
				require.NotNil(t, impl.dssvc)
			}
		})
	}
}

func TestGetRuleEngine(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name            string
		Cache           cacheType
		DBSetup         dbf.DBMockBuilder
		ExpectedError   string
		dsRegistryError error
	}{
		{
			Name:  "Retrieves rule engine from cache",
			Cache: cacheType{ruleTypeID: &rtengine2.RuleTypeEngine{}},
		},
		{
			Name:          "Returns error when rule type does not exist",
			Cache:         cacheType{},
			DBSetup:       dbf.NewDBMock(withRuleTypeLookup(nil, sql.ErrNoRows)),
			ExpectedError: "unknown rule type with ID",
		},
		{
			Name:          "Returns error when rule type lookup fails",
			Cache:         cacheType{},
			DBSetup:       dbf.NewDBMock(withRuleTypeLookup(nil, errTest)),
			ExpectedError: "error creating rule type engine",
		},
		{
			Name:          "Returns error when rule type cannot be parsed",
			Cache:         cacheType{},
			DBSetup:       dbf.NewDBMock(withRuleTypeLookup(&db.RuleType{}, nil)),
			ExpectedError: "error parsing rule type when parsing rule type",
		},
		{
			Name:          "Returns error when rule type engine cannot be instantiated",
			Cache:         cacheType{},
			DBSetup:       dbf.NewDBMock(withRuleTypeLookup(&malformedRuleType, nil)),
			ExpectedError: "error creating rule type engine",
		},
		{
			Name:    "Creates rule type engine for missing rule type and caches it",
			Cache:   cacheType{},
			DBSetup: dbf.NewDBMock(withRuleTypeLookup(&ruleType, nil)),
		},
		{
			Name:            "Returns error when building data source registry fails",
			Cache:           cacheType{},
			DBSetup:         dbf.NewDBMock(withRuleTypeLookup(&ruleType, nil)),
			dsRegistryError: errTest,
			ExpectedError:   errTest.Error(),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var store db.Store
			if scenario.DBSetup != nil {
				store = scenario.DBSetup(ctrl)
			}

			dssvc := mockdssvc.NewMockDataSourcesService(ctrl)
			reg := v1datasources.NewDataSourceRegistry()

			dssvc.EXPECT().BuildDataSourceRegistry(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(reg, scenario.dsRegistryError).AnyTimes()

			cache := ruleEngineCache{
				store:       store,
				provider:    testproviders.NewGitProvider(nil),
				ingestCache: ingestcache.NewNoopCache(),
				engines:     scenario.Cache,
				dssvc:       dssvc,
			}

			result, err := cache.GetRuleEngine(ctx, ruleTypeID)
			if scenario.ExpectedError != "" {
				require.ErrorContains(t, err, scenario.ExpectedError)
				require.Nil(t, result)
				// ensure that this rule type ID was not cached
				require.NotContains(t, cache.engines, ruleTypeID)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				// ensure that the value is present in the cache after testing
				require.Contains(t, cache.engines, ruleTypeID)
			}
		})
	}
}

var (
	ruleTypeID = uuid.New()
	errTest    = errors.New("error in rule type engine cache test")
	ruleType   = db.RuleType{
		ID:         ruleTypeID,
		Definition: []byte(ruleDefJSON),
	}
	malformedRuleType = db.RuleType{
		ID:         ruleTypeID,
		Definition: []byte(brokenRuleDef),
	}
)

func withRuleTypeLookup(ruleType *db.RuleType, err error) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		var rt db.RuleType
		if ruleType != nil {
			rt = *ruleType
		}
		mock.EXPECT().
			GetRuleTypeByID(gomock.Any(), ruleTypeID).
			Return(rt, err)
	}
}

// just enough of a rule type definition to pass validation
const ruleDefJSON = `
{
	"rule_schema": {},
	"ingest": {
		"type": "git",
        "git": {}
	},
	"eval": {
		"type": "jq",
		"jq": [{
			"ingested": {"def": ".abc"},
			"profile": {"def": ".xyz"}
		}]
	}
}
`

// just enough to create an error when instantiating the rule type engine
const brokenRuleDef = `
{
	"rule_schema": {},
	"ingest": {
		"type": "git",
        "git": {}
	}
}
`
