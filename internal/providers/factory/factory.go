// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package factory contains logic for creating Provider instances
package factory

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

// ProviderFactory describes an interface for creating instances of a Provider
type ProviderFactory interface {
	// BuildFromID creates the provider from the Provider's UUID
	BuildFromID(ctx context.Context, providerID uuid.UUID) (v1.Provider, error)
	// BuildFromNameProject creates the provider using the provider's name and
	// project hierarchy.
	BuildFromNameProject(ctx context.Context, name string, projectID uuid.UUID) (v1.Provider, error)
}

// ProviderClassFactory describes an interface for creating instances of a
// specific Provider class. The idea is that ProviderFactory determines the
// class of the Provider, and delegates to the appropraite ProviderClassFactory
type ProviderClassFactory interface {
	// Build creates an instance of Provider based on the config in the DB
	Build(ctx context.Context, config *db.Provider) (v1.Provider, error)
	// GetSupportedClasses lists the types of Provider class which this factory
	// can produce.
	GetSupportedClasses() []db.ProviderClass
}

type providerFactory struct {
	classFactories map[db.ProviderClass]ProviderClassFactory
	store          providers.ProviderStore
}

// NewProviderFactory creates a new instance of ProviderFactory
func NewProviderFactory(
	classFactories []ProviderClassFactory,
	store providers.ProviderStore,
) (ProviderFactory, error) {
	classes := make(map[db.ProviderClass]ProviderClassFactory)
	for _, factory := range classFactories {
		supportedClasses := factory.GetSupportedClasses()
		// Sanity check: make sure we don't inadvertently register the same
		// class to two different factories, and that the factory has at least
		// one type registered
		if len(supportedClasses) == 0 {
			return nil, errors.New("provider class factory has no registered classes")
		}
		for _, class := range supportedClasses {
			_, ok := classes[class]
			if ok {
				return nil, fmt.Errorf("attempted to register class %s more than once", class)
			}
			classes[class] = factory
		}
	}
	return &providerFactory{
		classFactories: classes,
		store:          store,
	}, nil
}

func (p *providerFactory) BuildFromID(ctx context.Context, providerID uuid.UUID) (v1.Provider, error) {
	config, err := p.store.GetByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("error retrieving db record: %w", err)
	}

	return p.buildFromDBRecord(ctx, config)
}

func (p *providerFactory) BuildFromNameProject(ctx context.Context, name string, projectID uuid.UUID) (v1.Provider, error) {
	config, err := p.store.GetByName(ctx, projectID, name)
	if err != nil {
		return nil, fmt.Errorf("error retrieving db record: %w", err)
	}

	return p.buildFromDBRecord(ctx, config)
}

func (p *providerFactory) buildFromDBRecord(ctx context.Context, config *db.Provider) (v1.Provider, error) {
	class := config.Class
	factory, ok := p.classFactories[class]
	if !ok {
		return nil, fmt.Errorf("unexpected provider class: %s", class)
	}
	return factory.Build(ctx, config)
}
