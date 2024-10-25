// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package engcontext defines the EngineContext type.
package engcontext

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/mindersec/minder/internal/providers"
	"github.com/mindersec/minder/pkg/db"
)

type key int

const (
	// EntityContextKey is the key used to store the entity context in the golang Context
	// object for a given API call.
	entityContextKey key = iota
)

// WithEntityContext stores an EntityContext in the current context.
func WithEntityContext(ctx context.Context, c *EntityContext) context.Context {
	return context.WithValue(ctx, entityContextKey, c)
}

// EntityFromContext extracts the current EntityContext, WHICH MAY BE NIL!
func EntityFromContext(ctx context.Context) EntityContext {
	ec, _ := ctx.Value(entityContextKey).(*EntityContext)
	if ec == nil {
		return EntityContext{}
	}
	return *ec
}

// Project is a construct relevant to an entity's context.
// This is relevant for getting the full information about an entity.
type Project struct {
	ID uuid.UUID
}

// Provider is a construct relevant to an entity's context.
// This is relevant for getting the full information about an entity.
type Provider struct {
	Name string
}

// EntityContext is the context of an entity.
// This is relevant for getting the full information about an entity.
type EntityContext struct {
	Project  Project
	Provider Provider
}

// Validate validates that the entity context contains values that are present in the DB
func (c *EntityContext) Validate(ctx context.Context, q db.Querier, providerStore providers.ProviderStore) error {
	_, err := q.GetProjectByID(ctx, c.Project.ID)
	if err != nil {
		return fmt.Errorf("unable to get context: failed getting project: %w", err)
	}

	_, err = providerStore.GetByName(ctx, c.Project.ID, c.Provider.Name)
	if err != nil {
		return fmt.Errorf("unable to get context: failed getting provider: %w", err)
	}

	return nil
}

// ValidateProject validates that the entity context contains a project that is present in the DB
func (c *EntityContext) ValidateProject(ctx context.Context, q db.Querier) error {
	_, err := q.GetProjectByID(ctx, c.Project.ID)
	if err != nil {
		return fmt.Errorf("unable to get context: failed getting project: %w", err)
	}

	return nil
}
