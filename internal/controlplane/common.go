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
	"slices"

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

func providerError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return util.UserVisibleError(codes.NotFound, "provider not found")
	}
	return fmt.Errorf("provider error: %w", err)
}

func getProviderFromRequestOrDefault(
	ctx context.Context,
	store db.Store,
	in HasProtoContext,
	projectId uuid.UUID,
) (db.Provider, error) {
	// Allows us to take into account the hierarchy to find the provider
	parents, err := store.GetParentProjects(ctx, projectId)
	if err != nil {
		return db.Provider{}, status.Errorf(codes.InvalidArgument, "cannot retrieve parent projects: %s", err)
	}
	providers, err := store.ListProvidersByProjectID(ctx, parents)
	if err != nil {
		return db.Provider{}, status.Errorf(codes.InvalidArgument, "cannot retrieve providers: %s", err)
	}
	// if we do not have a provider name, check if we can infer it
	if in.GetContext().GetProvider() == "" {
		if len(providers) == 1 {
			return providers[0], nil
		}
		return db.Provider{}, util.UserVisibleError(codes.InvalidArgument, "cannot infer provider, there are %d providers available",
			len(providers))
	}

	return findProvider(in.GetContext().GetProvider(), providers)
}

func listProvidersOrInferDefault(
	ctx context.Context,
	store db.Store,
	in HasProtoContext,
	projectId uuid.UUID,
) ([]db.Provider, error) {
	// Allows us to take into account the hierarchy to find the provider
	parents, err := store.GetParentProjects(ctx, projectId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot retrieve parent projects: %s", err)
	}
	providers, err := store.ListProvidersByProjectID(ctx, parents)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cannot retrieve providers: %s", err)
	}
	// if we do not have a provider name, check if we can infer it
	if in.GetContext().GetProvider() == "" {
		return providers, nil
	}

	prov, err := findProvider(in.GetContext().GetProvider(), providers)
	if err != nil {
		return nil, err
	}

	return []db.Provider{prov}, nil
}

func findProvider(name string, provs []db.Provider) (db.Provider, error) {
	matchesName := func(provider db.Provider) bool {
		return provider.Name == name
	}

	i := slices.IndexFunc(provs, matchesName)
	if i == -1 {
		return db.Provider{}, util.UserVisibleError(codes.InvalidArgument, "invalid provider name: %s", name)
	}
	return provs[i], nil
}
