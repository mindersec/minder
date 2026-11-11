// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package validators

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/mindersec/minder/internal/providers/oci"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/pkg/entities/properties"
)

// ErrContainerArtifactTagRequired is returned when a container artifact is registered without a tag
var ErrContainerArtifactTagRequired = util.UserVisibleError(codes.InvalidArgument,
	"container artifacts must be registered with an explicit tag, e.g. \"alpine:latest\" instead of \"alpine\"")

// ArtifactValidator validates artifact entity creation.
// This validator should be registered for ENTITY_ARTIFACTS using
// the ValidatorRegistry.AddValidator method.
type ArtifactValidator struct{}

// NewArtifactValidator creates a new ArtifactValidator
func NewArtifactValidator() *ArtifactValidator {
	return &ArtifactValidator{}
}

// Validate checks if an artifact entity can be created.
func (*ArtifactValidator) Validate(
	_ context.Context,
	props *properties.Properties,
	_ uuid.UUID,
) error {
	if props.GetProperty(oci.ArtifactPropertyRegistry) == nil {
		return nil
	}

	name, err := oci.ArtifactNameFromProperties(props)
	if err != nil {
		return fmt.Errorf("error checking container artifact name: %w", err)
	}
	if !oci.HasExplicitTag(name) {
		return ErrContainerArtifactTagRequired
	}

	return nil
}
