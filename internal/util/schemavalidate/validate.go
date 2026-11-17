// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package schemavalidate provides utilities for validating JSON schemas.
package schemavalidate

import (
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"google.golang.org/protobuf/types/known/structpb"
)

// CompileSchemaFromPB compiles a JSON schema from a protobuf Struct.
func CompileSchemaFromPB(schemaData *structpb.Struct) (*jsonschema.Schema, error) {
	if schemaData == nil {
		return nil, nil
	}

	return CompileSchemaFromMap(schemaData.AsMap())
}

// CompileSchemaFromMap compiles a JSON schema from a map.
func CompileSchemaFromMap(schemaData map[string]any) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", schemaData); err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}
	return compiler.Compile("schema.json")
}

// ValidateAgainstSchema validates an object against a JSON schema.
func ValidateAgainstSchema(schema *jsonschema.Schema, obj map[string]any) error {
	if err := schema.Validate(obj); err != nil {
		if verror, ok := err.(*jsonschema.ValidationError); ok {
			return buildValidationError(verror.Causes)
		}
		return fmt.Errorf("invalid json schema: %s", err)
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

// copyDefault returns a deep copy of a schema default. Only the container
// kinds JSON can produce need copying; scalars are already values.
func copyDefault(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, inner := range typed {
			out[k] = copyDefault(inner)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, inner := range typed {
			out[i] = copyDefault(inner)
		}
		return out
	default:
		return v
	}
}

// ApplyDefaults recursively applies default values from the schema to the object.
func ApplyDefaults(schema *jsonschema.Schema, obj map[string]any) {
	for key, def := range schema.Properties {
		// If the key does not exist in obj, apply the default value from the schema if present
		if _, exists := obj[key]; !exists && def.Default != nil {
			// Copy the default. The compiled schema is reused across callers,
			// so handing out the value itself lets a caller's write reach the
			// schema and every later caller.
			obj[key] = copyDefault(*def.Default)
		}

		// If def has properties, apply defaults to the nested object
		if def.Properties != nil {
			o, ok := obj[key].(map[string]any)
			if !ok {
				// cannot apply defaults to non-object types
				continue
			}
			ApplyDefaults(def, o)
		}
	}
}
