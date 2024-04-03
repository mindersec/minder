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

// ApplyDefaults applies default values to the Profile.
func (p *Profile) ApplyDefaults() {
	if p == nil {
		return
	}

	p.defaultDisplayName()

	if p.IsEmpty() {
		p.addEmptyProfileRule()
	}
}

func (p *Profile) defaultDisplayName() {
	displayName := p.GetDisplayName()
	// if empty use the name
	if displayName == "" {
		p.DisplayName = p.GetName()
	}
}

func (p *Profile) addEmptyProfileRule() {
	// this is a bit of a hack. When we store profiles, we store just the profile metadata
	// in the profiles table and then separately the rules per entity in the entity_profiles table.
	// Most of our SQL queries are written to expect that there is at least one rule per profile
	// by JOIN-ing the tables. A LEFT JOIN would be impractical because then all results would have
	// to handle a NULL type for the case where there are no rules and the callers would have to handle
	// the NULL values as well. So we add an empty rule here that does nothing, is not retrieved in the
	// profile get commands and because this rule is never evaluated, the profile status is pending
	// until an actual rule is added and evaluated.
	p.Repository = []*Profile_Rule{}
}

// IsEmpty returns true if the Profile has no rules.
func (p *Profile) IsEmpty() bool {
	repoRuleCount := len(p.GetRepository())
	buildEnvRuleCount := len(p.GetBuildEnvironment())
	artifactRuleCount := len(p.GetArtifact())
	pullRequestRuleCount := len(p.GetPullRequest())
	totalRuleCount := repoRuleCount + buildEnvRuleCount + artifactRuleCount + pullRequestRuleCount

	return totalRuleCount == 0
}
