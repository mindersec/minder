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
	"fmt"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/ingestcache"
	"github.com/stacklok/minder/internal/ruletypes"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// Cache contains a set of RuleTypeEngine instances
type Cache interface {
	// GetRuleEngine retrieves the rule type engine instance for the specified rule type
	GetRuleEngine(ctx context.Context, ruleTypeID uuid.UUID) (*RuleTypeEngine, error)
}

type cacheType = map[uuid.UUID]*RuleTypeEngine

type ruleEngineCache struct {
	store       db.Store
	provider    provinfv1.Provider
	ingestCache ingestcache.Cache
	engines     cacheType
}

// NewRuleEngineCache creates the rule engine cache
// It attempts to pre-populate the cache with all the relevant rule types
// for this entity and project hierarchy.
func NewRuleEngineCache(
	ctx context.Context,
	store db.Querier,
	entityType db.Entities,
	projectID uuid.UUID,
	provider provinfv1.Provider,
	ingestCache ingestcache.Cache,
) (Cache, error) {
	// Get the full project hierarchy
	hierarchy, err := store.GetParentProjects(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("error getting parent projects: %w", err)
	}

	// Query the rule types used in all rule instances for this entity type.
	// This is applied to every project in the hierarchy.
	ruleTypes, err := store.GetRuleTypesByEntityInHierarchy(ctx, db.GetRuleTypesByEntityInHierarchyParams{
		EntityType: entityType,
		Projects:   hierarchy,
	})
	if err != nil {
		return nil, fmt.Errorf("error while retrieving rule types from db: %w", err)
	}

	// Populate the cache with rule type engines for the rule types we found.
	engines := make(cacheType, len(ruleTypes))
	for _, ruleType := range ruleTypes {
		ruleEngine, err := cacheRuleEngine(ctx, &ruleType, provider, ingestCache, engines)
		if err != nil {
			return nil, err
		}
		engines[ruleType.ID] = ruleEngine
	}

	return &ruleEngineCache{engines: engines}, nil
}

func (r *ruleEngineCache) GetRuleEngine(ctx context.Context, ruleTypeID uuid.UUID) (*RuleTypeEngine, error) {
	if ruleTypeEngine, ok := r.engines[ruleTypeID]; ok {
		return ruleTypeEngine, nil
	}

	// If a new rule instance is added  to a profile after the rule engine
	// cache is populated, but before the list of profiles and rule instances
	// is queried, then the rule type may not be in the cache. This case is not
	// expected to happen often, so the code handles it by querying for that
	// rule type, building the rule type engine, and caching it.

	// In this part of the code, we can be sure that the rule type ID is
	// authorized for this project/user, since the rule type ID comes from
	// the rule_instances table, and it is validated before it is inserted
	// into that table.
	ruleType, err := r.store.GetRuleTypeByID(ctx, ruleTypeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("unknown rule type with ID: %s", ruleTypeID)
		}
		return nil, fmt.Errorf("error creating rule type engine: %s", ruleTypeID)
	}

	// If we find the rule type, insert into the cache and return.
	ruleTypeEngine, err := cacheRuleEngine(ctx, &ruleType, r.provider, r.ingestCache, r.engines)
	if err != nil {
		return nil, fmt.Errorf("error while caching rule type engine: %w", err)
	}
	return ruleTypeEngine, nil
}

func cacheRuleEngine(
	ctx context.Context,
	ruleType *db.RuleType,
	provider provinfv1.Provider,
	ingestCache ingestcache.Cache,
	engineCache cacheType,
) (*RuleTypeEngine, error) {
	// Parse the rule type
	pbRuleType, err := ruletypes.RuleTypePBFromDB(ruleType)
	if err != nil {
		return nil, fmt.Errorf("error parsing rule type when parsing rule type %s: %w", ruleType.ID, err)
	}

	// Create the rule type engine
	ruleEngine, err := NewRuleTypeEngine(ctx, pbRuleType, provider)
	if err != nil {
		return nil, fmt.Errorf("error creating rule type engine: %w", err)
	}

	// Add the rule type engine to the cache
	ruleEngine = ruleEngine.WithIngesterCache(ingestCache)
	engineCache[ruleType.ID] = ruleEngine
	return ruleEngine, nil
}
