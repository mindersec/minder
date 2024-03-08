package profiles

import (
	"context"
	"github.com/google/uuid"
	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

type ProfileService interface {
	Create(ctx context.Context, projectID uuid.UUID, profile *v1.Profile) error
	Update(ctx context.Context, projectID uuid.UUID, profile *v1.Profile) error
	Delete(ctx context.Context, projectID uuid.UUID, profile *v1.Profile) error
}
