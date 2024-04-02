// Copyright 2023 Stacklok, Inc.
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
