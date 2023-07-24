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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"

	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

var (
	// ErrInvalidRuleTypeDefinition is returned when a rule type definition is invalid
	ErrInvalidRuleTypeDefinition = errors.New("invalid rule type definition")
)

// RuleMeta is the metadata for a rule
// TODO: We probably should care about a version
type RuleMeta struct {
	// Name is the name of the rule
	Name string
	// Provider is the ID of the provider that this rule is for
	Provider string
	// Organization is the ID of the organization that this rule is for
	Organization *string
	// Group is the ID of the group that this rule is for
	Group *string
}

func (r *RuleMeta) String() string {
	if r.Group != nil {
		return fmt.Sprintf("%s/group/%s/%s", r.Provider, *r.Group, r.Name)
	}
	return fmt.Sprintf("%s/org/%s/%s", r.Provider, *r.Organization, r.Name)
}

// RuleTypeEngine is the engine for a rule type
type RuleTypeEngine struct {
	Meta RuleMeta

	// schema is the schema that this rule type must conform to
	schema *gojsonschema.Schema

	// rdi is the rule data ingest engine
	rdi RuleDataIngest

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

	rdi, err := NewRuleDataIngest(rt, cli)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule data ingest: %w", err)
	}

	rte := &RuleTypeEngine{
		Meta: RuleMeta{
			Name:     rt.Name,
			Provider: rt.Context.Provider,
		},
		schema: schema,
		rdi:    rdi,
		rt:     rt,
		cli:    cli,
	}

	// Set organization if it exists
	if rt.Context.Organization != nil && *rt.Context.Organization != "" {
		// We need to clone the string because the pointer is to a string literal
		// and we don't want to modify that
		org := strings.Clone(*rt.Context.Organization)
		rte.Meta.Organization = &org
	} else if rt.Context.Group != nil && *rt.Context.Group != "" {
		grp := strings.Clone(*rt.Context.Group)
		rte.Meta.Group = &grp
	} else {
		return nil, fmt.Errorf("rule type context must have an organization or group")
	}

	return rte, nil
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

// Eval runs the rule type engine against the given entity
func (r *RuleTypeEngine) Eval(ctx context.Context, ent any, pol, params map[string]any) error {
	return r.rdi.Eval(ctx, ent, pol, params)
}

// DBRuleDefFromPB converts a protobuf rule type definition to a database
// rule type definition
func DBRuleDefFromPB(def *pb.RuleType_Definition) ([]byte, error) {
	return json.Marshal(def)
}

// RuleDefFromDB converts a rule type definition from the database to a protobuf
// rule type definition
func RuleDefFromDB(r *db.RuleType) (*pb.RuleType_Definition, error) {
	def := &pb.RuleType_Definition{}

	if err := json.Unmarshal(r.Definition, def); err != nil {
		return nil, fmt.Errorf("cannot unmarshal rule type definition: %w", err)
	}
	return nil, nil
}

// RuleTypePBFromDB converts a rule type from the database to a protobuf
// rule type
func RuleTypePBFromDB(rt *db.RuleType, ectx *EntityContext) (*pb.RuleType, error) {
	gname := ectx.GetGroup().GetName()

	def, err := RuleDefFromDB(rt)
	if err != nil {
		return nil, fmt.Errorf("cannot get rule type definition: %w", err)
	}

	id := rt.ID

	return &pb.RuleType{
		Id:   &id,
		Name: rt.Name,
		Context: &pb.Context{
			Provider: ectx.GetProvider(),
			Group:    &gname,
		},
		Def: def,
	}, nil
}

// ValidateRuleTypeDefinition validates a rule type definition
func ValidateRuleTypeDefinition(def *pb.RuleType_Definition) error {
	if def == nil {
		return fmt.Errorf("%w: rule type definition is nil", ErrInvalidRuleTypeDefinition)
	}

	if !IsValidEntity(def.InEntity) {
		return fmt.Errorf("%w: invalid entity type: %s", ErrInvalidRuleTypeDefinition, def.InEntity)
	}

	if def.RuleSchema == nil {
		return fmt.Errorf("%w: rule schema is nil", ErrInvalidRuleTypeDefinition)
	}

	if def.DataEval == nil {
		return fmt.Errorf("%w: data evaluation is nil", ErrInvalidRuleTypeDefinition)
	}

	if def.DataEval.Data == nil {
		return fmt.Errorf("%w: data evaluation data is nil", ErrInvalidRuleTypeDefinition)
	}

	return nil
}
