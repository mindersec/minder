// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profiles

import (
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/mindersec/minder/internal/util/schemavalidate"
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
	if rt.GetDef().GetRuleSchema() == nil {
		return nil, fmt.Errorf("rule type %s does not have a rule schema", rt.Name)
	}
	// Create a new schema compiler
	// Compile the main rule schema
	mainSchema, err := schemavalidate.CompileSchemaFromPB(rt.GetDef().GetRuleSchema())
	if err != nil {
		return nil, fmt.Errorf("cannot create json schema: %w", err)
	}

	// Compile the parameter schema if it exists
	paramSchema, err := schemavalidate.CompileSchemaFromPB(rt.GetDef().GetParamSchema())
	if err != nil {
		return nil, fmt.Errorf("cannot create json schema for params: %w", err)
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
	if err := schemavalidate.ValidateAgainstSchema(r.schema, contextualProfile); err != nil {
		return &RuleValidationError{
			RuleType: r.ruleTypeName,
			Err:      err.Error(),
		}
	}
	schemavalidate.ApplyDefaults(r.schema, contextualProfile)
	return nil
}

// ValidateParamsAgainstSchema validates the given parameters against the
// schema for this rule type
func (r *RuleValidator) ValidateParamsAgainstSchema(params map[string]any) error {
	if r.paramSchema == nil {
		return nil
	}
	if err := schemavalidate.ValidateAgainstSchema(r.paramSchema, params); err != nil {
		return &RuleValidationError{
			RuleType: r.ruleTypeName,
			Err:      err.Error(),
		}
	}
	schemavalidate.ApplyDefaults(r.paramSchema, params)
	return nil
}
