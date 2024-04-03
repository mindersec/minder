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

const (
	// ProfileType is the type of the profile resource.
	ProfileType = "profile"
	// ProfileTypeVersion is the version of the profile resource.
	ProfileTypeVersion = "v1"
)

// GetContext returns the context from the nested Profile
func (r *CreateProfileRequest) GetContext() *Context {
	if r != nil && r.Profile != nil {
		return r.Profile.Context
	}
	return nil
}

// GetContext returns the context from the nested Profile
func (r *UpdateProfileRequest) GetContext() *Context {
	if r != nil && r.Profile != nil {
		return r.Profile.Context
	}
	return nil
}

func (p *Profile) ApplyDefaults() {
	if p == nil {
		return
	}

	p.defaultDisplayName()
}

func (p *Profile) defaultDisplayName() {
	displayName := p.GetDisplayName()
	// if empty use the name
	if displayName == "" {
		p.DisplayName = p.GetName()
	}
}
