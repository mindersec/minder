// Copyright 2024 Stacklok, Inc.
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

// Package factory contains logic for creating provider instances
package factory

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers"
	ghf "github.com/stacklok/minder/internal/providers/github/factory"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

type ProviderFactory interface {
	BuildFromID(ctx context.Context, providerID uuid.UUID) (v1.Provider, error)
	BuildFromNameProject(ctx context.Context, name string, projectID uuid.UUID) (v1.Provider, error)
}

type providerFactory struct {
	ghFactory ghf.GitHubProviderFactory
	store     providers.ProviderStore
}

func NewProviderFactory(
	ghFactory ghf.GitHubProviderFactory,
	store providers.ProviderStore,
) ProviderFactory {
	return &providerFactory{
		ghFactory: ghFactory,
		store:     store,
	}
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
	var class db.ProviderClass
	if config.Class.Valid {
		class = config.Class.ProviderClass
	} else if config.Name == "github" {
		// Have seen this assumption used elsewhere in the code
		class = db.ProviderClassGithub
	} else {
		class = db.ProviderClassGithubApp
	}
	switch class {
	case db.ProviderClassGithubApp, db.ProviderClassGithub:
		return p.ghFactory.Build(ctx, config)
	}
	// fall-through
	return nil, fmt.Errorf("unexpected provider class: %s", class)
}
