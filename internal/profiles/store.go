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

package profiles

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/profiles/models"
)

// ProfileStore encapsulates operations for fetching ProfileAggregates
type ProfileStore interface {
	GetProfilesForEvaluation(
		ctx context.Context,
		projectID uuid.UUID,
		entityType db.Entities,
	) ([]models.ProfileAggregate, error)
}

// NewProfileStore creates an instance of ProfileStore
func NewProfileStore(store db.Store) ProfileStore {
	return &profileStore{store: store}
}

type profileStore struct {
	store db.Store
}

func (p *profileStore) GetProfilesForEvaluation(
	ctx context.Context,
	projectID uuid.UUID,
	entityType db.Entities,
) ([]models.ProfileAggregate, error) {
	// Get the list of parent projects for the current project
	// This allows us to get all profiles in our hierarchy.
	projects, err := p.store.GetParentProjects(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("error while querying project hierarchy: %w", err)
	}

	// Get all profiles which belong in the project hierarchy, and which have
	// at least one rule instance for the specified entity type.
	profiles, err := p.store.GetProfilesInProjectsWithEntity(ctx,
		db.GetProfilesInProjectsWithEntityParams{
			EntityType: entityType,
			ProjectIds: projects,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error while querying profiles to use in evaluation: %w", err)
	}

	// Get all the rule instances in the project hierarchy for this entity.
	// Assumption: Set of rule instances line up with the set of profiles above.
	rules, err := p.store.GetRuleInstancesEntityInProjects(ctx,
		db.GetRuleInstancesEntityInProjectsParams{
			EntityType: entityType,
			ProjectIds: projects,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error while querying rule instances to use in evaluation: %w", err)
	}

	// Transform the rule instances into the structure we want, and group by project ID.
	rulesByProfileID := make(map[uuid.UUID][]models.RuleInstance, len(profiles))
	for _, dbRule := range rules {
		rule, err := models.RuleFromDB(dbRule)
		if err != nil {
			return nil, fmt.Errorf("error while procesing rule instance %s: %w", rule.RuleTypeID, err)
		}
		ruleList := rulesByProfileID[dbRule.ProfileID]
		ruleList = append(ruleList, rule)
		rulesByProfileID[dbRule.ProfileID] = ruleList
	}

	// Finally, create the ProfileAggregate instances
	aggregates := make([]models.ProfileAggregate, len(profiles))
	for _, profile := range profiles {
		profileRules, ok := rulesByProfileID[profile.ID]
		if !ok {
			return nil, fmt.Errorf("could not find rule instances for profile %s: %w", profile.ID, err)
		}
		aggregate := models.ProfileAggregate{
			ID:   profile.ID,
			Name: profile.Name,
			ActionConfig: models.ActionConfiguration{
				Remediate: models.ActionOptFromDB(profile.Remediate),
				Alert:     models.ActionOptFromDB(profile.Alert),
			},
			Rules: profileRules,
		}
		aggregates = append(aggregates, aggregate)
	}

	return aggregates, nil
}
