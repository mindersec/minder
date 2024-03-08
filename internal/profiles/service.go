package profiles

import (
	"context"

	"github.com/google/uuid"

	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// TODO: the implementation of this interface should contain logic from the
// controlplane methods for profile create/update/delete
type ProfileService interface {
	CreateProfile(ctx context.Context, projectID uuid.UUID, profile *v1.Profile) error
	CreateSubscriptionProfile(ctx context.Context, projectID uuid.UUID, profile *v1.Profile, subscriptionID uuid.UUID) error
	UpdateProfile(ctx context.Context, projectID uuid.UUID, profile *v1.Profile) error
	UpdateSubscriptionProfile(ctx context.Context, projectID uuid.UUID, profile *v1.Profile, subscriptionID uuid.UUID) error
	DeleteProfile(ctx context.Context, projectID uuid.UUID) error
	DeleteSubscriptionProfile(ctx context.Context, projectID uuid.UUID, subscriptionID uuid.UUID) error
}
