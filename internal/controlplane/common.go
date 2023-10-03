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

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/mediator/internal/auth"
	"github.com/stacklok/mediator/internal/util"
)

// ProjectIDGetter is an interface that can be implemented by a request
type ProjectIDGetter interface {
	// GetProjectId returns the project ID
	GetProjectId() string
}

func providerError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return util.UserVisibleError(codes.NotFound, "provider not found")
	}
	return fmt.Errorf("provider error: %w", err)
}

func getProjectFromRequestOrDefault(ctx context.Context, in ProjectIDGetter) (uuid.UUID, error) {
	// if we do not have a group, check if we can infer it
	if in.GetProjectId() == "" {
		proj, err := auth.GetDefaultProject(ctx)
		if err != nil {
			return uuid.UUID{}, status.Errorf(codes.InvalidArgument, "cannot infer project id: %s", err)
		}

		return proj, err
	}

	parsedProjectID, err := uuid.Parse(in.GetProjectId())
	if err != nil {
		return uuid.UUID{}, util.UserVisibleError(codes.InvalidArgument, "malformed project ID")
	}
	return parsedProjectID, nil
}
