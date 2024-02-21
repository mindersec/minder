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

import (
	"errors"
	"fmt"
)

var (
	// ErrValidationFailed is returned when a profile fails validation
	ErrValidationFailed = fmt.Errorf("validation failed")
)

// Validator is an interface which allows for the validation of a struct.
type Validator interface {
	Validate() error
}

// ensure GitHubProviderConfig implements the Validator interface
var _ Validator = (*GitHubProviderConfig)(nil)

// Validate is a utility function which allows for the validation of a struct.
func (_ *GitHubProviderConfig) Validate() error {
	// Unfortunately, we don't currently have a way to add custom tags to
	// protobuf-generated structs, so we have to do this manually.
	return nil
}

// ensure RESTProviderConfig implements the Validator interface
var _ Validator = (*RESTProviderConfig)(nil)

// Validate is a utility function which allows for the validation of a struct.
func (rpcfg *RESTProviderConfig) Validate() error {
	// Unfortunately, we don't currently have a way to add custom tags to
	// protobuf-generated structs, so we have to do this manually.
	if rpcfg.GetBaseUrl() == "" {
		return fmt.Errorf("base_url is required")
	}

	return nil
}

// Ensure Entity implements the Validator interface
var _ Validator = (*Entity)(nil)

var (
	// ErrInvalidEntity is returned when an entity is invalid
	ErrInvalidEntity = errors.New("invalid entity")
)

// Validate ensures that an entity is valid
func (entity *Entity) Validate() error {
	if !entity.IsValid() {
		return fmt.Errorf("%w: invalid entity type: %s", ErrInvalidEntity, entity.String())
	}

	return nil
}

var (
	// ErrInvalidRuleType is returned when a rule type is invalid
	ErrInvalidRuleType = errors.New("invalid rule type")
	// ErrInvalidRuleTypeDefinition is returned when a rule type definition is invalid
	ErrInvalidRuleTypeDefinition = errors.New("invalid rule type definition")
)

// Ensure RuleType implements the Validator interface
var _ Validator = (*RuleType)(nil)

// Validate ensures that a rule type is valid
func (rt *RuleType) Validate() error {
	if rt == nil {
		return fmt.Errorf("%w: rule type is nil", ErrInvalidRuleType)
	}

	if rt.Def == nil {
		return fmt.Errorf("%w: rule type definition is nil", ErrInvalidRuleType)
	}

	if err := rt.Def.Validate(); err != nil {
		return errors.Join(ErrInvalidRuleType, err)
	}

	return nil
}

// Validate validates a rule type definition
func (def *RuleType_Definition) Validate() error {
	if !EntityFromString(def.InEntity).IsValid() {
		return fmt.Errorf("%w: invalid entity type: %s", ErrInvalidRuleTypeDefinition, def.InEntity)
	}

	if def.RuleSchema == nil {
		return fmt.Errorf("%w: rule schema is nil", ErrInvalidRuleTypeDefinition)
	}

	if def.Ingest == nil {
		return fmt.Errorf("%w: data ingest is nil", ErrInvalidRuleTypeDefinition)
	} else if err := def.Ingest.Validate(); err != nil {
		return err
	}

	if def.Eval == nil {
		return fmt.Errorf("%w: data eval is nil", ErrInvalidRuleTypeDefinition)
	}

	return nil
}

// Validate validates a rule type definition ingest
func (ing *RuleType_Definition_Ingest) Validate() error {
	if ing == nil {
		return fmt.Errorf("%w: ingest is nil", ErrInvalidRuleTypeDefinition)
	}

	if ing.Type == IngestTypeDiff {
		if ing.GetDiff() == nil {
			return fmt.Errorf("%w: diff ingest is nil", ErrInvalidRuleTypeDefinition)
		} else if err := ing.GetDiff().Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates a rule type definition ingest diff
func (diffing *DiffType) Validate() error {
	if diffing == nil {
		return fmt.Errorf("%w: diffing is nil", ErrInvalidRuleTypeDefinition)
	}

	switch diffing.GetType() {
	case "", DiffTypeDep, DiffTypeFull:
		return nil
	default:
		return fmt.Errorf("%w: diffing type is invalid: %s", ErrInvalidRuleTypeDefinition, diffing.GetType())
	}
}

func (p *Profile) getTypeWithDefault() string {
	pt := p.GetType()
	if pt == "" {
		return ProfileType
	}
	return pt
}

func (p *Profile) getVersionWithDefault() string {
	pv := p.GetVersion()
	if pv == "" {
		return ProfileTypeVersion
	}
	return pv
}

// Validate validates a pipeline profile
func (p *Profile) Validate() error {
	if p.getTypeWithDefault() != ProfileType {
		return fmt.Errorf("%w: profile type is invalid: %s. Did you parse the wrong file?",
			ErrValidationFailed, p.Type)
	}
	if p.getVersionWithDefault() != ProfileTypeVersion {
		return fmt.Errorf("%w: profile version is invalid: %s", ErrValidationFailed, p.Version)
	}

	if p.Name == "" {
		return fmt.Errorf("%w: profile name cannot be empty", ErrValidationFailed)
	}

	repoRuleCount := len(p.Repository)
	buildEnvRuleCount := len(p.BuildEnvironment)
	artifactRuleCount := len(p.Artifact)
	pullRequestRuleCount := len(p.PullRequest)
	totalRuleCount := repoRuleCount + buildEnvRuleCount + artifactRuleCount + pullRequestRuleCount

	if totalRuleCount == 0 {
		return fmt.Errorf("%w: profile must have at least one rule", ErrValidationFailed)
	}

	// If the profile is nil or empty, we don't need to validate it
	if p.Repository != nil && repoRuleCount > 0 {
		return validateEntity(p.Repository)
	}

	if p.BuildEnvironment != nil && buildEnvRuleCount > 0 {
		return validateEntity(p.BuildEnvironment)
	}

	if p.Artifact != nil && artifactRuleCount > 0 {
		return validateEntity(p.Artifact)
	}

	if p.PullRequest != nil && pullRequestRuleCount > 0 {
		return validateEntity(p.PullRequest)
	}

	return nil
}

func validateEntity(e []*Profile_Rule) error {
	if len(e) == 0 {
		return fmt.Errorf("%w: entity rules cannot be empty", ErrValidationFailed)
	}

	for _, r := range e {
		if r == nil {
			return fmt.Errorf("%w: entity contextual rules cannot be nil", ErrValidationFailed)
		}

		if err := validateRule(r); err != nil {
			return err
		}
	}

	return nil
}

func validateRule(r *Profile_Rule) error {
	if r.Type == "" {
		return fmt.Errorf("%w: rule type cannot be empty", ErrValidationFailed)
	}

	if r.Def == nil {
		return fmt.Errorf("%w: rule def cannot be nil", ErrValidationFailed)
	}

	return nil
}

var _ Validator = (*RuleType_Definition_Remediate_PullRequestRemediation)(nil)

// Validate validates a rule definition
func (prRem *RuleType_Definition_Remediate_PullRequestRemediation) Validate() error {
	if prRem == nil {
		return errors.New("pull request remediation is nil")
	}

	if prRem.Title == "" {
		return errors.New("title is required")
	}

	if prRem.Body == "" {
		return errors.New("body is required")
	}

	return nil
}
