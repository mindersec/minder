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
	"google.golang.org/protobuf/encoding/protojson"

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

// String returns a string representation of the rule meta
func (r *RuleMeta) String() string {
	if r.Group != nil {
		return fmt.Sprintf("%s/group/%s/%s", r.Provider, *r.Group, r.Name)
	}
	return fmt.Sprintf("%s/org/%s/%s", r.Provider, *r.Organization, r.Name)
}

// RuleValidator validates a rule against a schema
type RuleValidator struct {
	// schema is the schema that this rule type must conform to
	schema *gojsonschema.Schema
}

// NewRuleValidator creates a new rule validator
func NewRuleValidator(rt *pb.RuleType) (*RuleValidator, error) {
	// Load schema
	schemaLoader := gojsonschema.NewGoLoader(rt.Def.RuleSchema)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return nil, fmt.Errorf("cannot create json schema: %w", err)
	}

	return &RuleValidator{
		schema: schema,
	}, nil
}

// ValidateAgainstSchema validates the given contextual policy against the
// schema for this rule type
func (r *RuleValidator) ValidateAgainstSchema(contextualPolicy any) error {
	documentLoader := gojsonschema.NewGoLoader(contextualPolicy)
	result, err := r.schema.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("cannot validate json schema: %v", err)
	}

	if !result.Valid() {
		problems := make([]string, 0, len(result.Errors()))
		for _, desc := range result.Errors() {
			problems = append(problems, desc.String())
		}

		return fmt.Errorf("invalid json schema: %s", strings.TrimSpace(strings.Join(problems, "\n")))
	}

	return nil
}

// RuleTypeEngine is the engine for a rule type
type RuleTypeEngine struct {
	Meta RuleMeta

	// rdi is the rule data ingest engine
	rdi RuleDataIngest

	rval *RuleValidator

	rt *pb.RuleType
	// TODO(JAORMX): We need to have an abstract client interface
	cli ghclient.RestAPI
}

// NewRuleTypeEngine creates a new rule type engine
func NewRuleTypeEngine(rt *pb.RuleType, cli ghclient.RestAPI) (*RuleTypeEngine, error) {
	rval, err := NewRuleValidator(rt)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule validator: %w", err)
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
		rval: rval,
		rdi:  rdi,
		rt:   rt,
		cli:  cli,
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
func (r *RuleTypeEngine) ValidateAgainstSchema(contextualPolicy any) error {
	return r.rval.ValidateAgainstSchema(contextualPolicy)
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

	if err := protojson.Unmarshal(r.Definition, def); err != nil {
		return nil, fmt.Errorf("cannot unmarshal rule type definition: %w", err)
	}
	return def, nil
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

	if !IsValidEntity(EntityType(def.InEntity)) {
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
