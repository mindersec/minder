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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/mediator/internal/util"
	"github.com/stacklok/mediator/pkg/db"
	"github.com/stacklok/mediator/pkg/entities"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

var (
	// ErrInvalidRuleTypeDefinition is returned when a rule type definition is invalid
	ErrInvalidRuleTypeDefinition = errors.New("invalid rule type definition")
)

// ParseRuleType parses a rule type from a reader
func ParseRuleType(r io.Reader) (*pb.RuleType, error) {
	// We transcode to JSON so we can decode it straight to the protobuf structure
	w := &bytes.Buffer{}
	if err := util.TranscodeYAMLToJSON(r, w); err != nil {
		return nil, fmt.Errorf("error converting yaml to json: %w", err)
	}

	rt := &pb.RuleType{}
	if err := json.NewDecoder(w).Decode(rt); err != nil {
		return nil, fmt.Errorf("error decoding json: %w", err)
	}

	return rt, nil
}

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
	// paramSchema is the schema that the parameters for this rule type must conform to
	paramSchema *gojsonschema.Schema
}

// NewRuleValidator creates a new rule validator
func NewRuleValidator(rt *pb.RuleType) (*RuleValidator, error) {
	// Load schemas
	schemaLoader := gojsonschema.NewGoLoader(rt.Def.RuleSchema)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return nil, fmt.Errorf("cannot create json schema: %w", err)
	}

	var paramSchema *gojsonschema.Schema
	if rt.Def.ParamSchema != nil {
		paramSchemaLoader := gojsonschema.NewGoLoader(rt.Def.ParamSchema)
		paramSchema, err = gojsonschema.NewSchema(paramSchemaLoader)
		if err != nil {
			return nil, fmt.Errorf("cannot create json schema for params: %w", err)
		}
	}

	return &RuleValidator{
		schema:      schema,
		paramSchema: paramSchema,
	}, nil
}

// ValidateRuleDefAgainstSchema validates the given contextual policy against the
// schema for this rule type
func (r *RuleValidator) ValidateRuleDefAgainstSchema(contextualPolicy map[string]any) error {
	return validateAgainstSchema(r.schema, contextualPolicy)
}

// ValidateParamsAgainstSchema validates the given parameters against the
// schema for this rule type
func (r *RuleValidator) ValidateParamsAgainstSchema(params *structpb.Struct) error {
	if r.paramSchema == nil {
		return nil
	}

	if params == nil {
		return fmt.Errorf("params cannot be nil")
	}

	return validateAgainstSchema(r.paramSchema, params.AsMap())
}

func validateAgainstSchema(schema *gojsonschema.Schema, obj map[string]any) error {
	documentLoader := gojsonschema.NewGoLoader(obj)
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("cannot validate json schema: %w", err)
	}

	if !result.Valid() {
		return buildValidationError(result.Errors())
	}

	return nil
}

func buildValidationError(errs []gojsonschema.ResultError) error {
	problems := make([]string, 0, len(errs))
	for _, desc := range errs {
		problems = append(problems, desc.String())
	}

	return fmt.Errorf("invalid json schema: %s", strings.TrimSpace(strings.Join(problems, "\n")))
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
func NewRuleTypeEngine(rt *pb.RuleType, cli ghclient.RestAPI, accessToken string) (*RuleTypeEngine, error) {
	rval, err := NewRuleValidator(rt)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule validator: %w", err)
	}

	rdi, err := NewRuleDataIngest(rt, cli, accessToken)
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

// GetRuleInstanceValidator returns the rule instance validator for this rule type.
// By instance we mean a rule that has been instantiated in a policy from a given rule type.
func (r *RuleTypeEngine) GetRuleInstanceValidator() *RuleValidator {
	return r.rval
}

// Eval runs the rule type engine against the given entity
func (r *RuleTypeEngine) Eval(ctx context.Context, ent protoreflect.ProtoMessage, pol, params map[string]any) error {
	return r.rdi.Eval(ctx, ent, pol, params)
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
		Description: rt.Description,
		Def:         def,
	}, nil
}

// ValidateRuleTypeDefinition validates a rule type definition
func ValidateRuleTypeDefinition(def *pb.RuleType_Definition) error {
	if def == nil {
		return fmt.Errorf("%w: rule type definition is nil", ErrInvalidRuleTypeDefinition)
	}

	if !entities.IsValidEntity(entities.FromString(def.InEntity)) {
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

// GetRulesFromPolicyOfType returns the rules from the policy of the given type
func GetRulesFromPolicyOfType(p *pb.PipelinePolicy, rt *pb.RuleType) ([]*pb.PipelinePolicy_Rule, error) {
	contextualRules, err := GetRulesForEntity(p, entities.FromString(rt.Def.InEntity))
	if err != nil {
		return nil, fmt.Errorf("error getting rules for entity: %w", err)
	}

	rules := []*pb.PipelinePolicy_Rule{}
	err = TraverseRules(contextualRules, func(r *pb.PipelinePolicy_Rule) error {
		if r.Type == rt.Name {
			rules = append(rules, r)
		}
		return nil
	})

	// This shouldn't happen
	if err != nil {
		return nil, fmt.Errorf("error traversing rules: %w", err)
	}

	return rules, nil
}
