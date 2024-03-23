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
	"regexp"
	"strings"
)

var (
	// ErrValidationFailed is returned when a profile fails validation
	ErrValidationFailed = fmt.Errorf("validation failed")

	// Starts with a letter/digit, then a string of letter/digit and hypen or underscore
	dnsStyleNameRegex = regexp.MustCompile(`^[a-zA-Z0-9](?:[-_a-zA-Z0-9]{0,61}[a-zA-Z0-9])?$`)
	// ErrBadDNSStyleName is the error returned when a name fails the
	// `dnsStyleNameRegex` regex.
	// TODO: this is an overloaded error message - consider more fine
	// grained validation so we can provide more fine grained errors.
	ErrBadDNSStyleName = errors.New(
		"name may only contain letters, numbers, hyphens and underscores, and is limited to a maximum of 63 characters",
	)
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

// ensure GitHubAppProviderConfig implements the Validator interface
var _ Validator = (*GitHubAppProviderConfig)(nil)

// Validate is a utility function which allows for the validation of a struct.
func (_ *GitHubAppProviderConfig) Validate() error {
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

// Ensure DockerHubProviderConfig implements the Validator interface
var _ Validator = (*DockerHubProviderConfig)(nil)

// Validate is a utility function which allows for the validation of a struct.
func (d *DockerHubProviderConfig) Validate() error {
	if d.GetNamespace() == "" {
		return fmt.Errorf("namespace is required")
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

	if rt.GetName() == "" {
		return fmt.Errorf("%w: rule type name is empty", ErrInvalidRuleType)
	}

	if err := validateNamespacedName(rt.GetName()); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidRuleType, err)
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

	if p.GetName() == "" {
		return fmt.Errorf("%w: profile name cannot be empty", ErrValidationFailed)
	}

	if err := validateNamespacedName(p.GetName()); err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	// If the profile is nil or empty, we don't need to validate it
	for i, r := range p.GetRepository() {
		if err := validateRule(r); err != nil {
			return fmt.Errorf("repository rule %d is invalid: %w", i, err)
		}
	}

	for i, b := range p.GetBuildEnvironment() {
		if err := validateRule(b); err != nil {
			return fmt.Errorf("build environment rule %d is invalid: %w", i, err)
		}
	}

	for i, a := range p.GetArtifact() {
		if err := validateRule(a); err != nil {
			return fmt.Errorf("artifact rule %d is invalid: %w", i, err)
		}
	}

	for i, pr := range p.GetPullRequest() {
		if err := validateRule(pr); err != nil {
			return fmt.Errorf("pull request rule %d is invalid: %w", i, err)
		}
	}

	return nil
}

func validateRule(r *Profile_Rule) error {
	if r.GetType() == "" {
		return fmt.Errorf("%w: rule type cannot be empty", ErrValidationFailed)
	}

	// TODO: can we omit this if the rule doesn't have values?
	if r.GetDef() == nil {
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

func validateNamespacedName(name string) error {
	components := strings.Split(name, "/")
	if len(components) > 2 {
		return errors.New("cannot have more than one slash in name")
	}
	// if this is a namespaced name, validate both the namespace and the name
	for _, component := range components {
		if !dnsStyleNameRegex.MatchString(component) {
			return ErrBadDNSStyleName
		}
	}
	return nil
}
