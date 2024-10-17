// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
