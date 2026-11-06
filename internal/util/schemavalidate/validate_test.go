// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package schemavalidate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestApplyDefaults_ScalarDefault is the baseline: a scalar default is applied
// and an explicit value is left alone.
func TestApplyDefaults_ScalarDefault(t *testing.T) {
	t.Parallel()

	schema, err := CompileSchemaFromMap(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"mode": map[string]any{"type": "string", "default": "strict"},
		},
	})
	require.NoError(t, err)

	obj := map[string]any{}
	ApplyDefaults(schema, obj)
	require.Equal(t, "strict", obj["mode"])

	explicit := map[string]any{"mode": "loose"}
	ApplyDefaults(schema, explicit)
	require.Equal(t, "loose", explicit["mode"])
}

// TestApplyDefaults_ObjectDefaultIsNotShared checks that an object-valued
// default is copied rather than handed out by reference. Sharing it lets one
// caller's mutation reach both the schema and every later caller.
func TestApplyDefaults_ObjectDefaultIsNotShared(t *testing.T) {
	t.Parallel()

	schema, err := CompileSchemaFromMap(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"params": map[string]any{
				"type":    "object",
				"default": map[string]any{"mode": "strict"},
				"properties": map[string]any{
					"mode": map[string]any{"type": "string"},
				},
			},
		},
	})
	require.NoError(t, err)

	first := map[string]any{}
	ApplyDefaults(schema, first)
	second := map[string]any{}
	ApplyDefaults(schema, second)

	firstParams, ok := first["params"].(map[string]any)
	require.True(t, ok)
	secondParams, ok := second["params"].(map[string]any)
	require.True(t, ok)

	require.Equal(t, "strict", firstParams["mode"])
	require.Equal(t, "strict", secondParams["mode"])

	// Mutating one caller's copy must not reach the other, nor the schema.
	firstParams["mode"] = "mutated"
	require.Equal(t, "strict", secondParams["mode"],
		"the second caller saw the first caller's mutation")

	third := map[string]any{}
	ApplyDefaults(schema, third)
	thirdParams, ok := third["params"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "strict", thirdParams["mode"],
		"the schema default was mutated, so later callers get the wrong value")
}

// TestApplyDefaults_ArrayDefaultIsNotShared is the same check for a slice.
func TestApplyDefaults_ArrayDefaultIsNotShared(t *testing.T) {
	t.Parallel()

	schema, err := CompileSchemaFromMap(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tags": map[string]any{
				"type":    "array",
				"default": []any{"a"},
			},
		},
	})
	require.NoError(t, err)

	first := map[string]any{}
	ApplyDefaults(schema, first)
	firstTags, ok := first["tags"].([]any)
	require.True(t, ok)
	firstTags[0] = "mutated"

	second := map[string]any{}
	ApplyDefaults(schema, second)
	secondTags, ok := second["tags"].([]any)
	require.True(t, ok)
	require.Equal(t, "a", secondTags[0],
		"the schema default slice was mutated through a previous caller")
}
