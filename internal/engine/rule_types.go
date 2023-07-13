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
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

// RuleMeta is the metadata for a rule
// TODO: We probably should care about a version
type RuleMeta struct {
	// Name is the name of the rule
	Name string
	// Provider is the ID of the provider that this rule is for
	Provider string
	// Organization is the ID of the organization that this rule is for
	Organization string
}

func (r *RuleMeta) String() string {
	return fmt.Sprintf("%s/%s/%s", r.Provider, r.Organization, r.Name)
}

// RuleTypeEngine is the engine for a rule type
type RuleTypeEngine struct {
	Meta RuleMeta

	// schema is the schema that this rule type must conform to
	schema *gojsonschema.Schema

	rt *pb.RuleType
	// TODO(JAORMX): We need to have an abstract client interface
	cli ghclient.RestAPI
}

// NewRuleTypeEngine creates a new rule type engine
func NewRuleTypeEngine(rt *pb.RuleType, cli ghclient.RestAPI) (*RuleTypeEngine, error) {
	// Load schema
	schemaLoader := gojsonschema.NewGoLoader(rt.Def.RuleSchema)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return nil, fmt.Errorf("cannot create json schema: %w", err)
	}

	return &RuleTypeEngine{
		Meta: RuleMeta{
			Name:         rt.Name,
			Provider:     rt.Context.Provider,
			Organization: rt.Context.Organization,
		},
		schema: schema,
		rt:     rt,
		cli:    cli,
	}, nil
}

// GetID returns the ID of the rule type. The ID is meant to be
// a serializable unique identifier for the rule type.
func (r *RuleTypeEngine) GetID() string {
	return r.Meta.String()
}

// ValidateAgainstSchema validates the given contextual policy against the
// schema for this rule type
func (r *RuleTypeEngine) ValidateAgainstSchema(contextualPolicy any) (*bool, error) {
	documentLoader := gojsonschema.NewGoLoader(contextualPolicy)
	result, err := r.schema.Validate(documentLoader)
	if err != nil {
		return nil, fmt.Errorf("cannot validate json schema: %v", err)
	}

	out := result.Valid()

	if !out {
		description := ""
		for _, desc := range result.Errors() {
			description += fmt.Sprintf("%s\n", desc)
		}

		description = strings.TrimSpace(description)

		return &out, fmt.Errorf("invalid json schema: %s", description)
	}

	return &out, nil
}
