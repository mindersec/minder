// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package mock provides a no-op implementation of the minder the authorization client
package mock

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/authz"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// NoopClient is a no-op implementation of the authz.Client interface, which always returns
// the same authorization result.
type NoopClient struct {
	// If Authorized is true, all Check calls will return nil
	Authorized bool
}

var _ authz.Client = &NoopClient{}

// Check implements authz.Client
func (n *NoopClient) Check(ctx context.Context, action string, project uuid.UUID) error {
	zerolog.Ctx(ctx).Debug().Str("action", action).Str("project", project.String()).Msg("noop authz check")
	if n.Authorized {
		return nil
	}
	return authz.ErrNotAuthorized
}

// Write_ implements authz.Client
func (*NoopClient) Write(_ context.Context, _ string, _ authz.Role, _ uuid.UUID) error {
	return nil
}

// Delete implements authz.Client
func (*NoopClient) Delete(_ context.Context, _ string, _ authz.Role, _ uuid.UUID) error {
	return nil
}

// DeleteUser implements authz.Client
func (*NoopClient) DeleteUser(_ context.Context, _ string) error {
	return nil
}

// AssignmentsToProject implements authz.Client
func (*NoopClient) AssignmentsToProject(_ context.Context, _ uuid.UUID) ([]*minderv1.RoleAssignment, error) {
	return nil, nil
}

// ProjectsForUser implements authz.Client
func (*NoopClient) ProjectsForUser(_ context.Context, _ string) ([]uuid.UUID, error) {
	return nil, nil
}

// PrepareForRun implements authz.Client
func (*NoopClient) PrepareForRun(_ context.Context) error {
	return nil
}

// MigrateUp implements authz.Client
func (*NoopClient) MigrateUp(_ context.Context) error {
	return nil
}

// Adopt implements authz.Client
func (*NoopClient) Adopt(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

// Orphan implements authz.Client
func (*NoopClient) Orphan(_ context.Context, _, _ uuid.UUID) error {
	return nil
}
