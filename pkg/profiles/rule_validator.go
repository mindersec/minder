// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profiles

import (
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// RuleValidator takes a rule type and validates an instance of it. The main
// purpose of this is to validate the schemas that are associated with the rule.
type RuleValidator struct {
	ruleTypeName string

	// schema is the schema that this rule type must conform to
	schema *jsonschema.Schema
	// paramSchema is the schema that the parameters for this rule type must conform to
	paramSchema *jsonschema.Schema
}

// NewRuleValidator creates a new rule validator
func NewRuleValidator(rt *minderv1.RuleType) (*RuleValidator, error) {
	// Create a new schema compiler
	// Compile the main rule schema
	mainSchema, err := compileSchema(rt.GetDef().GetRuleSchema().AsMap())
	if err != nil {
		return nil, fmt.Errorf("cannot create json schema: %w", err)
	}

	// Compile the parameter schema if it exists
	var paramSchema *jsonschema.Schema
	if rt.Def.ParamSchema != nil {
		paramSchema, err = compileSchema(rt.GetDef().GetParamSchema().AsMap())
		if err != nil {
			return nil, fmt.Errorf("cannot create json schema for params: %w", err)
		}
	}

	return &RuleValidator{
		ruleTypeName: rt.Name,
		schema:       mainSchema,
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
	applyDefaults(r.schema, contextualProfile)
	return nil
}

// ValidateParamsAgainstSchema validates the given parameters against the
// schema for this rule type
func (r *RuleValidator) ValidateParamsAgainstSchema(params map[string]any) error {
	if r.paramSchema == nil {
		return nil
	}
	if err := validateAgainstSchema(r.paramSchema, params); err != nil {
		return &RuleValidationError{
			RuleType: r.ruleTypeName,
			Err:      err.Error(),
		}
	}
	applyDefaults(r.paramSchema, params)
	return nil
}

func compileSchema(schemaData interface{}) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", schemaData); err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}
	return compiler.Compile("schema.json")
}

func validateAgainstSchema(schema *jsonschema.Schema, obj map[string]any) error {
	if err := schema.Validate(obj); err != nil {
		return buildValidationError(err.(*jsonschema.ValidationError).Causes)
	}
	return nil
}

func buildValidationError(errs []*jsonschema.ValidationError) error {
	problems := make([]string, 0, len(errs))
	for _, desc := range errs {
		problems = append(problems, desc.Error())
	}
	return fmt.Errorf("invalid json schema: %s", strings.TrimSpace(strings.Join(problems, "\n")))
}

// applyDefaults recursively applies default values from the schema to the object.
func applyDefaults(schema *jsonschema.Schema, obj map[string]any) {
	for key, def := range schema.Properties {
		// If the key does not exist in obj, apply the default value from the schema if present
		if _, exists := obj[key]; !exists && def.Default != nil {
			obj[key] = *def.Default
		}

		// If def has properties, apply defaults to the nested object
		if def.Properties != nil {
			o, ok := obj[key].(map[string]any)
			if !ok {
				// cannot apply defaults to non-object types
				continue
			}
			applyDefaults(def, o)
		}
	}
}
