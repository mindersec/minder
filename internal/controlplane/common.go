//
// Copyright 2023 Stacklok, Inc.
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

package controlplane

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// HasProtoContext is an interface that can be implemented by a request
type HasProtoContext interface {
	GetContext() *pb.Context
}

// providerError wraps an error with a user visible error message
func providerError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return util.UserVisibleError(codes.NotFound, "provider not found")
	}
	return fmt.Errorf("provider error: %w", err)
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

func getProviderFromRequestOrDefault(
	ctx context.Context,
	store db.Store,
	in HasProtoContext,
	projectId uuid.UUID,
) (db.Provider, error) {
	name := getNameFilterParam(in.GetContext())
	providers, err := findProvider(ctx, name, db.NullProviderType{}, projectId, store)
	if err != nil {
		return db.Provider{}, err
	}

	p, err := inferProvider(providers, name)
	if err != nil {
		return db.Provider{}, err
	}

	return p, nil
}

func getProvidersByTrait(
	ctx context.Context,
	store db.Store,
	in HasProtoContext,
	projectId uuid.UUID,
	trait db.ProviderType,
) ([]db.Provider, error) {
	name := getNameFilterParam(in.GetContext())
	t := db.NullProviderType{ProviderType: trait, Valid: true}
	providers, err := findProvider(ctx, name, t, projectId, store)
	if err != nil {
		return nil, err
	}

	return providers, nil
}

// findProvider is a helper function to find a provider by name and trait
func findProvider(
	ctx context.Context,
	name sql.NullString,
	trait db.NullProviderType,
	projectId uuid.UUID,
	store db.Store,
) ([]db.Provider, error) {
	// Allows us to take into account the hierarchy to find the provider
	parents, err := store.GetParentProjects(ctx, projectId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot retrieve parent projects: %s", err)
	}

	provs, err := store.FindProviders(ctx, db.FindProvidersParams{
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
func getNameFilterParam(in *pb.Context) sql.NullString {
	if in.GetProvider() == "" {
		return sql.NullString{}
	}
	return sql.NullString{
		String: in.GetProvider(),
		Valid:  true,
	}
}

// given a list of providers, inferProvider will validate the filter and
// return the provider if it can be inferred. Note that this assumes that validation
// has already been made and that the list of providers is not empty.
func inferProvider(providers []db.Provider, nameFilter sql.NullString) (db.Provider, error) {
	if !nameFilter.Valid {
		if len(providers) == 1 {
			return providers[0], nil
		}
		return db.Provider{}, util.UserVisibleError(codes.InvalidArgument, "cannot infer provider, there are %d providers available",
			len(providers))
	}

	return providers[0], nil
}
