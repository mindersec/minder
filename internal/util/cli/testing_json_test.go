// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTryParseJSONStream_Stream tests parsing of newline-delimited JSON (the main use case).
func TestTryParseJSONStream_Stream(t *testing.T) {
	t.Parallel()
	input := `{"id":1}
{"id":2}`

	result, err := tryParseJSONStream(input)
	require.NoError(t, err)
	require.Len(t, result, 2)

	// Verify actual content was parsed correctly
	obj1 := result[0].(map[string]interface{})
	require.Equal(t, float64(1), obj1["id"])

	obj2 := result[1].(map[string]interface{})
	require.Equal(t, float64(2), obj2["id"])
}

// TestTryParseJSONStream_SingleObject tests parsing of a single JSON object (backward compatibility).
func TestTryParseJSONStream_SingleObject(t *testing.T) {
	t.Parallel()
	input := `{"id":1}`

	result, err := tryParseJSONStream(input)
	require.NoError(t, err)
	require.Len(t, result, 1)

	// Verify actual content was parsed correctly
	obj := result[0].(map[string]interface{})
	require.Equal(t, float64(1), obj["id"])
}

// TestTryParseJSONStream_NonJSON tests that non-JSON input (like YAML) is rejected.
func TestTryParseJSONStream_NonJSON(t *testing.T) {
	t.Parallel()
	input := `
name: sachin
age: 22
`

	_, err := tryParseJSONStream(input)
	require.Error(t, err)
}

// TestTryParseJSONStream_EmptyArray tests the edge case of empty JSON array.
func TestTryParseJSONStream_EmptyArray(t *testing.T) {
	t.Parallel()
	input := `[]`

	result, err := tryParseJSONStream(input)
	require.NoError(t, err)
	require.Len(t, result, 1) // decoder treats [] as one decoded value
}

// TestTryParseJSONStream_InvalidJSON tests that malformed JSON is properly rejected.
func TestTryParseJSONStream_InvalidJSON(t *testing.T) {
	t.Parallel()
	input := `{"id":1`

	_, err := tryParseJSONStream(input)
	require.Error(t, err)
}
