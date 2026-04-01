// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package schemavalidate

import (
	"testing"
)

func TestCompileSchemaFromPB_Nil(t *testing.T) {
	t.Parallel()
	schema, err := CompileSchemaFromPB(nil)
	if err != nil {
		t.Fatalf("CompileSchemaFromPB(nil) error = %v, want nil", err)
	}
	if schema != nil {
		t.Error("CompileSchemaFromPB(nil) schema = non-nil, want nil")
	}
}

func TestCompileSchemaFromMap_Valid(t *testing.T) {
	t.Parallel()
	m := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
	}
	schema, err := CompileSchemaFromMap(m)
	if err != nil {
		t.Fatalf("CompileSchemaFromMap() error = %v, want nil", err)
	}
	if schema == nil {
		t.Fatal("CompileSchemaFromMap() returned nil schema")
	}
}

func TestValidateAgainstSchema_Valid(t *testing.T) {
	t.Parallel()
	m := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"required": []any{"name"},
	}
	schema, err := CompileSchemaFromMap(m)
	if err != nil {
		t.Fatalf("CompileSchemaFromMap() error = %v", err)
	}
	err = ValidateAgainstSchema(schema, map[string]any{"name": "Alice"})
	if err != nil {
		t.Errorf("ValidateAgainstSchema() error = %v, want nil", err)
	}
}

func TestValidateAgainstSchema_Invalid(t *testing.T) {
	t.Parallel()
	m := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"count": map[string]any{"type": "integer"},
		},
		"required": []any{"count"},
	}
	schema, err := CompileSchemaFromMap(m)
	if err != nil {
		t.Fatalf("CompileSchemaFromMap() error = %v", err)
	}
	err = ValidateAgainstSchema(schema, map[string]any{})
	if err == nil {
		t.Error("ValidateAgainstSchema() want error for missing required field")
	}
}

func TestApplyDefaults_AppliesDefault(t *testing.T) {
	t.Parallel()
	defVal := "default-name"
	m := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string", "default": defVal},
		},
	}
	schema, err := CompileSchemaFromMap(m)
	if err != nil {
		t.Fatalf("CompileSchemaFromMap() error = %v", err)
	}
	obj := map[string]any{}
	ApplyDefaults(schema, obj)
	if obj["name"] != defVal {
		t.Errorf("ApplyDefaults() name = %v, want %q", obj["name"], defVal)
	}
}

func TestApplyDefaults_DoesNotOverwrite(t *testing.T) {
	t.Parallel()
	m := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string", "default": "default-name"},
		},
	}
	schema, err := CompileSchemaFromMap(m)
	if err != nil {
		t.Fatalf("CompileSchemaFromMap() error = %v", err)
	}
	obj := map[string]any{"name": "custom"}
	ApplyDefaults(schema, obj)
	if obj["name"] != "custom" {
		t.Errorf("ApplyDefaults() name = %v, want 'custom'", obj["name"])
	}
}
