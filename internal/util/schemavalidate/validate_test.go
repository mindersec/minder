// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package schemavalidate_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/util/schemavalidate"
)

func TestCompileSchemaFromPB(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		schemaPB   *structpb.Struct
		wantErr    bool
		errContain string
	}{
		{
			name:     "nil protobuf struct returns nil schema without error",
			schemaPB: nil,
			wantErr:  false,
		},
		{
			name: "valid protobuf struct compiles successfully",
			schemaPB: mustStructpbFromMap(t, map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type": "string",
					},
				},
				"required": []any{"name"},
			}),
			wantErr: false,
		},
		{
			name: "invalid schema definition in protobuf struct returns compilation error",
			schemaPB: mustStructpbFromMap(t, map[string]any{
				"type": 12345, // invalid type property fails metaschema validation
			}),
			wantErr:    true,
			errContain: "not valid against metaschema",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			schema, err := schemavalidate.CompileSchemaFromPB(tt.schemaPB)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Nil(t, schema)
				return
			}

			require.NoError(t, err)
			if tt.schemaPB == nil {
				assert.Nil(t, schema)
			} else {
				assert.NotNil(t, schema)
			}
		})
	}
}

func TestCompileSchemaFromMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		schemaMap   map[string]any
		wantErr     bool
		errContains []string
	}{
		{
			name: "valid map schema compiles successfully",
			schemaMap: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"port": map[string]any{
						"type":    "integer",
						"minimum": 1,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "malformed schema fails during compilation",
			schemaMap: map[string]any{
				"$id": "://bad-url",
			},
			wantErr:     true,
			errContains: []string{"error in parsing id"},
		},
		{
			name: "schema invalid against metaschema returns error from Compile",
			schemaMap: map[string]any{
				"type": []any{"invalid", 123},
			},
			wantErr:     true,
			errContains: []string{"not valid against metaschema"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			schema, err := schemavalidate.CompileSchemaFromMap(tt.schemaMap)
			if tt.wantErr {
				require.Error(t, err)
				for _, exp := range tt.errContains {
					assert.Contains(t, err.Error(), exp)
				}
				assert.Nil(t, schema)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, schema)
		})
	}
}

func TestValidateAgainstSchema(t *testing.T) {
	t.Parallel()

	schemaMap := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"username": map[string]any{
				"type": "string",
			},
			"retries": map[string]any{
				"type":    "integer",
				"minimum": 0,
			},
		},
		"required":             []any{"username"},
		"additionalProperties": false,
	}

	schema, err := schemavalidate.CompileSchemaFromMap(schemaMap)
	require.NoError(t, err)
	require.NotNil(t, schema)

	tests := []struct {
		name        string
		inputObj    map[string]any
		wantErr     bool
		errContains []string
	}{
		{
			name: "valid payload matching schema passes validation",
			inputObj: map[string]any{
				"username": "minder-user",
				"retries":  3,
			},
			wantErr: false,
		},
		{
			name: "missing required property returns validation error with details",
			inputObj: map[string]any{
				"retries": 1,
			},
			wantErr:     true,
			errContains: []string{"invalid json schema", "missing property 'username'"},
		},
		{
			name: "disallowed property returns validation error with details",
			inputObj: map[string]any{
				"username": "minder-user",
				"extra":    "disallowed",
			},
			wantErr:     true,
			errContains: []string{"invalid json schema", "additional properties 'extra' not allowed"},
		},
		{
			name: "type mismatch returns validation error with details",
			inputObj: map[string]any{
				"username": 12345,
			},
			wantErr:     true,
			errContains: []string{"invalid json schema", "got number, want string"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := schemavalidate.ValidateAgainstSchema(schema, tt.inputObj)
			if tt.wantErr {
				require.Error(t, err)
				for _, exp := range tt.errContains {
					assert.Contains(t, err.Error(), exp)
				}
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	t.Parallel()

	rawSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"timeout": map[string]any{
				"type":    "string",
				"default": "30s",
			},
			"max_retries": map[string]any{
				"type":    "integer",
				"default": 5,
			},
			"enabled": map[string]any{
				"type":    "boolean",
				"default": true,
			},
			"nested": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sub_timeout": map[string]any{
						"type":    "string",
						"default": "10s",
					},
					"inner": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"endpoint": map[string]any{
								"type":    "string",
								"default": "https://api.example.com",
							},
						},
					},
				},
			},
			"non_object_branch": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"param": map[string]any{
						"type":    "string",
						"default": "fallback",
					},
				},
			},
		},
	}

	compiledSchema, err := schemavalidate.CompileSchemaFromMap(rawSchema)
	require.NoError(t, err)
	require.NotNil(t, compiledSchema)

	t.Run("applies defaults to empty map including nested structures", func(t *testing.T) {
		t.Parallel()

		obj := map[string]any{
			"nested": map[string]any{
				"inner": map[string]any{},
			},
		}

		schemavalidate.ApplyDefaults(compiledSchema, obj)

		assert.Equal(t, "30s", obj["timeout"])
		assert.Equal(t, 5, obj["max_retries"])
		assert.Equal(t, true, obj["enabled"])

		nested, ok := obj["nested"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "10s", nested["sub_timeout"])

		inner, ok := nested["inner"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://api.example.com", inner["endpoint"])
	})

	t.Run("does not overwrite already existing properties", func(t *testing.T) {
		t.Parallel()

		obj := map[string]any{
			"timeout":     "60s",
			"max_retries": 10,
			"enabled":     false,
			"nested": map[string]any{
				"sub_timeout": "25s",
				"inner": map[string]any{
					"endpoint": "https://custom.endpoint.com",
				},
			},
		}

		schemavalidate.ApplyDefaults(compiledSchema, obj)

		assert.Equal(t, "60s", obj["timeout"])
		assert.Equal(t, 10, obj["max_retries"])
		assert.Equal(t, false, obj["enabled"])

		nested, ok := obj["nested"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "25s", nested["sub_timeout"])

		inner, ok := nested["inner"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "https://custom.endpoint.com", inner["endpoint"])
	})

	t.Run("gracefully skips non-object values when schema expects nested object", func(t *testing.T) {
		t.Parallel()

		// "non_object_branch" has properties in schema, but the object contains a primitive string
		obj := map[string]any{
			"non_object_branch": "primitive-string-instead-of-map",
		}

		// Should not panic, gracefully skip nested traversal
		schemavalidate.ApplyDefaults(compiledSchema, obj)

		assert.Equal(t, "primitive-string-instead-of-map", obj["non_object_branch"])
		assert.Equal(t, "30s", obj["timeout"])
	})
}

func mustStructpbFromMap(t *testing.T, m map[string]any) *structpb.Struct {
	t.Helper()
	data, err := json.Marshal(m)
	require.NoError(t, err)

	st := &structpb.Struct{}
	err = json.Unmarshal(data, st)
	require.NoError(t, err)
	return st
}
