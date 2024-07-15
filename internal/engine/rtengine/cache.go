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
	"fmt"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/ingestcache"
	"github.com/stacklok/minder/internal/ruletypes"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// Cache contains a set of RuleTypeEngine instances
type Cache interface {
	GetRuleEngine(ruleTypeID uuid.UUID) (*RuleTypeEngine, error)
}

type ruleEngineCache struct {
	engines map[uuid.UUID]*RuleTypeEngine
}

// NewRuleEngineCache creates the rule engine cache
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

	engines := make(map[uuid.UUID]*RuleTypeEngine, len(ruleTypes))
	for _, ruleType := range ruleTypes {
		// Parse the rule type
		pbRuleType, err := ruletypes.RuleTypePBFromDB(&ruleType)
		if err != nil {
			return nil, fmt.Errorf("error parsing rule type when parsing rule type %s: %w", ruleType.ID, err)
		}

		// Create the rule type engine
		ruleEngine, err := NewRuleTypeEngine(ctx, pbRuleType, provider)
		if err != nil {
			return nil, fmt.Errorf("error creating rule type engine: %w", err)
		}

		engines[ruleType.ID] = ruleEngine.WithIngesterCache(ingestCache)
	}

	return &ruleEngineCache{engines: engines}, nil
}

func (r *ruleEngineCache) GetRuleEngine(ruleTypeID uuid.UUID) (*RuleTypeEngine, error) {
	if ruleTypeEngine, ok := r.engines[ruleTypeID]; ok {
		return ruleTypeEngine, nil
	}
	return nil, fmt.Errorf("unknown rule type with ID: %s", ruleTypeID)
}
