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

	"github.com/mindersec/minder/internal/engine/ingestcache"
	"github.com/mindersec/minder/internal/providers/testproviders"
	"github.com/mindersec/minder/pkg/db"
	dbf "github.com/mindersec/minder/pkg/db/fixtures"
	rtengine2 "github.com/mindersec/minder/pkg/engine/v1/rtengine"
)

func TestGetRuleEngine(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		Name          string
		Cache         cacheType
		DBSetup       dbf.DBMockBuilder
		ExpectedError string
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

			cache := ruleEngineCache{
				store:       store,
				provider:    testproviders.NewGitProvider(nil),
				ingestCache: ingestcache.NewNoopCache(),
				engines:     scenario.Cache,
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
			"ingested": {"def": "abc"},
			"profile": {"def": "xyz"}
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
