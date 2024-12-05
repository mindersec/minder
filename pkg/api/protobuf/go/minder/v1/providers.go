// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import "slices"

// ToString returns the string representation of the ProviderType
func (provt ProviderType) ToString() string {
	return enumToStringViaDescriptor(provt.Descriptor(), provt.Number())
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
