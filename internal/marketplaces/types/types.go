// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package types contains domain models for marketplaces
package types

import (
	"github.com/google/uuid"

	"github.com/stacklok/minder/internal/db"
)

// ProjectContext contains the information needed to create rule types and
// profiles in a project.
// This may be useful in various parts of the codebase outside of marketplaces.
type ProjectContext struct {
	// Project ID
	ID uuid.UUID
	// Provider which profiles/rule types will be linked to
	Provider *db.Provider
}

// NewProjectContext is a convenience function for creating a ProjectContext
func NewProjectContext(id uuid.UUID, provider *db.Provider) ProjectContext {
	return ProjectContext{
		ID:       id,
		Provider: provider,
	}
}
