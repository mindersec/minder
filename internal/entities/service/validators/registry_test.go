// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package validators

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

// mockValidator is a simple validator for testing
type mockValidator struct {
	err error
}

func (m *mockValidator) Validate(_ context.Context, _ *properties.Properties, _ uuid.UUID) error {
	return m.err
}

func TestValidatorRegistry_AddAndHasValidators(t *testing.T) {
	t.Parallel()

	registry := NewValidatorRegistry()

	// Initially no validators
	assert.False(t, registry.HasValidators(pb.Entity_ENTITY_REPOSITORIES))
	assert.False(t, registry.HasValidators(pb.Entity_ENTITY_ARTIFACTS))

	// Add a validator for repositories
	v1 := &mockValidator{}
	handle := registry.AddValidator(pb.Entity_ENTITY_REPOSITORIES, v1)
	assert.NotEmpty(t, handle)

	// Now has validators for repos but not artifacts
	assert.True(t, registry.HasValidators(pb.Entity_ENTITY_REPOSITORIES))
	assert.False(t, registry.HasValidators(pb.Entity_ENTITY_ARTIFACTS))
}

func TestValidatorRegistry_RemoveValidator(t *testing.T) {
	t.Parallel()

	registry := NewValidatorRegistry()

	v1 := &mockValidator{}
	handle := registry.AddValidator(pb.Entity_ENTITY_REPOSITORIES, v1)

	assert.True(t, registry.HasValidators(pb.Entity_ENTITY_REPOSITORIES))

	registry.RemoveValidator(handle)

	assert.False(t, registry.HasValidators(pb.Entity_ENTITY_REPOSITORIES))
}

func TestValidatorRegistry_RemoveValidatorMultiple(t *testing.T) {
	t.Parallel()

	registry := NewValidatorRegistry()

	v1 := &mockValidator{}
	v2 := &mockValidator{}
	v3 := &mockValidator{}

	handle1 := registry.AddValidator(pb.Entity_ENTITY_REPOSITORIES, v1)
	_ = registry.AddValidator(pb.Entity_ENTITY_REPOSITORIES, v2)
	handle3 := registry.AddValidator(pb.Entity_ENTITY_REPOSITORIES, v3)

	// Remove middle one (v2 is between v1 and v3, but we remove v1)
	registry.RemoveValidator(handle1)
	assert.True(t, registry.HasValidators(pb.Entity_ENTITY_REPOSITORIES))

	// Remove last one
	registry.RemoveValidator(handle3)
	assert.True(t, registry.HasValidators(pb.Entity_ENTITY_REPOSITORIES)) // v2 still there

	// Adding handle1 back shouldn't work (already removed)
	registry.RemoveValidator(handle1)
	assert.True(t, registry.HasValidators(pb.Entity_ENTITY_REPOSITORIES))
}

func TestValidatorRegistry_ValidateNoValidators(t *testing.T) {
	t.Parallel()

	registry := NewValidatorRegistry()
	ctx := context.Background()
	props := properties.NewProperties(map[string]any{"key": "value"})
	projectID := uuid.New()

	// No validators = passes
	err := registry.Validate(ctx, pb.Entity_ENTITY_REPOSITORIES, props, projectID)
	assert.NoError(t, err)
}

func TestValidatorRegistry_ValidateSuccess(t *testing.T) {
	t.Parallel()

	registry := NewValidatorRegistry()
	ctx := context.Background()
	props := properties.NewProperties(map[string]any{"key": "value"})
	projectID := uuid.New()

	v1 := &mockValidator{err: nil}
	v2 := &mockValidator{err: nil}
	registry.AddValidator(pb.Entity_ENTITY_REPOSITORIES, v1)
	registry.AddValidator(pb.Entity_ENTITY_REPOSITORIES, v2)

	err := registry.Validate(ctx, pb.Entity_ENTITY_REPOSITORIES, props, projectID)
	assert.NoError(t, err)
}

func TestValidatorRegistry_ValidateFailure(t *testing.T) {
	t.Parallel()

	registry := NewValidatorRegistry()
	ctx := context.Background()
	props := properties.NewProperties(map[string]any{"key": "value"})
	projectID := uuid.New()

	expectedErr := errors.New("validation failed")
	v1 := &mockValidator{err: nil}
	v2 := &mockValidator{err: expectedErr}
	v3 := &mockValidator{err: nil}

	registry.AddValidator(pb.Entity_ENTITY_REPOSITORIES, v1)
	registry.AddValidator(pb.Entity_ENTITY_REPOSITORIES, v2)
	registry.AddValidator(pb.Entity_ENTITY_REPOSITORIES, v3)

	err := registry.Validate(ctx, pb.Entity_ENTITY_REPOSITORIES, props, projectID)
	assert.ErrorIs(t, err, expectedErr)
}

func TestValidatorRegistry_ValidateDifferentEntityTypes(t *testing.T) {
	t.Parallel()

	registry := NewValidatorRegistry()
	ctx := context.Background()
	props := properties.NewProperties(map[string]any{"key": "value"})
	projectID := uuid.New()

	repoErr := errors.New("repo error")
	artifactErr := errors.New("artifact error")

	repoValidator := &mockValidator{err: repoErr}
	artifactValidator := &mockValidator{err: artifactErr}

	registry.AddValidator(pb.Entity_ENTITY_REPOSITORIES, repoValidator)
	registry.AddValidator(pb.Entity_ENTITY_ARTIFACTS, artifactValidator)

	// Repos get repo error
	err := registry.Validate(ctx, pb.Entity_ENTITY_REPOSITORIES, props, projectID)
	assert.ErrorIs(t, err, repoErr)

	// Artifacts get artifact error
	err = registry.Validate(ctx, pb.Entity_ENTITY_ARTIFACTS, props, projectID)
	assert.ErrorIs(t, err, artifactErr)

	// Pull requests have no validators, so pass
	err = registry.Validate(ctx, pb.Entity_ENTITY_PULL_REQUESTS, props, projectID)
	assert.NoError(t, err)
}

func TestValidatorRegistry_ThreadSafety(t *testing.T) {
	t.Parallel()

	registry := NewValidatorRegistry()
	ctx := context.Background()
	props := properties.NewProperties(map[string]any{"key": "value"})
	projectID := uuid.New()

	var wg sync.WaitGroup
	const numGoroutines = 100

	// Start goroutines that add, remove, and validate concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(3)

		// Adder
		go func() {
			defer wg.Done()
			v := &mockValidator{}
			handle := registry.AddValidator(pb.Entity_ENTITY_REPOSITORIES, v)
			// Sometimes remove it
			registry.RemoveValidator(handle)
		}()

		// Validator
		go func() {
			defer wg.Done()
			_ = registry.Validate(ctx, pb.Entity_ENTITY_REPOSITORIES, props, projectID)
		}()

		// HasValidators checker
		go func() {
			defer wg.Done()
			_ = registry.HasValidators(pb.Entity_ENTITY_REPOSITORIES)
		}()
	}

	wg.Wait()
	// Test passes if no race conditions detected (run with -race)
}

func TestValidatorRegistry_RemoveInvalidHandle(t *testing.T) {
	t.Parallel()

	registry := NewValidatorRegistry()

	// Remove a handle that was never added - should not panic
	invalidHandle := ValidatorHandle{
		entityType: pb.Entity_ENTITY_REPOSITORIES,
		id:         99999,
	}
	require.NotPanics(t, func() {
		registry.RemoveValidator(invalidHandle)
	})
}
