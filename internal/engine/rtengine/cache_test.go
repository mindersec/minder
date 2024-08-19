// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rtengine

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
	"github.com/stacklok/minder/internal/engine/ingestcache"
	"github.com/stacklok/minder/internal/providers/testproviders"
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
			Cache: cacheType{ruleTypeID: &RuleTypeEngine{}},
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
