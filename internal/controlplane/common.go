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
	"github.com/stacklok/minder/internal/engine"
	"slices"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/auth"
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

func getProjectFromRequestOrDefault(ctx context.Context, in HasProtoContext) (uuid.UUID, error) {
	var requestedProject string

	// Prefer the context message from the protobuf
	if in.GetContext().GetProject() != "" {
		requestedProject = in.GetContext().GetProject()
	} else {
		proj, err := auth.GetDefaultProject(ctx)
		if err != nil {
			return uuid.UUID{}, status.Errorf(codes.InvalidArgument, "cannot infer project id: %s", err)
		}

		return proj, err
	}

	parsedProjectID, err := uuid.Parse(requestedProject)
	if err != nil {
		return uuid.UUID{}, util.UserVisibleError(codes.InvalidArgument, "malformed project ID")
	}
	return parsedProjectID, nil
}

func getProviderFromRequestOrDefault(
	ctx context.Context,
	store db.Store,
	in HasProtoContext,
	projectId uuid.UUID,
) (db.Provider, error) {
	providers, err := store.ListProvidersByProjectID(ctx, projectId)
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

	matchesName := func(provider db.Provider) bool {
		return provider.Name == in.GetContext().GetProvider()
	}

	i := slices.IndexFunc(providers, matchesName)
	if i == -1 {
		return db.Provider{}, util.UserVisibleError(codes.InvalidArgument, "invalid provider name: %s", in.GetContext().GetProvider())
	}
	return providers[i], nil
}

// contextValidation is a helper function to initialize entity context info and validate input
// It also sets up the needed information in the `in` entity context that's needed for the rest of the flow
func (s *Server) contextValidation(ctx context.Context, inout *pb.Context) (context.Context, error) {
	if inout == nil {
		return ctx, fmt.Errorf("context cannot be nil")
	}

	if err := ensureDefaultProjectForContext(ctx, inout); err != nil {
		return ctx, err
	}

	entityCtx, err := engine.GetContextFromInput(ctx, inout, s.store)
	if err != nil {
		return ctx, fmt.Errorf("cannot get context from input: %v", err)
	}

	return engine.WithEntityContext(ctx, entityCtx), nil
}

// ensureDefaultProjectForContext ensures a valid project is set in the context or sets the default project
// if the project is not set in the incoming entity context, it'll set it.
func ensureDefaultProjectForContext(ctx context.Context, inout *pb.Context) error {
	// Project is already set
	if inout.GetProject() != "" {
		return nil
	}

	gid, err := auth.GetDefaultProject(ctx)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "cannot infer project id")
	}

	project := gid.String()
	inout.Project = &project
	return nil
}
