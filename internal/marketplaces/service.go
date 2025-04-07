// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package marketplaces holds logic for the importing rule types and profiles
// from bundles into projects.
package marketplaces

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/db"
	sub "github.com/mindersec/minder/internal/marketplaces/subscriptions"
	"github.com/mindersec/minder/pkg/mindpak"
	"github.com/mindersec/minder/pkg/mindpak/reader"
	"github.com/mindersec/minder/pkg/mindpak/sources"
)

// Marketplace encapsulates the operations which allow profiles and rule types
// from bundles to projects. Subscriptions are implicitly created and managed
// by these operations.
type Marketplace interface {
	// Subscribe creates a subscription between the specified project and
	// bundle and adds all rules from that bundle to the project.
	Subscribe(
		ctx context.Context,
		projectID uuid.UUID,
		bundleID mindpak.BundleID,
		qtx db.ExtendQuerier,
	) error
	// AddProfile adds the specified profile from the bundle to the project.
	AddProfile(
		ctx context.Context,
		projectID uuid.UUID,
		bundleID mindpak.BundleID,
		profileName string,
		qtx db.Querier,
	) error
}

// trivial implementation of Marketplace with a single source
type marketplace struct {
	// ASSUMPTION: all sources are known at application startup
	// This will need more complex logic if external sources can be added
	// dynamically by customers.
	sources       map[mindpak.BundleID]sources.BundleSource
	subscriptions sub.SubscriptionService
}

func (s *marketplace) Subscribe(
	ctx context.Context,
	projectID uuid.UUID,
	bundleID mindpak.BundleID,
	qtx db.ExtendQuerier,
) error {
	bundle, err := s.getBundle(bundleID)
	if err != nil {
		return err
	}
	if err = s.subscriptions.Subscribe(ctx, projectID, bundle, qtx); err != nil {
		return fmt.Errorf("error while creating subscription: %w", err)
	}
	return nil
}

func (s *marketplace) AddProfile(
	ctx context.Context,
	projectID uuid.UUID,
	bundleID mindpak.BundleID,
	profileName string,
	qtx db.Querier,
) error {
	bundle, err := s.getBundle(bundleID)
	if err != nil {
		return err
	}

	if err = s.subscriptions.CreateProfile(ctx, projectID, bundle, profileName, qtx); err != nil {
		return fmt.Errorf("error while creating profile in project: %w", err)
	}

	return nil
}

func (s *marketplace) getBundle(bundleID mindpak.BundleID) (reader.BundleReader, error) {
	source, ok := s.sources[bundleID]
	if !ok {
		return nil, fmt.Errorf("unknown bundle: %s", bundleID)
	}
	bundle, err := source.GetBundle(bundleID)
	if err != nil {
		return nil, fmt.Errorf("error while retrieving bundle: %w", err)
	}
	return bundle, nil
}

// noopMarketplace is an instance of Marketplace which does nothing.
// This is used when the Marketplace functionality is disabled
type noopMarketplace struct{}

func (*noopMarketplace) Subscribe(_ context.Context, _ uuid.UUID, _ mindpak.BundleID, _ db.ExtendQuerier) error {
	return nil
}

func (*noopMarketplace) AddProfile(
	_ context.Context,
	_ uuid.UUID,
	_ mindpak.BundleID,
	_ string,
	_ db.Querier,
) error {
	return nil
}
