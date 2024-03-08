package subscriptions

import (
	"context"
	"github.com/google/uuid"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/marketplaces/bundles"
	"github.com/stacklok/minder/internal/profiles"
	"github.com/stacklok/minder/internal/ruletypes"
)

// DB
/*type Bundle struct {
	BundleID        uuid.UUID
	BundleNamespace string
	BundleName      string
	BundleVersion   string
}*/

type Subscription struct {
	BundleID  uuid.UUID
	ProfileID uuid.UUID
}

type SubscriptionService interface {
	ListForBundle(ctx context.Context, bundle bundles.Bundle) ([]Subscription, error)
	Create(ctx context.Context, projectID uuid.UUID, bundle bundles.Bundle) error
	Update(ctx context.Context, projectID uuid.UUID, bundle bundles.Bundle) error
	EnableProfile(ctx context.Context, projectID uuid.UUID, bundle bundles.Bundle, profileName string) error
}

type subscriptionService struct {
	profileService profiles.ProfileService
	ruleService    ruletypes.RuleTypeService
	store          db.Store
}

func (s *subscriptionService) ListForBundle(ctx context.Context, bundle bundles.Bundle) ([]Subscription, error) {
	metadata := bundle.GetMetadata()
	res, err := s.store.GetSubscriptionsByBundle(ctx, db.GetSubscriptionsByBundleParams{
		Namespace: metadata.Namespace,
		Name:      metadata.BundleName,
	})

}

func (s *subscriptionService) Create(ctx context.Context, projectID uuid.UUID, bundle bundles.Bundle) error {

}

func (s *subscriptionService) Update(ctx context.Context, projectID uuid.UUID, bundle bundles.Bundle) error {
	metadata := bundle.GetMetadata()
	currentVersion, err := s.store.GetCurrentVersionByProjectBundle(
		ctx,
		db.GetCurrentVersionByProjectBundleParams{
			Namespace: metadata.Namespace,
			Name:      metadata.BundleName,
			ProjectID: projectID,
		},
	)
	if err != nil {
		return err
	}
	if !needsUpdate(metadata.Version, currentVersion) {
		// Nothing to do here...
		return nil
	}

	for _, ruleTypeName := range metadata.RuleTypes {
		ruleType, err := bundle.GetRuleType(ruleTypeName)
		if err != nil {
			return err
		}
		err = s.ruleService.Update()
		if err != nil {
			return err
		}
	}

	for _, profileName := range metadata.Profiles {
		profile, err := bundle.GetProfile(profileName)
		if err != nil {
			return err
		}
		err = s.profileService.Update(ctx, projectID, profile)
		if err != nil {
			return err
		}
	}

	return s.store.SetCurrentVersion(ctx, db.SetCurrentVersionParams{
		StreamVersion: metadata.Version,
		ProjectID:     projectID,
	})
}

func (s *subscriptionService) EnableProfile(ctx context.Context, projectID uuid.UUID, bundle bundles.Bundle, profileName string) error {
	profile, err := bundle.GetProfile(profileName)
	if err != nil {
		return err
	}

	return s.profileService.Create(ctx, projectID, profile)
}

// TODO: implement for real
func needsUpdate(newVersion, currentVersion string) bool {
	return false
}
