// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.role/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// Package rule provides the CLI subcommand for managing rules

package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/stacklok/mediator/internal/util"
	"github.com/stacklok/mediator/pkg/db"
	"github.com/stacklok/mediator/pkg/entities"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

var (
	// ErrValidationFailed is returned when a policy fails validation
	ErrValidationFailed = fmt.Errorf("validation failed")
)

// RuleValidationError is used to report errors from evaluating a rule, including
// attribution of the particular error encountered.
type RuleValidationError struct {
	Err string
	// RuleType is a rule name
	RuleType string
}

// String implements fmt.Stringer
func (e *RuleValidationError) String() string {
	return fmt.Sprintf("error in rule %q: %s", e.RuleType, e.Err)
}

// Error implements error.Error
func (e *RuleValidationError) Error() string {
	return e.String()
}

// ParseYAML parses a YAML pipeline policy and validates it
func ParseYAML(r io.Reader) (*pb.PipelinePolicy, error) {
	w := &bytes.Buffer{}
	if err := util.TranscodeYAMLToJSON(r, w); err != nil {
		return nil, fmt.Errorf("error converting yaml to json: %w", err)
	}
	return ParseJSON(w)
}

// ParseJSON parses a JSON pipeline policy and validates it
func ParseJSON(r io.Reader) (*pb.PipelinePolicy, error) {
	var out pb.PipelinePolicy

	dec := json.NewDecoder(r)
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	if err := ValidatePolicy(&out); err != nil {
		return nil, fmt.Errorf("error validating policy: %w", err)
	}

	return &out, nil
}

// ReadPolicyFromFile reads a pipeline policy from a file and returns it as a protobuf
func ReadPolicyFromFile(fpath string) (*pb.PipelinePolicy, error) {
	f, err := os.Open(filepath.Clean(fpath))
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	defer f.Close()
	var out *pb.PipelinePolicy

	if filepath.Ext(fpath) == ".json" {
		out, err = ParseJSON(f)
	} else {
		// parse yaml by default
		out, err = ParseYAML(f)
	}
	if err != nil {
		return nil, fmt.Errorf("error parsing policy: %w", err)
	}

	return out, nil
}

// ValidatePolicy validates a pipeline policy
func ValidatePolicy(p *pb.PipelinePolicy) error {
	if err := validateContext(p.Context); err != nil {
		return err
	}

	// If the policy is nil or empty, we don't need to validate it
	if p.Repository != nil && len(p.Repository) > 0 {
		return validateEntity(p.Repository)
	}

	if p.BuildEnvironment != nil && len(p.BuildEnvironment) > 0 {
		return validateEntity(p.BuildEnvironment)
	}

	if p.Artifact != nil && len(p.Artifact) > 0 {
		return validateEntity(p.Artifact)
	}

	return nil
}

func validateContext(c *pb.Context) error {
	if c == nil {
		return fmt.Errorf("%w: context cannot be empty", ErrValidationFailed)
	}

	if c.Provider == "" {
		return fmt.Errorf("%w: context provider cannot be empty", ErrValidationFailed)
	}

	if c.Organization == nil && c.Group == nil {
		return fmt.Errorf("%w: context organization or group must be set", ErrValidationFailed)
	}

	if c.Organization != nil && *c.Organization == "" {
		return fmt.Errorf("%w: context organization cannot be empty", ErrValidationFailed)
	}

	if c.Group != nil && *c.Group == "" {
		return fmt.Errorf("%w: context group cannot be empty", ErrValidationFailed)
	}

	return nil
}

func validateEntity(e []*pb.PipelinePolicy_ContextualRuleSet) error {
	if len(e) == 0 {
		return fmt.Errorf("%w: entity rules cannot be empty", ErrValidationFailed)
	}

	for _, r := range e {
		if r == nil {
			return fmt.Errorf("%w: entity contextual rules cannot be nil", ErrValidationFailed)
		}

		if err := validateContextualRuleSet(r); err != nil {
			return err
		}
	}

	return nil
}

func validateContextualRuleSet(e *pb.PipelinePolicy_ContextualRuleSet) error {
	if e.Rules == nil {
		return fmt.Errorf("%w: entity rules cannot be nil", ErrValidationFailed)
	}

	for _, r := range e.Rules {
		if err := validateRule(r); err != nil {
			return err
		}
	}

	return nil
}

func validateRule(r *pb.PipelinePolicy_Rule) error {
	if r.Type == "" {
		return fmt.Errorf("%w: rule type cannot be empty", ErrValidationFailed)
	}

	if r.Def == nil {
		return fmt.Errorf("%w: rule def cannot be nil", ErrValidationFailed)
	}

	return nil
}

// GetRulesForEntity returns the rules for the given entity
func GetRulesForEntity(p *pb.PipelinePolicy, entity pb.Entity) ([]*pb.PipelinePolicy_ContextualRuleSet, error) {
	switch entity {
	case pb.Entity_ENTITY_REPOSITORIES:
		return p.Repository, nil
	case pb.Entity_ENTITY_BUILD_ENVIRONMENTS:
		return p.BuildEnvironment, nil
	case pb.Entity_ENTITY_ARTIFACTS:
		return p.Artifact, nil
	default:
		return nil, fmt.Errorf("unknown entity: %s", entity)
	}
}

// TraverseAllRulesForPipeline traverses all rules for the given pipeline policy
func TraverseAllRulesForPipeline(p *pb.PipelinePolicy, fn func(*pb.PipelinePolicy_Rule) error) error {
	if err := TraverseRules(p.Repository, fn); err != nil {
		return fmt.Errorf("error traversing repository rules: %w", err)
	}

	if err := TraverseRules(p.BuildEnvironment, fn); err != nil {
		return fmt.Errorf("error traversing build environment rules: %w", err)
	}

	if err := TraverseRules(p.Artifact, fn); err != nil {
		return fmt.Errorf("error traversing artifact rules: %w", err)
	}

	return nil
}

// TraverseRules traverses the rules and calls the given function for each rule
// TODO: do we want to collect and return _all_ errors, rather than just the first,
// to prevent whack-a-mole fixing?
func TraverseRules(cr []*pb.PipelinePolicy_ContextualRuleSet, fn func(*pb.PipelinePolicy_Rule) error) error {
	for _, r := range cr {
		for _, rule := range r.Rules {
			if err := fn(rule); err != nil {
				return &RuleValidationError{err.Error(), rule.GetType()}
			}
		}
	}

	return nil
}

// MergeDatabaseListIntoPolicies merges the database list policies into the given
// policies map. This assumes that the policies belong to the same group.
//
// TODO(jaosorior): This will have to consider the project tree once we	migrate to that
func MergeDatabaseListIntoPolicies(ppl []db.ListPoliciesByGroupIDRow, ectx *EntityContext) map[string]*pb.PipelinePolicy {
	policies := map[string]*pb.PipelinePolicy{}

	for idx := range ppl {
		p := ppl[idx]

		// NOTE: names are unique within a given Provider & Group ID (Unique index),
		// so we don't need to worry about collisions.
		// first we check if policy already exists, if not we create a new one
		// first we check if policy already exists, if not we create a new one
		if _, ok := policies[p.Name]; !ok {
			policies[p.Name] = &pb.PipelinePolicy{
				Id:   &p.ID,
				Name: p.Name,
				Context: &pb.Context{
					Provider: p.Provider,
					Group:    &ectx.Group.Name,
				},
			}
		}
		if pm := rowInfoToPolicyMap(policies[p.Name], p.Entity, p.ContextualRules); pm != nil {
			policies[p.Name] = pm
		}
	}

	return policies
}

// MergeDatabaseGetIntoPolicies merges the database get policies into the given
// policies map. This assumes that the policies belong to the same group.
//
// TODO(jaosorior): This will have to consider the project tree once we migrate to that
func MergeDatabaseGetIntoPolicies(ppl []db.GetPolicyByGroupAndIDRow, ectx *EntityContext) map[string]*pb.PipelinePolicy {
	policies := map[string]*pb.PipelinePolicy{}

	for idx := range ppl {
		p := ppl[idx]

		// NOTE: names are unique within a given Provider & Group ID (Unique index),
		// so we don't need to worry about collisions.

		// first we check if policy already exists, if not we create a new one
		if _, ok := policies[p.Name]; !ok {
			policies[p.Name] = &pb.PipelinePolicy{
				Id:   &p.ID,
				Name: p.Name,
				Context: &pb.Context{
					Provider: p.Provider,
					Group:    &ectx.Group.Name,
				},
			}
		}
		if pm := rowInfoToPolicyMap(policies[p.Name], p.Entity, p.ContextualRules); pm != nil {
			policies[p.Name] = pm
		}
	}

	return policies
}

// rowInfoToPolicyMap adds the database row information to the given map of
// policies. This assumes that the policies belong to the same group.
// Note that this function is thought to be called from scpecific Merge functions
// and thus the logic is targetted to that.
func rowInfoToPolicyMap(
	policy *pb.PipelinePolicy,
	entity db.Entities,
	contextualRules json.RawMessage,
) *pb.PipelinePolicy {
	if !entities.IsValidEntity(entities.EntityTypeFromDB(entity)) {
		log.Printf("unknown entity found in database: %s", entity)
		return nil
	}

	var ruleset []*pb.PipelinePolicy_ContextualRuleSet

	if err := json.Unmarshal(contextualRules, &ruleset); err != nil {
		// We merely print the error and continue. This is because the user
		// can't do anything about it and it's not a critical error.
		log.Printf("error unmarshalling contextual rules; there is corruption in the database: %s", err)
		return nil
	}

	switch entities.EntityTypeFromDB(entity) {
	case pb.Entity_ENTITY_REPOSITORIES:
		policy.Repository = ruleset
	case pb.Entity_ENTITY_BUILD_ENVIRONMENTS:
		policy.BuildEnvironment = ruleset
	case pb.Entity_ENTITY_ARTIFACTS:
		policy.Artifact = ruleset
	}

	return policy
}
