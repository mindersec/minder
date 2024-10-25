// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profiles

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/exp/maps"

	"github.com/mindersec/minder/pkg/db"
	"github.com/mindersec/minder/pkg/profiles/models"
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

	// Get all the rule instances in the project hierarchy for this entity.
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
	rulesByProfileID := map[uuid.UUID][]models.RuleInstance{}
	for _, dbRule := range rules {
		rule, err := models.RuleFromDB(dbRule)
		if err != nil {
			return nil, fmt.Errorf("error while procesing rule instance %s: %w", rule.RuleTypeID, err)
		}
		ruleList := rulesByProfileID[dbRule.ProfileID]
		ruleList = append(ruleList, rule)
		rulesByProfileID[dbRule.ProfileID] = ruleList
	}

	// Get all profiles which belong to the above rule instances
	// The query is written in such a way that if a profile is deleted between
	// the rule instance query and this query, it will simply be omitted from
	// the results.
	profiles, err := p.store.BulkGetProfilesByID(ctx, maps.Keys(rulesByProfileID))
	if err != nil {
		return nil, fmt.Errorf("error while querying profiles to use in evaluation: %w", err)
	}

	// Finally, create the ProfileAggregate instances
	aggregates := make([]models.ProfileAggregate, len(profiles))
	for _, profile := range profiles {
		profileRules, ok := rulesByProfileID[profile.Profile.ID]
		if !ok {
			return nil, fmt.Errorf("could not find rule instances for profile %s: %w", profile.Profile.ID, err)
		}
		aggregate := models.ProfileAggregate{
			ID:   profile.Profile.ID,
			Name: profile.Profile.Name,
			ActionConfig: models.ActionConfiguration{
				Remediate: models.ActionOptFromDB(profile.Profile.Remediate),
				Alert:     models.ActionOptFromDB(profile.Profile.Alert),
			},
			Rules:     profileRules,
			Selectors: models.SelectorSliceFromDB(profile.ProfilesWithSelectors),
		}
		aggregates = append(aggregates, aggregate)
	}

	return aggregates, nil
}
