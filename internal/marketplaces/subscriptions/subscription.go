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

package subscriptions

import (
	"context"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/marketplaces/bundles"
	"github.com/stacklok/minder/internal/profiles"
	"github.com/stacklok/minder/internal/ruletypes"
)

// This does not define delete operations since we do not need them yet
// delete should get rid of the profiles/rules from the project and then
// delete the subscription
type SubscriptionService interface {
	// TODO: create subscription domain model type
	ListByBundle(ctx context.Context, bundle bundles.Bundle) ([]db.ListSubscriptionsByBundleRow, error)
	Create(ctx context.Context, projectID uuid.UUID, bundle bundles.Bundle) error
	Update(ctx context.Context, projectID uuid.UUID, bundle bundles.Bundle) error
	EnableProfile(ctx context.Context, projectID uuid.UUID, bundle bundles.Bundle, profileName string) error
}

type subscriptionService struct {
	profileService profiles.ProfileService
	ruleService    ruletypes.RuleTypeService
	store          db.Store
}

func (s *subscriptionService) ListByBundle(
	ctx context.Context,
	bundle bundles.Bundle,
) ([]db.ListSubscriptionsByBundleRow, error) {
	metadata := bundle.GetMetadata()
	return s.store.ListSubscriptionsByBundle(ctx, db.ListSubscriptionsByBundleParams{
		Namespace: metadata.Namespace,
		Name:      metadata.BundleName,
	})
}

// create subscription to bundle, apply all rule types
func (s *subscriptionService) Create(ctx context.Context, projectID uuid.UUID, bundle bundles.Bundle) error {
	metadata := bundle.GetMetadata()
	_, err := s.store.GetSubscriptionByProjectBundle(ctx, db.GetSubscriptionByProjectBundleParams{
		Namespace: metadata.Namespace,
		Name:      metadata.BundleName,
		ProjectID: projectID,
	})
	// project already subscribed to bundle, skip
	if err == nil {
		return nil
	}
	if err != nil {
		// we expect a "no result" type of error, filter here
		return err
	}

	// if creating new subscription, ensure bundle exists
	bundleID, err := s.ensureBundleExists(ctx, metadata.Namespace, metadata.BundleName)
	if err != nil {
		return err
	}

	// create subscription
	// TODO: do this in a transaction with operation which follow
	subscription, err := s.store.CreateSubscription(ctx, db.CreateSubscriptionParams{
		ProjectID:      projectID,
		BundleID:       bundleID,
		CurrentVersion: metadata.Version,
	})
	if err != nil {
		return err
	}

	// populate all rule types from this bundle into the project
	return s.createRuleTypesFromBundle(ctx, projectID, bundle, subscription.ID)
}

// if this project is subscribed to the bundle, check if the version is up to
// date and update any of the rules/profiles that this project uses from the
// bundle
func (s *subscriptionService) Update(ctx context.Context, projectID uuid.UUID, bundle bundles.Bundle) error {
	metadata := bundle.GetMetadata()
	subscription, err := s.store.GetSubscriptionByProjectBundle(ctx, db.GetSubscriptionByProjectBundleParams{
		Namespace: metadata.Namespace,
		Name:      metadata.BundleName,
		ProjectID: projectID,
	})
	if err != nil {
		return err
	}

	if !needsUpdate(metadata.Version, subscription.CurrentVersion) {
		// Nothing to do here...
		return nil
	}

	if err = s.updateRuleTypes(ctx, projectID, bundle, subscription.ID); err != nil {
		return err
	}
	if err = s.updateProfiles(ctx, projectID, bundle, subscription.ID); err != nil {
		return err
	}

	// TODO: we may want to do these changes in a transaction
	return s.store.SetCurrentVersion(ctx, db.SetCurrentVersionParams{
		CurrentVersion: metadata.Version,
		ProjectID:      projectID,
	})
}

func (s *subscriptionService) EnableProfile(
	ctx context.Context,
	projectID uuid.UUID,
	bundle bundles.Bundle,
	profileName string,
) error {
	// ensure project is subscribed to this bundle
	subscription, err := s.store.GetSubscriptionByProjectBundleVersion(ctx,
		db.GetSubscriptionByProjectBundleVersionParams{
			Namespace:      bundle.GetMetadata().Namespace,
			Name:           bundle.GetMetadata().BundleName,
			ProjectID:      projectID,
			CurrentVersion: bundle.GetMetadata().Version,
		},
	)
	if err != nil {
		return err
	}

	profile, err := bundle.GetProfile(profileName)
	if err != nil {
		return err
	}

	return s.profileService.CreateSubscriptionProfile(ctx, projectID, profile, subscription.ID)
}

// TODO: Can we implement an "upsert rules" operation?
func (s *subscriptionService) updateRuleTypes(
	ctx context.Context,
	projectID uuid.UUID,
	bundle bundles.Bundle,
	subscriptionID uuid.UUID,
) error {
	metadata := bundle.GetMetadata()
	subscriptionRules, err := s.store.ListSubscriptionRuleTypesInProject(ctx,
		db.ListSubscriptionRuleTypesInProjectParams{
			ProjectID: projectID,
			Namespace: metadata.Namespace,
			Name:      metadata.BundleName,
		},
	)
	if err != nil {
		return err
	}

	for _, rule := range subscriptionRules {
		newRule, err := bundle.GetRuleType(rule.Name)
		if err != nil {
			return err
		}

		err = s.ruleService.UpdateSubscriptionRule(ctx, rule.ID, newRule, subscriptionID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *subscriptionService) createRuleTypesFromBundle(
	ctx context.Context,
	projectID uuid.UUID,
	bundle bundles.Bundle,
	subscriptionID uuid.UUID,
) error {
	metadata := bundle.GetMetadata()
	for _, rule := range metadata.RuleTypes {
		newRule, err := bundle.GetRuleType(rule)
		if err != nil {
			return err
		}

		err = s.ruleService.CreateSubscriptionRule(ctx, newRule, subscriptionID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *subscriptionService) updateProfiles(
	ctx context.Context,
	projectID uuid.UUID,
	bundle bundles.Bundle,
	subscriptionID uuid.UUID,
) error {
	metadata := bundle.GetMetadata()
	// update rule types first
	subscriptionProfiles, err := s.store.ListSubscriptionProfilesInProject(ctx,
		db.ListSubscriptionProfilesInProjectParams{
			ID:        projectID,
			Namespace: metadata.Namespace,
			Name:      metadata.BundleName,
		},
	)
	if err != nil {
		return err
	}

	for _, profile := range subscriptionProfiles {
		newProfile, err := bundle.GetProfile(profile.Name)
		if err != nil {
			return err
		}

		err = s.profileService.UpdateSubscriptionProfile(ctx, profile.ID, newProfile, subscriptionID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *subscriptionService) ensureBundleExists(ctx context.Context, namespace, bundleName string) (uuid.UUID, error) {
	// TODO: this can probably be implemented as a single SQL query
	// or at least it should be inside a transaction
	id, err := s.store.BundleExists(ctx, db.BundleExistsParams{
		Namespace: namespace,
		Name:      bundleName,
	})
	if err == nil {
		// already exists
		return id, nil
	}
	if err != nil {
		// TODO: filter no row error
		return uuid.Nil, nil
	}
	newBundle, err := s.store.CreateBundle(ctx, db.CreateBundleParams{
		Namespace: namespace,
		Name:      bundleName,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return newBundle.ID, nil
}

// TODO: implement for real - use semver comparison logic
func needsUpdate(newVersion, currentVersion string) bool {
	return false
}
