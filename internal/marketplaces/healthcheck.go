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
	"context"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/marketplaces/bundles"
	"github.com/stacklok/minder/internal/marketplaces/subscriptions"
)

// rather than model a `Marketplace` type, let's just build a throwaway
// Healthcheck interface which loads the healthcheck bundle and calls the
// correct methods on the `Bundle` and `SubscriptionService` interfaces
type HealthcheckService interface {
	OnboardProject(ctx context.Context, projectID uuid.UUID) error
	UpdateHealthcheckProjects(ctx context.Context) error
}

const (
	healthcheckProfile = "health-check.yaml"
)

// TODO: write a `NewHealthcheckService` function which:
//  1. Attempts to read bundle off disk
//  2. Returns an instance of `noopHealthCheckService` if it does not exist
//  3. Otherwise attempts to instantiate a struct which implements the `Bundle`
//     interface
//  4. Creates an instance of `defaultHealthCheckService` with bundle
func NewHealthcheckService(pathToBundle string) (HealthcheckService, error) {
	return &noopHealthcheckService{}, nil
}

type defaultHealthcheckService struct {
	healthCheckBundle bundles.Bundle
	subscriptions     subscriptions.SubscriptionService
}

func (d *defaultHealthcheckService) OnboardProject(ctx context.Context, projectID uuid.UUID) error {
	err := d.subscriptions.Create(ctx, projectID, d.healthCheckBundle)
	if err != nil {
		return err
	}
	return d.subscriptions.EnableProfile(ctx, projectID, d.healthCheckBundle, healthcheckProfile)
}

func (d *defaultHealthcheckService) UpdateHealthcheckProjects(ctx context.Context) error {
	subs, err := d.subscriptions.ListByBundle(ctx, d.healthCheckBundle)
	for _, subscription := range subs {
		if err = d.subscriptions.Update(ctx, subscription.ProjectID, d.healthCheckBundle); err != nil {
			return err
		}
	}
	return nil
}

// noop implementation for minder instances where the bundle is not present
type noopHealthcheckService struct{}

func (n *noopHealthcheckService) OnboardProject(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (n *noopHealthcheckService) UpdateHealthcheckProjects(_ context.Context) error {
	return nil
}
