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
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

var (
	// ErrValidationFailed is returned when a policy fails validation
	ErrValidationFailed = fmt.Errorf("validation failed")
)

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

	if filepath.Ext(fpath) == ".json" {
		return ParseJSON(f)
	}

	// parse yaml by default
	return ReadYAMLPolicyFromReader(f)
}

// ReadYAMLPolicyFromReader reads a pipeline policy from a reader and returns it as a protobuf
func ReadYAMLPolicyFromReader(r io.Reader) (*pb.PipelinePolicy, error) {
	return ParseYAML(r)
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
func GetRulesForEntity(p *pb.PipelinePolicy, entity EntityType) ([]*pb.PipelinePolicy_ContextualRuleSet, error) {
	switch entity {
	case RepositoryEntity:
		return p.Repository, nil
	case BuildEnvironmentEntity:
		return p.BuildEnvironment, nil
	case ArtifactEntity:
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
func TraverseRules(cr []*pb.PipelinePolicy_ContextualRuleSet, fn func(*pb.PipelinePolicy_Rule) error) error {
	for _, r := range cr {
		for _, rule := range r.Rules {
			if err := fn(rule); err != nil {
				return err
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

		rowInfoToPolicyMap(p.Entity, p.Name, p.Provider, p.ContextualRules, ectx, policies)
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

		policies = rowInfoToPolicyMap(p.Entity, p.Name, p.Provider, p.ContextualRules, ectx, policies)
	}

	return policies
}

// rowInfoToPolicyMap adds the database row information to the given map of
// policies. This assumes that the policies belong to the same group.
// Note that this function is thought to be called from scpecific Merge functions
// and thus the logic is targetted to that.
func rowInfoToPolicyMap(
	entity db.Entities,
	name string,
	provider string,
	contextualRules json.RawMessage,
	ectx *EntityContext,
	in map[string]*pb.PipelinePolicy,
) map[string]*pb.PipelinePolicy {
	if !IsValidEntity(EntityTypeFromDB(entity)) {
		log.Printf("unknown entity found in database: %s", entity)
		return in
	}

	if _, ok := in[name]; !ok {
		in[name] = &pb.PipelinePolicy{
			Name: name,
			Context: &pb.Context{
				Provider: provider,
				Group:    &ectx.Group.Name,
			},
		}
	}

	var ruleset []*pb.PipelinePolicy_ContextualRuleSet

	parsedContextualRules := []*pb.PipelinePolicy_ContextualRuleSet{}
	if err := json.Unmarshal(contextualRules, &parsedContextualRules); err != nil {
		// We merely print the error and continue. This is because the user
		// can't do anything about it and it's not a critical error.
		log.Printf("error unmarshalling contextual rules; there is corruption in the database: %s", err)
		return in
	}
	ruleset = append(ruleset, parsedContextualRules...)

	switch EntityTypeFromDB(entity) {
	case RepositoryEntity:
		in[name].Repository = ruleset
	case BuildEnvironmentEntity:
		in[name].BuildEnvironment = ruleset
	case ArtifactEntity:
		in[name].Artifact = ruleset
	}

	return in
}
