// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package validators contains entity validation logic
package validators

import (
	"context"
	"sync"

	"github.com/google/uuid"

	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
)

// Validator validates an entity of a specific type.
// Unlike a general validator, this does NOT receive entityType since
// validators are registered for specific entity types.
type Validator interface {
	// Validate returns nil if entity is valid, error otherwise.
	Validate(
		ctx context.Context,
		props *properties.Properties,
		projectID uuid.UUID,
	) error
}

// ValidatorHandle is an opaque handle returned when registering a validator.
// It can be used to remove the validator later.
type ValidatorHandle struct {
	id uint64
}

// ValidatorRegistry manages entity validators by entity type.
// Validators register for specific entity types and are called when
// entities of that type are created.
type ValidatorRegistry interface {
	// AddValidator registers a validator for a specific entity type.
	// Returns a handle that can be used to remove the validator.
	AddValidator(entityType pb.Entity, validator Validator) ValidatorHandle

	// RemoveValidator removes a previously registered validator.
	RemoveValidator(handle ValidatorHandle)

	// Validate runs all validators registered for the given entity type.
	// Returns nil if no validators are registered (validation is optional).
	// Returns the first validation error encountered.
	Validate(
		ctx context.Context,
		entityType pb.Entity,
		props *properties.Properties,
		projectID uuid.UUID,
	) error

	// HasValidators returns true if at least one validator is registered
	// for the given entity type.
	HasValidators(entityType pb.Entity) bool
}

type validatorEntry struct {
	id        uint64
	validator Validator
}

type validatorRegistry struct {
	mu         sync.RWMutex
	validators map[pb.Entity][]validatorEntry
	nextID     uint64
}

// NewValidatorRegistry creates a new validator registry.
func NewValidatorRegistry() ValidatorRegistry {
	return &validatorRegistry{
		validators: make(map[pb.Entity][]validatorEntry),
	}
}

// AddValidator registers a validator for a specific entity type.
func (r *validatorRegistry) AddValidator(entityType pb.Entity, validator Validator) ValidatorHandle {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	entry := validatorEntry{
		id:        r.nextID,
		validator: validator,
	}

	r.validators[entityType] = append(r.validators[entityType], entry)

	return ValidatorHandle{
		id: r.nextID,
	}
}

// RemoveValidator removes a previously registered validator.
// It searches all entity types to find and remove the validator with the given handle.
func (r *validatorRegistry) RemoveValidator(handle ValidatorHandle) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Search all entity types for this validator ID
	for entityType, entries := range r.validators {
		for i, entry := range entries {
			if entry.id == handle.id {
				// Remove by creating a new slice without this element
				r.validators[entityType] = append(entries[:i], entries[i+1:]...)
				return
			}
		}
	}
}

// Validate runs all validators registered for the given entity type.
func (r *validatorRegistry) Validate(
	ctx context.Context,
	entityType pb.Entity,
	props *properties.Properties,
	projectID uuid.UUID,
) error {
	r.mu.RLock()
	entries := r.validators[entityType]
	r.mu.RUnlock()

	// No validators = validation passes (optional validation)
	for _, entry := range entries {
		if err := entry.validator.Validate(ctx, props, projectID); err != nil {
			return err
		}
	}

	return nil
}

// HasValidators returns true if at least one validator is registered.
func (r *validatorRegistry) HasValidators(entityType pb.Entity) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.validators[entityType]) > 0
}
