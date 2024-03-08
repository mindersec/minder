package marketplaces

import (
	"context"
	"github.com/google/uuid"
	"github.com/stacklok/minder/internal/marketplaces/bundles"
	"github.com/stacklok/minder/internal/marketplaces/subscriptions"
)

type HealthcheckService interface {
	AddProject(ctx context.Context, projectID uuid.UUID) error
	UpdateHealthcheckProjects(ctx context.Context) error
}

const (
	healthcheckProfile = "health-check.yaml"
)

type defaultHealthcheckService struct {
	healthCheckBundle bundles.Bundle
	bundleService     subscriptions.SubscriptionService
}

func (d *defaultHealthcheckService) AddProject(ctx context.Context, projectID uuid.UUID) error {
	err := d.bundleService.Create(ctx, projectID, d.healthCheckBundle)
	if err != nil {
		return err
	}
	return d.bundleService.EnableProfile(ctx, projectID, d.healthCheckBundle, healthcheckProfile)
}

func (d *defaultHealthcheckService) UpdateHealthcheckProjects(ctx context.Context) error {
	subs, err := d.bundleService.ListForBundle(ctx, d.healthCheckBundle)
	for _, subscription := range subs {
		if err = d.bundleService.Update(ctx, subscription.ProfileID, d.healthCheckBundle); err != nil {
			return err
		}
	}
	return nil
}

type noopHealthcheckService struct{}

func (n *noopHealthcheckService) AddProject(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (n *noopHealthcheckService) UpdateHealthcheckProjects(_ context.Context) error {
	return nil
}
