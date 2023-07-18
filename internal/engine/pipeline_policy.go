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

	"github.com/stacklok/mediator/internal/util"
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
