// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package db contains domain-level methods for db structs.
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
	GetSelectors() []ProfileSelector
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

// GetSelectors returns the selectors
func (r ListProfilesByProjectIDAndLabelRow) GetSelectors() []ProfileSelector {
	return r.ProfilesWithSelectors
}

// LabelsFromFilter parses the filter string and populates the IncludeLabels and ExcludeLabels fields
func (lp *ListProfilesByProjectIDAndLabelParams) LabelsFromFilter(filter string) {
	// If s does not contain sep and sep is not empty, Split returns a
	// slice of length 1 whose only element is s. Work around that by
	// returning early if filter is empty.
	if filter == "" {
		return
	}

	var starMatched bool
	for _, label := range strings.Split(filter, ",") {
		switch {
		case label == "*":
			starMatched = true
		case strings.HasPrefix(label, "!"):
			// if the label starts with a "!", it is a negative filter, add it to the negative list
			lp.ExcludeLabels = append(lp.ExcludeLabels, label[1:])
		default:
			lp.IncludeLabels = append(lp.IncludeLabels, label)
		}
	}

	if starMatched {
		lp.IncludeLabels = []string{"*"}
	}
}
