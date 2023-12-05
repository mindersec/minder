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
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
func EntityFromContext(ctx context.Context) *EntityContext {
	ec, ok := ctx.Value(entityContextKey).(*EntityContext)
	if !ok {
		return nil
	}
	return ec
}

// Project is a construct relevant to an entity's context.
// This is relevant for getting the full information about an entity.
type Project struct {
	ID   uuid.UUID
	Name string
}

// GetID returns the ID of the project
func (g Project) GetID() uuid.UUID {
	return g.ID
}

// GetName returns the name of the project
func (g Project) GetName() string {
	return g.Name
}

// Provider is a construct relevant to an entity's context.
// This is relevant for getting the full information about an entity.
type Provider struct {
	ID   uuid.UUID
	Name string
}

// EntityContext is the context of an entity.
// This is relevant for getting the full information about an entity.
type EntityContext struct {
	Project  Project
	Provider Provider
}

// GetProject returns the project of the entity
func (c *EntityContext) GetProject() Project {
	return c.Project
}

// GetProvider returns the provider of the entity
func (c *EntityContext) GetProvider() Provider {
	return c.Provider
}

// GetContextFromInput returns the context from the input. The
// input is the context from the gRPC request which merely holds
// user-friendly information about an object.
func GetContextFromInput(ctx context.Context, in *pb.Context, q db.Querier) (*EntityContext, error) {
	if in.Project == nil || *in.Project == "" {
		return nil, fmt.Errorf("invalid context: missing project")
	}

	project, err := q.GetProjectByName(ctx, *in.Project)
	if err != nil {
		return nil, fmt.Errorf("unable to get context: %w", err)
	}

	prov, err := q.GetProviderByName(ctx, db.GetProviderByNameParams{
		Name:      in.Provider,
		ProjectID: project.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get context: failed getting provider: %w", err)
	}

	return &EntityContext{
		Project: Project{
			ID:   project.ID,
			Name: project.Name,
		},
		Provider: Provider{
			ID:   prov.ID,
			Name: prov.Name,
		},
	}, nil
}
