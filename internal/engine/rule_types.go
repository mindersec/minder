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
// Package rule provides the CLI subcommand for managing rules

package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/engine/actions"
	evalerrors "github.com/stacklok/mediator/internal/engine/errors"
	"github.com/stacklok/mediator/internal/engine/eval"
	"github.com/stacklok/mediator/internal/engine/ingester"
	engif "github.com/stacklok/mediator/internal/engine/interfaces"
	"github.com/stacklok/mediator/internal/providers"
	mediatorv1 "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
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
	// Project is the ID of the group that this rule is for
	Project *string
}

// String returns a string representation of the rule meta
func (r *RuleMeta) String() string {
	if r.Project != nil {
		return fmt.Sprintf("%s/group/%s/%s", r.Provider, *r.Project, r.Name)
	}
	return fmt.Sprintf("%s/org/%s/%s", r.Provider, *r.Organization, r.Name)
}

// RuleValidator validates a rule against a schema
type RuleValidator struct {
	ruleTypeName string

	// schema is the schema that this rule type must conform to
	schema *gojsonschema.Schema
	// paramSchema is the schema that the parameters for this rule type must conform to
	paramSchema *gojsonschema.Schema
}

// NewRuleValidator creates a new rule validator
func NewRuleValidator(rt *mediatorv1.RuleType) (*RuleValidator, error) {
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
		ruleTypeName: rt.Name,
		schema:       schema,
		paramSchema:  paramSchema,
	}, nil
}

// ValidateRuleDefAgainstSchema validates the given contextual profile against the
// schema for this rule type
func (r *RuleValidator) ValidateRuleDefAgainstSchema(contextualProfile map[string]any) error {
	if err := validateAgainstSchema(r.schema, contextualProfile); err != nil {
		return &RuleValidationError{
			RuleType: r.ruleTypeName,
			Err:      err.Error(),
		}
	}

	return nil
}

// ValidateParamsAgainstSchema validates the given parameters against the
// schema for this rule type
func (r *RuleValidator) ValidateParamsAgainstSchema(params *structpb.Struct) error {
	if r.paramSchema == nil {
		return nil
	}

	if params == nil {
		return &RuleValidationError{
			RuleType: r.ruleTypeName,
			Err:      "params cannot be nil",
		}
	}

	if err := validateAgainstSchema(r.paramSchema, params.AsMap()); err != nil {
		return &RuleValidationError{
			RuleType: r.ruleTypeName,
			Err:      err.Error(),
		}
	}

	return nil
}

func validateAgainstSchema(schema *gojsonschema.Schema, obj map[string]any) error {
	documentLoader := gojsonschema.NewGoLoader(obj)
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("cannot validate json schema: %s", err)
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
	rdi engif.Ingester

	// reval is the rule evaluator
	reval engif.Evaluator

	// rae is the rule actions engine
	rae *actions.RuleActionsEngine

	rval *RuleValidator

	rt *mediatorv1.RuleType

	cli *providers.ProviderBuilder
}

// NewRuleTypeEngine creates a new rule type engine
func NewRuleTypeEngine(p *mediatorv1.Profile, rt *mediatorv1.RuleType, cli *providers.ProviderBuilder,
) (*RuleTypeEngine, error) {
	rval, err := NewRuleValidator(rt)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule validator: %w", err)
	}

	rdi, err := ingester.NewRuleDataIngest(rt, cli)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule data ingest: %w", err)
	}

	reval, err := eval.NewRuleEvaluator(rt, cli)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule evaluator: %w", err)
	}

	ae, err := actions.NewRuleActions(p, rt, cli)
	if err != nil {
		return nil, fmt.Errorf("cannot create rule actions engine: %w", err)
	}

	rte := &RuleTypeEngine{
		Meta: RuleMeta{
			Name:     rt.Name,
			Provider: rt.Context.Provider,
		},
		rval:  rval,
		rdi:   rdi,
		reval: reval,
		rae:   ae,
		rt:    rt,
		cli:   cli,
	}

	// Set organization if it exists
	if rt.Context.Organization != nil && *rt.Context.Organization != "" {
		// We need to clone the string because the pointer is to a string literal
		// and we don't want to modify that
		org := strings.Clone(*rt.Context.Organization)
		rte.Meta.Organization = &org
	} else if rt.Context.Project != nil && *rt.Context.Project != "" {
		prj := strings.Clone(*rt.Context.Project)
		rte.Meta.Project = &prj
	} else {
		return nil, fmt.Errorf("rule type context must have an organization or project")
	}

	return rte, nil
}

// GetID returns the ID of the rule type. The ID is meant to be
// a serializable unique identifier for the rule type.
func (r *RuleTypeEngine) GetID() string {
	return r.Meta.String()
}

// GetRuleInstanceValidator returns the rule instance validator for this rule type.
// By instance we mean a rule that has been instantiated in a profile from a given rule type.
func (r *RuleTypeEngine) GetRuleInstanceValidator() *RuleValidator {
	return r.rval
}

// Eval runs the rule type engine against the given entity
func (r *RuleTypeEngine) Eval(ctx context.Context, ent protoreflect.ProtoMessage, ruleDef, ruleParams map[string]any,
	evalParams *EvalStatusParams) {
	result, err := r.rdi.Ingest(ctx, ent, ruleParams)
	if err != nil {
		evalParams.EvalErr = fmt.Errorf("error ingesting data: %w", err)
		evalParams.ActionsErr = evalerrors.ActionsError{
			RemediateErr: evalerrors.ErrActionSkipped,
			AlertErr:     evalerrors.ErrActionSkipped,
		}
		return
	}
	evalParams.EvalErr = r.reval.Eval(ctx, ruleDef, result)
}

// Actions runs all actions for the rule type engine against the given entity
func (r *RuleTypeEngine) Actions(
	ctx context.Context,
	ent protoreflect.ProtoMessage,
	ruleDef, ruleParams map[string]any,
	evalParams *EvalStatusParams,
) {
	// Skip actions in case ingesting failed during evaluation
	if errors.Is(evalParams.ActionsErr.AlertErr, evalerrors.ErrActionSkipped) &&
		errors.Is(evalParams.ActionsErr.RemediateErr, evalerrors.ErrActionSkipped) {
		return
	}

	// Process actions
	evalParams.ActionsErr = r.rae.DoActions(ctx,
		ent,
		ruleDef,
		ruleParams,
		evalParams.EvalErr,
		evalParams.EvalStatusFromDb,
	)
}

// RuleDefFromDB converts a rule type definition from the database to a protobuf
// rule type definition
func RuleDefFromDB(r *db.RuleType) (*mediatorv1.RuleType_Definition, error) {
	def := &mediatorv1.RuleType_Definition{}

	if err := protojson.Unmarshal(r.Definition, def); err != nil {
		return nil, fmt.Errorf("cannot unmarshal rule type definition: %w", err)
	}
	return def, nil
}

// RuleTypePBFromDB converts a rule type from the database to a protobuf
// rule type
func RuleTypePBFromDB(rt *db.RuleType, ectx *EntityContext) (*mediatorv1.RuleType, error) {
	gname := ectx.GetProject().GetName()

	def, err := RuleDefFromDB(rt)
	if err != nil {
		return nil, fmt.Errorf("cannot get rule type definition: %w", err)
	}

	id := rt.ID.String()

	return &mediatorv1.RuleType{
		Id:   &id,
		Name: rt.Name,
		Context: &mediatorv1.Context{
			Provider: ectx.GetProvider().Name,
			Project:  &gname,
		},
		Description: rt.Description,
		Guidance:    rt.Guidance,
		Def:         def,
	}, nil
}

// GetRulesFromProfileOfType returns the rules from the profile of the given type
func GetRulesFromProfileOfType(p *mediatorv1.Profile, rt *mediatorv1.RuleType) ([]*mediatorv1.Profile_Rule, error) {
	contextualRules, err := GetRulesForEntity(p, mediatorv1.EntityFromString(rt.Def.InEntity))
	if err != nil {
		return nil, fmt.Errorf("error getting rules for entity: %w", err)
	}

	rules := []*mediatorv1.Profile_Rule{}
	err = TraverseRules(contextualRules, func(r *mediatorv1.Profile_Rule) error {
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
