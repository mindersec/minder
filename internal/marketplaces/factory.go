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

package marketplaces

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/stacklok/minder/internal/config/server"
	sub "github.com/stacklok/minder/internal/marketplaces/subscriptions"
	"github.com/stacklok/minder/internal/profiles"
	"github.com/stacklok/minder/internal/ruletypes"
	"github.com/stacklok/minder/pkg/mindpak"
	src "github.com/stacklok/minder/pkg/mindpak/sources"
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
