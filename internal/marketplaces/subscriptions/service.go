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

// Package subscriptions contains logic relating to the concept of
// `subscriptions` - which describe a linkage between a project and a
// marketplace bundle
package subscriptions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/marketplaces/types"
	profsvc "github.com/stacklok/minder/internal/profiles"
	"github.com/stacklok/minder/internal/ruletypes"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/stacklok/minder/pkg/mindpak"
	"github.com/stacklok/minder/pkg/mindpak/reader"
)

// SubscriptionService defines operations on the subscriptions, as well as the
// profiles and rules linked to subscriptions
// It is assumed that all methods will be called in the context of a
// transaction, and they take
type SubscriptionService interface {
	// Subscribe creates a subscription record for the specified project
	// and bundle. It is a no-op if the project is already subscribed.
	Subscribe(
		ctx context.Context,
		project types.ProjectContext,
		bundle reader.BundleReader,
		qtx db.ExtendQuerier,
	) error
	// CreateProfile creates the specified profile from the bundle in the project.
	CreateProfile(
		ctx context.Context,
		project types.ProjectContext,
		bundle reader.BundleReader,
		profileName string,
		qtx db.ExtendQuerier,
	) error
}

type subscriptionService struct {
	profiles profsvc.ProfileService
	rules    ruletypes.RuleTypeService
}

// NewSubscriptionService creates an instance of the SubscriptionService interface
func NewSubscriptionService(
	profiles profsvc.ProfileService,
	rules ruletypes.RuleTypeService,
) SubscriptionService {
	return &subscriptionService{
		profiles: profiles,
		rules:    rules,
	}
}

func (s *subscriptionService) Subscribe(
	ctx context.Context,
	project types.ProjectContext,
	bundle reader.BundleReader,
	qtx db.ExtendQuerier,
) error {
	metadata := bundle.GetMetadata()
	_, err := qtx.GetSubscriptionByProjectBundle(ctx, db.GetSubscriptionByProjectBundleParams{
		Namespace: metadata.Namespace,
		Name:      metadata.Name,
		ProjectID: project.ID,
	})
	// project already subscribed to bundle, skip
	if err == nil {
		return nil
	}
	// we expect the query to have no results
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("error while querying subscriptions: %w", err)
	}

	// if creating new subscription, ensure bundle exists
	bundleID, err := ensureBundleExists(ctx, qtx, metadata.Namespace, metadata.Name)
	if err != nil {
		return fmt.Errorf("error while ensuring bundle exists: %w", err)
	}

	// create subscription
	subscription, err := qtx.CreateSubscription(ctx, db.CreateSubscriptionParams{
		ProjectID:      project.ID,
		BundleID:       bundleID,
		CurrentVersion: metadata.Version,
	})
	if err != nil {
		return fmt.Errorf("error while creating subscription: %w", err)
	}

	// populate all rule types from this bundle into the project
	err = s.upsertBundleRules(ctx, qtx, project.ID, project.Provider, bundle, subscription.ID)
	if err != nil {
		return fmt.Errorf("error while creating rules in project: %w", err)
	}
	return nil
}

func (s *subscriptionService) CreateProfile(
	ctx context.Context,
	project types.ProjectContext,
	bundle reader.BundleReader,
	profileName string,
	qtx db.ExtendQuerier,
) error {
	// ensure project is subscribed to this bundle
	subscription, err := s.findSubscription(ctx, qtx, project.ID, bundle.GetMetadata())
	if err != nil {
		return err
	}

	profile, err := bundle.GetProfile(profileName)
	if err != nil {
		return fmt.Errorf("error while retrieving profile from bundle: %w", err)
	}

	_, err = s.profiles.CreateProfile(ctx, project.ID, project.Provider, subscription.ID, profile, qtx)
	if err != nil {
		return fmt.Errorf("error while creating profile in project: %w", err)
	}
	return nil
}

func (_ *subscriptionService) findSubscription(
	ctx context.Context,
	qtx db.ExtendQuerier,
	projectID uuid.UUID,
	metadata *mindpak.Metadata,
) (result db.Subscription, err error) {
	result, err = qtx.GetSubscriptionByProjectBundle(ctx,
		db.GetSubscriptionByProjectBundleParams{
			Namespace: metadata.Namespace,
			Name:      metadata.Name,
			ProjectID: projectID,
		},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result, fmt.Errorf("project %s is not subscribed to bundle %s/%s",
				projectID, metadata.Namespace, metadata.Name)
		}
		return result, fmt.Errorf("error while querying subscriptions: %w", err)
	}
	return result, nil
}

func ensureBundleExists(
	ctx context.Context,
	qtx db.ExtendQuerier,
	namespace, bundleName string,
) (uuid.UUID, error) {
	// This is a no-op if this namespace/name pair already exists
	dbBundle, err := qtx.UpsertBundle(ctx, db.UpsertBundleParams{
		Namespace: namespace,
		Name:      bundleName,
	})
	if err != nil {
		return uuid.Nil, err
	}

	return dbBundle.ID, nil
}

func (s *subscriptionService) upsertBundleRules(
	ctx context.Context,
	qtx db.ExtendQuerier,
	projectID uuid.UUID,
	provider *db.Provider,
	bundle reader.BundleReader,
	subscriptionID uuid.UUID,
) error {
	return bundle.ForEachRuleType(func(ruleType *minderv1.RuleType) error {
		return s.rules.UpsertRuleType(ctx, projectID, provider, subscriptionID, ruleType, qtx)
	})
}
