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

package db

import (
	"slices"
	"strings"

	"github.com/sqlc-dev/pqtype"
)

// This file contains domain-level methods for db structs

// CanImplement returns true if the provider implements the given type.
func (p *Provider) CanImplement(impl ProviderType) bool {
	return slices.Contains(p.Implements, impl)
}

// ProfileRow is an interface row in the profiles table
type ProfileRow interface {
	GetProfile() Profile
	GetEntityProfile() NullEntities
	GetContextualRules() pqtype.NullRawMessage
}

// GetProfile returns the profile
func (r ListProfilesByProjectIDAndLabelRow) GetProfile() Profile {
	return r.Profile
}

// GetEntityProfile returns the entity profile
func (r ListProfilesByProjectIDAndLabelRow) GetEntityProfile() NullEntities {
	return r.ProfilesWithEntityProfile.Entity
}

// GetContextualRules returns the contextual rules
func (r ListProfilesByProjectIDAndLabelRow) GetContextualRules() pqtype.NullRawMessage {
	return r.ProfilesWithEntityProfile.ContextualRules
}

// GetProfile returns the profile
func (r ListProfilesByProjectIDRow) GetProfile() Profile {
	return r.Profile
}

// GetEntityProfile returns the entity profile
func (r ListProfilesByProjectIDRow) GetEntityProfile() NullEntities {
	return r.ProfilesWithEntityProfile.Entity
}

// GetContextualRules returns the contextual rules
func (r ListProfilesByProjectIDRow) GetContextualRules() pqtype.NullRawMessage {
	return r.ProfilesWithEntityProfile.ContextualRules
}

// LabelsFromFilter parses the filter string and populates the IncludeLabels and ExcludeLabels fields
func (lp *ListProfilesByProjectIDAndLabelParams) LabelsFromFilter(filter string) {
	// otherwise Split would have returned a slice with one empty string
	if filter == "" {
		return
	}

	for _, label := range strings.Split(filter, ",") {
		switch {
		case label == "*":
			lp.IncludeLabels = append(lp.IncludeLabels, label)
		case strings.HasPrefix(label, "!"):
			// if the label starts with a "!", it is a negative filter, add it to the negative list
			lp.ExcludeLabels = append(lp.ExcludeLabels, label[1:])
		default:
			lp.IncludeLabels = append(lp.IncludeLabels, label)
		}
	}
}
