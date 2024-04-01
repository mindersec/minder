// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package engine

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
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
func (c *EntityContext) Validate(ctx context.Context, q db.Querier) error {
	_, err := q.GetProjectByID(ctx, c.Project.ID)
	if err != nil {
		return fmt.Errorf("unable to get context: failed getting project: %w", err)
	}

	ph, err := q.GetParentProjects(ctx, c.Project.ID)
	if err != nil {
		return fmt.Errorf("unable to get context: failed getting project hierarchy: %w", err)
	}

	_, err = q.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:     c.Provider.Name,
		Projects: ph,
	})
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
