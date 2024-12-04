// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/itchyny/gojq"
	"github.com/open-policy-agent/opa/ast"

	"github.com/mindersec/minder/internal/util"
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

var (
	validate = validator.New(validator.WithRequiredStructEnabled())
)

// Validator is an interface which allows for the validation of a struct.
type Validator interface {
	Validate() error
}

// ensure ProviderConfig implements the Validator interface
var _ Validator = (*ProviderConfig)(nil)

// Validate is a utility function which allows for the validation of the ProviderConfig struct.
func (p *ProviderConfig) Validate() error {
	if err := p.GetAutoRegistration().Validate(); err != nil {
		return fmt.Errorf("auto_registration: %w", err)
	}
	return nil
}

// ensure AutoRegistration implements the Validator interface
var _ Validator = (*AutoRegistration)(nil)

// Validate is a utility function which allows for the validation of the AutoRegistration struct.
func (a *AutoRegistration) Validate() error {
	for entity := range a.GetEntities() {
		if !EntityFromString(entity).IsValid() {
			return fmt.Errorf("invalid entity type: %s", entity)
		}
	}

	return nil
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

	if err := rt.Def.Validate(); err != nil {
		return errors.Join(ErrInvalidRuleType, err)
	}

	return nil
}

// Validate validates a rule type definition
func (def *RuleType_Definition) Validate() error {
	if def == nil {
		return fmt.Errorf("%w: rule type definition is nil", ErrInvalidRuleTypeDefinition)
	}

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

	// Alert is not required and can be nil
	if def.Alert != nil {
		if err := def.Alert.Validate(); err != nil {
			return err
		}
	}

	return def.Eval.Validate()
}

// Validate validates a rule type definition eval
func (ev *RuleType_Definition_Eval) Validate() error {
	if ev == nil {
		return fmt.Errorf("%w: eval is nil", ErrInvalidRuleTypeDefinition)
	}

	// Not using import to avoid circular dependency
	if ev.Type == "rego" {
		if err := ev.GetRego().Validate(); err != nil {
			return err
		}
	} else if ev.Type == "jq" {
		if len(ev.GetJq()) == 0 {
			return fmt.Errorf("%w: jq definition is empty", ErrInvalidRuleTypeDefinition)
		}

		for i, jq := range ev.GetJq() {
			if err := jq.Validate(); err != nil {
				return fmt.Errorf("jq rule %d is invalid: %w", i, err)
			}
		}
	}
	return nil
}

// Validate validates a rule type definition eval rego
func (rego *RuleType_Definition_Eval_Rego) Validate() error {
	if rego == nil {
		return fmt.Errorf("%w: rego is nil", ErrInvalidRuleTypeDefinition)
	}

	if rego.Def == "" {
		return fmt.Errorf("%w: rego definition is empty", ErrInvalidRuleTypeDefinition)
	}

	_, err := ast.ParseModule("minder-ruletype-def.rego", rego.Def)
	if err != nil {
		return fmt.Errorf("%w: rego definition is invalid: %s", ErrInvalidRuleTypeDefinition, err)
	}

	return nil
}

// Validate validates a rule type definition eval jq
func (jq *RuleType_Definition_Eval_JQComparison) Validate() error {
	if jq == nil {
		return fmt.Errorf("%w: jq is nil", ErrInvalidRuleTypeDefinition)
	}

	if err := jq.GetIngested().Validate(); err != nil {
		return fmt.Errorf("%w: jq ingested definition is invalid: %w", ErrInvalidRuleTypeDefinition, err)
	}

	if jq.GetProfile() != nil && jq.GetConstant() != nil {
		return fmt.Errorf("%w: jq profile and constant accessors are mutually exclusive", ErrInvalidRuleTypeDefinition)
	} else if jq.GetProfile() == nil && jq.GetConstant() == nil {
		return fmt.Errorf("%w: jq missing profile or constant accessor", ErrInvalidRuleTypeDefinition)
	}

	if jq.GetProfile() != nil {
		if err := jq.GetProfile().Validate(); err != nil {
			return fmt.Errorf("%w: jq profile accessor is invalid: %w", ErrInvalidRuleTypeDefinition, err)
		}
	}

	return nil
}

// Validate validates a rule type definition eval jq operator
func (op *RuleType_Definition_Eval_JQComparison_Operator) Validate() error {
	if op == nil {
		return fmt.Errorf("%w: operator is nil", ErrInvalidRuleTypeDefinition)
	}

	if op.GetDef() == "" {
		return fmt.Errorf("%w: definition is empty", ErrInvalidRuleTypeDefinition)
	}

	q, err := gojq.Parse(op.GetDef())
	if err != nil {
		return fmt.Errorf("%w: definition is not parsable: %w", ErrInvalidRuleTypeDefinition, err)
	}
	if _, err = gojq.Compile(q); err != nil {
		return fmt.Errorf("%w: definition is invalid: %w", ErrInvalidRuleTypeDefinition, err)
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
	} else if ing.Type == "rest" {
		if err := ing.GetRest().Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates a rule type definition alert
func (alert *RuleType_Definition_Alert) Validate() error {
	if alert == nil {
		return nil
	}

	// Not using import to avoid circular dependency
	if alert.Type == "security_advisory" {
		if err := alert.GetSecurityAdvisory().Validate(); err != nil {
			return err
		}
	} else if alert.Type == "pull_request_comment" {
		if err := alert.GetPullRequestComment().Validate(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("%w: alert type cannot be empty", ErrInvalidRuleTypeDefinition)
	}
	return nil
}

// Validate validates a rule type alert security advisory
func (sa *RuleType_Definition_Alert_AlertTypeSA) Validate() error {
	if sa == nil {
		return fmt.Errorf("%w: security advisory is nil", ErrInvalidRuleTypeDefinition)
	}

	return nil
}

// Validate validates a rule type alert pull request comment
func (comment *RuleType_Definition_Alert_AlertTypePRComment) Validate() error {
	if comment == nil {
		return fmt.Errorf("%w: pull request comment is nil", ErrInvalidRuleTypeDefinition)
	}

	if comment.GetReviewMessage() == "" {
		return fmt.Errorf("%w: pull request comment review message cannot be empty", ErrInvalidRuleTypeDefinition)
	}

	return nil
}

// Validate validates a rule type definition remediate
func (rem *RuleType_Definition_Remediate) Validate() error {
	if rem == nil {
		return nil
	}

	// Not using import to avoid circular dependency
	if rem.Type == "rest" {
		if err := rem.GetRest().Validate(); err != nil {
			return err
		}
	} else if rem.Type == "pull_request" {
		if err := rem.GetPullRequest().Validate(); err != nil {
			return err
		}
	} else if rem.Type == "gh_branch_protection" {
		if err := rem.GetGhBranchProtection().Validate(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("%w: remediate type cannot be empty", ErrInvalidRuleTypeDefinition)
	}
	return nil
}

// Validate validates a rest remediation
func (rest *RestType) Validate() error {
	if rest == nil {
		return fmt.Errorf("%w: rest remediation is nil", ErrInvalidRuleTypeDefinition)
	}

	if rest.Endpoint == "" {
		return fmt.Errorf("%w: rest endpoint cannot be empty", ErrInvalidRuleTypeDefinition)
	}

	return nil
}

// Validate validates a GitHub branch protection remediation
func (ghp *RuleType_Definition_Remediate_GhBranchProtectionType) Validate() error {
	if ghp == nil {
		return fmt.Errorf("%w: github branch protection remediation is nil", ErrInvalidRuleTypeDefinition)
	}

	_, err := util.NewSafeTextTemplate(&ghp.Patch, "patch")
	if err != nil {
		return fmt.Errorf("%w: patch template is not parsable: %w", ErrInvalidRuleTypeDefinition, err)
	}

	return nil
}

// Validate validates a pull request remediation
func (prRem *RuleType_Definition_Remediate_PullRequestRemediation) Validate() error {
	if prRem == nil {
		return fmt.Errorf("%w: pull request remediation is nil", ErrInvalidRuleTypeDefinition)
	}

	if prRem.Title == "" {
		return fmt.Errorf("%w: pull request title cannot be empty", ErrInvalidRuleTypeDefinition)
	}

	if prRem.Body == "" {
		return fmt.Errorf("%w: pull request body cannot be empty", ErrInvalidRuleTypeDefinition)
	}

	_, err := util.NewSafeHTMLTemplate(&prRem.Title, "title")
	if err != nil {
		return fmt.Errorf("%w: pull request title is not parsable: %w", ErrInvalidRuleTypeDefinition, err)
	}

	_, err = util.NewSafeHTMLTemplate(&prRem.Body, "body")
	if err != nil {
		return fmt.Errorf("%w: pull request body is not parsable: %w", ErrInvalidRuleTypeDefinition, err)
	}

	return nil
}

// Validate validates a rule type definition ingest diff
func (diffing *DiffType) Validate() error {
	if diffing == nil {
		return fmt.Errorf("%w: diffing is nil", ErrInvalidRuleTypeDefinition)
	}

	switch diffing.GetType() {
	case "", DiffTypeDep, DiffTypeNewDeps, DiffTypeFull:
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

// Validate validates data sources
func (ds *DataSource) Validate() error {
	if ds == nil {
		return fmt.Errorf("%w: data source is nil", ErrValidationFailed)
	}

	if ds.GetName() == "" {
		return fmt.Errorf("%w: data source name cannot be empty", ErrValidationFailed)
	}

	if ds.GetDriver() == nil {
		return fmt.Errorf("%w: data source driver cannot be nil", ErrValidationFailed)
	}

	// All data source drivers must include validation
	val, ok := ds.GetDriver().(Validator)
	if !ok {
		return fmt.Errorf("%w: data source driver is not a valid driver", ErrValidationFailed)
	}

	return val.Validate()
}

// Validate is the entrypoint for the actual driver's validation
func (dsRestDriver *DataSource_Rest) Validate() error {
	if dsRestDriver == nil {
		return fmt.Errorf("%w: rest driver is nil", ErrValidationFailed)
	}

	if dsRestDriver.Rest == nil {
		return fmt.Errorf("%w: rest driver is nil", ErrValidationFailed)
	}

	return dsRestDriver.Rest.Validate()
}

// Validate validates a rest data source
func (rest *RestDataSource) Validate() error {
	if rest == nil {
		return fmt.Errorf("%w: rest data source is nil", ErrValidationFailed)
	}

	if len(rest.GetDef()) == 0 {
		return fmt.Errorf("%w: rest definition is empty", ErrValidationFailed)
	}

	var errs []error
	for i, def := range rest.GetDef() {
		if i == "" {
			errs = append(errs, fmt.Errorf("rest function name %s is empty", i))
		}

		// TODO: Should we validate valid characters here? We already do that
		// in the protobuf definition.
		if err := def.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("rest function %s is invalid: %w", i, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Validate validates a rest function
func (rest *RestDataSource_Def) Validate() error {
	if rest == nil {
		return fmt.Errorf("%w: rest function is nil", ErrValidationFailed)
	}

	if rest.GetEndpoint() == "" {
		return fmt.Errorf("%w: rest function endpoint is empty", ErrValidationFailed)
	}

	if rest.GetInputSchema() == nil {
		return fmt.Errorf("%w: rest function input schema is nil", ErrValidationFailed)
	}

	if rest.GetBody() != nil {
		switch rest.GetBody().(type) {
		case *RestDataSource_Def_Bodyobj:
			if rest.GetBodyobj() == nil {
				return fmt.Errorf("%w: rest function body is nil", ErrValidationFailed)
			}
		case *RestDataSource_Def_BodyFromField:
			if rest.GetBodyFromField() == "" {
				return fmt.Errorf("%w: rest function body from field is empty", ErrValidationFailed)
			}
			if err := keyInProperties(rest.GetBodyFromField(), rest.GetInputSchema().AsMap()); err != nil {
				return fmt.Errorf("%w: %v", ErrValidationFailed, err)
			}
		case *RestDataSource_Def_Bodystr:
			if rest.GetBodystr() == "" {
				return fmt.Errorf("%w: rest function body from input is empty", ErrValidationFailed)
			}
		}
	}

	return nil
}

// validate that the given key exists in the given properties.
// they key must exist in the top level properties. It must contain a default
// or be marked as required.
func keyInProperties(key string, schema map[string]any) error {
	if schema == nil {
		return fmt.Errorf("properties are missing")
	}

	// check required
	required, ok := schema["required"]
	if ok {
		req, ok := required.([]any)
		if !ok {
			return fmt.Errorf("required is invalid")
		}
		for _, r := range req {
			if r == key {
				return nil
			}
		}
	}

	props, ok := schema["properties"]
	if !ok {
		return fmt.Errorf("properties are missing")
	}

	properties, ok := props.(map[string]any)
	if !ok {
		return fmt.Errorf("properties are invalid")
	}

	prop, ok := properties[key]
	if !ok {
		return fmt.Errorf("key %q is missing", key)
	}

	p, ok := prop.(map[string]any)
	if !ok {
		return fmt.Errorf("key %q is invalid", key)
	}

	if _, ok := p["default"]; !ok {
		if _, ok := p["required"]; !ok {
			return fmt.Errorf("key %q is missing default or required", key)
		}
	}

	return nil
}
