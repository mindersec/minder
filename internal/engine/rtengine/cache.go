// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rtengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/ingestcache"
	eoptions "github.com/mindersec/minder/internal/engine/options"
	rtengine2 "github.com/mindersec/minder/pkg/engine/v1/rtengine"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
	"github.com/mindersec/minder/pkg/ruletypes"
)

// Cache contains a set of RuleTypeEngine instances
type Cache interface {
	// GetRuleEngine retrieves the rule type engine instance for the specified rule type
	GetRuleEngine(context.Context, uuid.UUID) (*rtengine2.RuleTypeEngine, error)
}

type cacheType = map[uuid.UUID]*rtengine2.RuleTypeEngine

type ruleEngineCache struct {
	store       db.Store
	provider    provinfv1.Provider
	ingestCache ingestcache.Cache
	engines     cacheType
	opts        []eoptions.Option
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
	opts ...eoptions.Option,
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
		ruleEngine, err := cacheRuleEngine(ctx, &ruleType, provider, ingestCache, engines, opts...)
		if err != nil {
			return nil, err
		}
		engines[ruleType.ID] = ruleEngine
	}

	return &ruleEngineCache{engines: engines, opts: opts}, nil
}

func (r *ruleEngineCache) GetRuleEngine(ctx context.Context, ruleTypeID uuid.UUID) (*rtengine2.RuleTypeEngine, error) {
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
	ruleTypeEngine, err := cacheRuleEngine(ctx, &ruleType, r.provider, r.ingestCache, r.engines, r.opts...)
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
	opts ...eoptions.Option,
) (*rtengine2.RuleTypeEngine, error) {
	// Parse the rule type
	pbRuleType, err := ruletypes.RuleTypePBFromDB(ruleType)
	if err != nil {
		return nil, fmt.Errorf("error parsing rule type when parsing rule type %s: %w", ruleType.ID, err)
	}

	// Create the rule type engine
	ruleEngine, err := rtengine2.NewRuleTypeEngine(ctx, pbRuleType, provider, opts...)
	if err != nil {
		return nil, fmt.Errorf("error creating rule type engine: %w", err)
	}

	// Add the rule type engine to the cache
	ruleEngine = ruleEngine.WithIngesterCache(ingestCache)
	engineCache[ruleType.ID] = ruleEngine
	return ruleEngine, nil
}
