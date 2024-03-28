// Copyright 2024 Stacklok, Inc
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

package providers

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/util"
)

// ProviderStore provides methods for retrieving Providers from the database
type ProviderStore interface {
	// GetByName returns the provider instance in the database as identified
	// by its project ID and name.
	GetByName(ctx context.Context, projectID uuid.UUID, name string) (*db.Provider, error)
	// GetByNameAndTrait returns the providers in the project which match the
	// specified trait.
	// Note that if error is nil, there will always be at least one element
	// in the list of providers which is returned.
	GetByNameAndTrait(
		ctx context.Context,
		projectID uuid.UUID,
		name string,
		trait db.ProviderType,
	) ([]db.Provider, error)
}

type providerStore struct {
	store db.Store
}

// NewProviderStore returns a new instance of ProviderStore.
func NewProviderStore(store db.Store) ProviderStore {
	return &providerStore{store: store}
}

func (p *providerStore) GetByName(ctx context.Context, projectID uuid.UUID, name string) (*db.Provider, error) {
	nameFilter := getNameFilterParam(name)
	providers, err := p.findProvider(ctx, nameFilter, db.NullProviderType{}, projectID)
	if err != nil {
		return nil, err
	}

	// Note that by the time we get here, `providers` will always have at
	// least one element.
	if nameFilter.Valid {
		if len(providers) == 1 {
			return &providers[0], nil
		}
		return nil, util.UserVisibleError(
			codes.InvalidArgument,
			"cannot infer provider, there are %d providers available",
			len(providers),
		)
	}

	return &providers[0], nil
}

func (p *providerStore) GetByNameAndTrait(
	ctx context.Context,
	projectID uuid.UUID,
	name string,
	trait db.ProviderType,
) ([]db.Provider, error) {
	nameFilter := getNameFilterParam(name)
	t := db.NullProviderType{ProviderType: trait, Valid: true}
	providers, err := p.findProvider(ctx, nameFilter, t, projectID)
	if err != nil {
		return nil, err
	}

	return providers, nil
}

// builds an error message based on the given filters.
func filteredResultNotFoundError(name sql.NullString, trait db.NullProviderType) error {
	msgs := []string{}
	if name.Valid {
		msgs = append(msgs, fmt.Sprintf("name: %s", name.String))
	}
	if trait.Valid {
		msgs = append(msgs, fmt.Sprintf("trait: %s", trait.ProviderType))
	}

	return util.UserVisibleError(codes.NotFound, "provider not found with filters: %s", strings.Join(msgs, ", "))
}

// findProvider is a helper function to find a provider by name and trait
func (p *providerStore) findProvider(
	ctx context.Context,
	name sql.NullString,
	trait db.NullProviderType,
	projectId uuid.UUID,
) ([]db.Provider, error) {
	// Allows us to take into account the hierarchy to find the provider
	parents, err := p.store.GetParentProjects(ctx, projectId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot retrieve parent projects: %s", err)
	}

	provs, err := p.store.FindProviders(ctx, db.FindProvidersParams{
		Projects: parents,
		Name:     name,
		Trait:    trait,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve providers: %w", err)
	}

	if len(provs) == 0 {
		return nil, filteredResultNotFoundError(name, trait)
	}

	return provs, nil
}

// getNameFilterParam allows us to build a name filter for our provider queries
func getNameFilterParam(name string) sql.NullString {
	return sql.NullString{
		String: name,
		Valid:  name != "",
	}
}

// given a list of providers, inferProvider will validate the filter and
// return the provider if it can be inferred. Note that this assumes that validation
// has already been made and that the list of providers is not empty.
func inferProvider(providers []db.Provider, nameFilter sql.NullString) (*db.Provider, error) {
	if nameFilter.Valid {
		if len(providers) == 1 {
			return &providers[0], nil
		}
		return nil, util.UserVisibleError(
			codes.InvalidArgument,
			"cannot infer provider, there are %d providers available",
			len(providers),
		)
	}

	return &providers[0], nil
}
