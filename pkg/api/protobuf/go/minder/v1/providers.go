// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"slices"
	"sort"
)

// providerTypeByName maps the (name) option of every defined ProviderType
// value (e.g. "github", "git") to that ProviderType. Built once at init
// time rather than on every lookup, since this is consulted in loops
// (rule type validation, rule engine construction) where re-walking the
// enum descriptor on each call would be wasteful.
var providerTypeByName map[string]ProviderType

// validProviderTraitNames is the sorted list of valid provider trait
// names, i.e. the keys of providerTypeByName. Kept separately so
// ValidProviderTraitNames can return a fresh copy without re-deriving and
// re-sorting it on every call.
var validProviderTraitNames []string

func init() {
	providerTypeByName = make(map[string]ProviderType, len(ProviderType_name))
	for v := range ProviderType_name {
		t := ProviderType(v)
		if t == ProviderType_PROVIDER_TYPE_UNSPECIFIED {
			continue
		}
		providerTypeByName[t.ToString()] = t
	}

	validProviderTraitNames = make([]string, 0, len(providerTypeByName))
	for name := range providerTypeByName {
		validProviderTraitNames = append(validProviderTraitNames, name)
	}
	sort.Strings(validProviderTraitNames)
}

// ToString returns the string representation of the ProviderType
func (provt ProviderType) ToString() string {
	return enumToStringViaDescriptor(provt.Descriptor(), provt.Number())
}

// ProviderTypeFromString returns the ProviderType whose (name) option
// matches s (e.g. "github", "git"), and true if one was found. It is the
// inverse of ProviderType.ToString.
func ProviderTypeFromString(s string) (ProviderType, bool) {
	t, ok := providerTypeByName[s]
	if !ok {
		return ProviderType_PROVIDER_TYPE_UNSPECIFIED, false
	}

	return t, true
}

// ValidProviderTraitNames returns the sorted list of valid provider trait
// names, i.e. the (name) option of every defined ProviderType value except
// PROVIDER_TYPE_UNSPECIFIED. Used to build helpful error messages when a
// rule type declares an unrecognized provider_traits entry.
//
// The returned slice is a copy; callers are free to sort or mutate it
// without affecting other callers.
func ValidProviderTraitNames() []string {
	return slices.Clone(validProviderTraitNames)
}

// ToString returns the string representation of the AuthorizationFlow
func (a AuthorizationFlow) ToString() string {
	return enumToStringViaDescriptor(a.Descriptor(), a.Number())
}

// SupportsAuthFlow returns true if the provider supports the given auth flow
func (p *Provider) SupportsAuthFlow(flow AuthorizationFlow) bool {
	return slices.Contains(p.GetAuthFlows(), flow)
}

// ToString returns the string representation of the ProviderClass
func (p ProviderClass) ToString() string {
	return enumToStringViaDescriptor(p.Descriptor(), p.Number())
}

// ToString returns the string representation of the CredentialsState
func (c CredentialsState) ToString() string {
	return enumToStringViaDescriptor(c.Descriptor(), c.Number())
}
