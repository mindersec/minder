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
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/protobuf/types/known/structpb"

	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// RuleValidator takes a rule type and validates an instance of it. The main
// purpose of this is to validate the schemas that are associated with the rule.
type RuleValidator struct {
	ruleTypeName string

	// schema is the schema that this rule type must conform to
	schema *gojsonschema.Schema
	// paramSchema is the schema that the parameters for this rule type must conform to
	paramSchema *gojsonschema.Schema
}

// NewRuleValidator creates a new rule validator
func NewRuleValidator(rt *minderv1.RuleType) (*RuleValidator, error) {
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
