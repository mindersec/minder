// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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

	datasourceservice "github.com/mindersec/minder/internal/datasources/service"
	"github.com/mindersec/minder/internal/db"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/mindpak"
	"github.com/mindersec/minder/pkg/mindpak/reader"
	profsvc "github.com/mindersec/minder/pkg/profiles"
	"github.com/mindersec/minder/pkg/ruletypes"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// SubscriptionService defines operations on the subscriptions, as well as the
// profiles and rules linked to subscriptions
// It is assumed that all methods will be called in the context of a
// transaction, and they take
type SubscriptionService interface {
	// Subscribe creates a subscription record for the specified project
	// and bundle. It is a no-op if the project is already subscribed.
	Subscribe(
		ctx context.Context,
		projectID uuid.UUID,
		bundle reader.BundleReader,
		qtx db.ExtendQuerier,
	) error
	// CreateProfile creates the specified profile from the bundle in the project.
	CreateProfile(
		ctx context.Context,
		projectID uuid.UUID,
		bundle reader.BundleReader,
		profileName string,
		qtx db.Querier,
	) error
}

type subscriptionService struct {
	profiles    profsvc.ProfileService
	rules       ruletypes.RuleTypeService
	dataSources datasourceservice.DataSourcesService
}

// NewSubscriptionService creates an instance of the SubscriptionService interface
func NewSubscriptionService(
	profiles profsvc.ProfileService,
	rules ruletypes.RuleTypeService,
	dataSources datasourceservice.DataSourcesService,
) SubscriptionService {
	return &subscriptionService{
		profiles:    profiles,
		rules:       rules,
		dataSources: dataSources,
	}
}

func (s *subscriptionService) Subscribe(
	ctx context.Context,
	projectID uuid.UUID,
	bundle reader.BundleReader,
	qtx db.ExtendQuerier,
) error {
	metadata := bundle.GetMetadata()
	_, err := qtx.GetSubscriptionByProjectBundle(ctx, db.GetSubscriptionByProjectBundleParams{
		Namespace: metadata.Namespace,
		Name:      metadata.Name,
		ProjectID: projectID,
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
		ProjectID:      projectID,
		BundleID:       bundleID,
		CurrentVersion: metadata.Version,
	})
	if err != nil {
		return fmt.Errorf("error while creating subscription: %w", err)
	}

	// populate all data sources from this bundle into the project
	// this should happen before populating the rules, as rules may depend on data sources
	err = s.upsertBundleDataSources(ctx, qtx, projectID, bundle, subscription.ID)
	if err != nil {
		return fmt.Errorf("error while creating data sources in project: %w", err)
	}

	// populate all rule types from this bundle into the project
	err = s.upsertBundleRules(ctx, qtx, projectID, bundle, subscription.ID)
	if err != nil {
		return fmt.Errorf("error while creating rules in project: %w", err)
	}

	return nil
}

func (s *subscriptionService) CreateProfile(
	ctx context.Context,
	projectID uuid.UUID,
	bundle reader.BundleReader,
	profileName string,
	qtx db.Querier,
) error {
	// ensure project is subscribed to this bundle
	subscription, err := s.findSubscription(ctx, qtx, projectID, bundle.GetMetadata())
	if err != nil {
		return err
	}

	profile, err := bundle.GetProfile(profileName)
	if err != nil {
		return fmt.Errorf("error while retrieving profile from bundle: %w", err)
	}

	_, err = s.profiles.CreateProfile(ctx, projectID, subscription.ID, profile, qtx)
	if err != nil {
		return fmt.Errorf("error while creating profile in project: %w", err)
	}
	return nil
}

func (_ *subscriptionService) findSubscription(
	ctx context.Context,
	qtx db.Querier,
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
	qtx db.Querier,
	namespace, bundleName string,
) (uuid.UUID, error) {
	// This is a no-op if this namespace/name pair already exists
	err := qtx.UpsertBundle(ctx, db.UpsertBundleParams{
		Namespace: namespace,
		Name:      bundleName,
	})
	if err != nil {
		return uuid.Nil, err
	}

	// fetch the row after insertion so we can get the autogenerated ID
	// TODO: may be possible to do both queries in a single SQL query
	bundle, err := qtx.GetBundle(ctx, db.GetBundleParams{
		Namespace: namespace,
		Name:      bundleName,
	})
	if err != nil {
		return uuid.Nil, err
	}

	return bundle.ID, nil
}

func (s *subscriptionService) upsertBundleRules(
	ctx context.Context,
	qtx db.Querier,
	projectID uuid.UUID,
	bundle reader.BundleReader,
	subscriptionID uuid.UUID,
) error {
	return bundle.ForEachRuleType(func(ruleType *minderv1.RuleType) error {
		return s.rules.UpsertRuleType(ctx, projectID, subscriptionID, ruleType, qtx)
	})
}

func (s *subscriptionService) upsertBundleDataSources(
	ctx context.Context,
	qtx db.ExtendQuerier,
	projectID uuid.UUID,
	bundle reader.BundleReader,
	subscriptionID uuid.UUID,
) error {
	return bundle.ForEachDataSource(func(dataSource *minderv1.DataSource) error {
		return s.dataSources.Upsert(ctx, projectID, subscriptionID, dataSource, datasourceservice.OptionsBuilder().WithTransaction(qtx))
	})
}
