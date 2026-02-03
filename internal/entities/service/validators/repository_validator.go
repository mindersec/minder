// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package validators contains entity validation logic
package validators

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/projects/features"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/pkg/entities/properties"
)

var (
	// ErrPrivateRepoForbidden is returned when a private repository is not allowed
	ErrPrivateRepoForbidden = util.UserVisibleError(codes.InvalidArgument, "private repositories are not allowed in this project")
	// ErrArchivedRepoForbidden is returned when an archived repository cannot be registered
	ErrArchivedRepoForbidden = util.UserVisibleError(codes.InvalidArgument, "archived repositories cannot be registered")
)

// RepositoryValidator validates repository entity creation.
// This validator should be registered for ENTITY_REPOSITORIES using
// the ValidatorRegistry.AddValidator method.
type RepositoryValidator struct {
	store db.Store
}

// NewRepositoryValidator creates a new RepositoryValidator
func NewRepositoryValidator(store db.Store) *RepositoryValidator {
	return &RepositoryValidator{store: store}
}

// Validate checks if a repository entity can be created.
// This validator is called only for repositories since it's registered
// specifically for that entity type via the ValidatorRegistry.
func (v *RepositoryValidator) Validate(
	ctx context.Context,
	props *properties.Properties,
	projectID uuid.UUID,
) error {
	// Check if archived
	isArchived, err := props.GetProperty(properties.RepoPropertyIsArchived).AsBool()
	if err != nil {
		return fmt.Errorf("error checking is_archived property: %w", err)
	}
	if isArchived {
		return ErrArchivedRepoForbidden
	}

	// Check if private
	isPrivate, err := props.GetProperty(properties.RepoPropertyIsPrivate).AsBool()
	if err != nil {
		return fmt.Errorf("error checking is_private property: %w", err)
	}
	if isPrivate && !features.ProjectAllowsPrivateRepos(ctx, v.store, projectID) {
		return ErrPrivateRepoForbidden
	}

	return nil
}
