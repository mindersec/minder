// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package marketplaces

import (
	"errors"
	"fmt"
	"path/filepath"

	sub "github.com/mindersec/minder/internal/marketplaces/subscriptions"
	"github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/mindpak"
	src "github.com/mindersec/minder/pkg/mindpak/sources"
	"github.com/mindersec/minder/pkg/profiles"
	"github.com/mindersec/minder/pkg/ruletypes"
)

// NewMarketplaceFromServiceConfig takes the Minder service config and
// instantiates the object graph needed for the Marketplace. If the marketplace
// functionality is disabled in the config or missing, this returns a no-op
// implementation of Marketplace. Otherwise, it loads the bundle specified in
// the service config and builds a single-source Marketplace for it.
func NewMarketplaceFromServiceConfig(
	config server.MarketplaceConfig,
	profile profiles.ProfileService,
	ruleType ruletypes.RuleTypeService,
) (Marketplace, error) {
	if !config.Enabled {
		return NewNoopMarketplace(), nil
	}

	cfgSources := config.Sources
	// If the marketplace is enabled, require at least one source to be
	// defined.
	if len(cfgSources) == 0 {
		return nil, errors.New("no sources defined in marketplace config")
	}

	newSources := make([]src.BundleSource, len(cfgSources))
	for i, cfgSource := range cfgSources {
		// This is just used for validation
		// TODO: support other sources
		if t, err := cfgSource.GetType(); err != nil || t != server.TgzSource {
			return nil, fmt.Errorf("unexpected source type: %s", cfgSource.Type)
		}

		tarPath := filepath.Clean(cfgSource.Location)
		source, err := src.NewSourceFromTarGZ(tarPath)
		if err != nil {
			return nil, fmt.Errorf("unable to load tar from path %s: %w", tarPath, err)
		}

		newSources[i] = source
	}

	subscription := sub.NewSubscriptionService(profile, ruleType)
	marketplace, err := NewMarketplace(newSources, subscription)
	if err != nil {
		return nil, fmt.Errorf("error while creating marketplace: %w", err)
	}
	return marketplace, nil
}

// NewMarketplace creates an instance of Marketplace with a single source
func NewMarketplace(sources []src.BundleSource, subscriptions sub.SubscriptionService) (Marketplace, error) {
	sourceMapping := make(map[mindpak.BundleID]src.BundleSource)
	for _, source := range sources {
		bundles, err := source.ListBundles()
		if err != nil {
			return nil, fmt.Errorf("error while listing bundles: %w", err)
		}
		for _, id := range bundles {
			sourceMapping[id] = source
		}
	}
	return &marketplace{
		sources:       sourceMapping,
		subscriptions: subscriptions,
	}, nil
}

// NewNoopMarketplace returns an instance of Marketplace which does nothing
// when any methods are called.
func NewNoopMarketplace() Marketplace {
	return &noopMarketplace{}
}
