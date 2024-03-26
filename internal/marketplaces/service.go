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

// Package marketplaces holds logic for the importing rule types and profiles
// from bundles into projects.
package marketplaces

import (
	"context"
	"fmt"

	"github.com/stacklok/minder/internal/db"
	sub "github.com/stacklok/minder/internal/marketplaces/subscriptions"
	"github.com/stacklok/minder/internal/marketplaces/types"
	"github.com/stacklok/minder/pkg/mindpak"
	"github.com/stacklok/minder/pkg/mindpak/sources"
)

// Marketplace encapsulates the operations which allow profiles and rule types
// from bundles to projects. Subscriptions are implicitly created and managed
// by these operations.
type Marketplace interface {
	Subscribe(
		ctx context.Context,
		project types.ProjectContext,
		bundleID mindpak.BundleID,
		qtx db.ExtendQuerier,
	) error
	AddProfile(
		ctx context.Context,
		project types.ProjectContext,
		bundleID mindpak.BundleID,
		profileName string,
		qtx db.ExtendQuerier,
	) error
}

// trivial implementation of Marketplace with a single source
type singleSourceMarketplace struct {
	source        sources.BundleSource
	subscriptions sub.SubscriptionService
}

// NewSingleSourceMarketplace creates an instance of Marketplace with a single source
func NewSingleSourceMarketplace(source sources.BundleSource, subscriptions sub.SubscriptionService) Marketplace {
	return &singleSourceMarketplace{
		source:        source,
		subscriptions: subscriptions,
	}
}

func (s *singleSourceMarketplace) Subscribe(
	ctx context.Context,
	project types.ProjectContext,
	bundleID mindpak.BundleID,
	qtx db.ExtendQuerier,
) error {
	bundle, err := s.source.GetBundle(bundleID)
	if err != nil {
		return fmt.Errorf("error while retrieving bundle: %w", err)
	}

	if err = s.subscriptions.Subscribe(ctx, project, bundle, qtx); err != nil {
		return fmt.Errorf("error while creating subscription: %w", err)
	}
	return nil
}

func (s *singleSourceMarketplace) AddProfile(
	ctx context.Context,
	project types.ProjectContext,
	bundleID mindpak.BundleID,
	profileName string,
	qtx db.ExtendQuerier,
) error {
	bundle, err := s.source.GetBundle(bundleID)
	if err != nil {
		return fmt.Errorf("error while retrieving bundle: %w", err)
	}

	if err = s.subscriptions.CreateProfile(ctx, project, bundle, profileName, qtx); err != nil {
		return fmt.Errorf("error while creating profile in project: %w", err)
	}

	return nil
}
