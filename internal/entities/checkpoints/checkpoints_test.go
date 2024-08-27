// Copyright 2024 Stacklok, Inc.
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

// Package checkpoints contains logic relating to checkpoint management for entities
package checkpoints

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckpointEnvelopeV1_MarshalJSON(t *testing.T) {
	t.Parallel()

	// Create a fixed timestamp for consistent test results
	timestamp := time.Date(2023, 7, 31, 12, 0, 0, 0, time.UTC)

	// Test cases
	tests := []struct {
		name     string
		input    *CheckpointEnvelopeV1
		expected string
	}{
		{
			name: "all fields set",
			input: NewCheckpointV1(timestamp).
				WithCommitHash("abc123").
				WithBranch("main").
				WithVersion("v1.0.0").
				WithDigest("sha256:xyz"),
			expected: `{"version":"v1","checkpoint":{"timestamp":"2023-07-31T12:00:00Z","commitHash":"abc123","branch":"main","version":"v1.0.0","digest":"sha256:xyz"}}`,
		},
		{
			name:     "optional fields omitted",
			input:    NewCheckpointV1(timestamp),
			expected: `{"version":"v1","checkpoint":{"timestamp":"2023-07-31T12:00:00Z"}}`,
		},
		{
			name: "commit-related fields set",
			input: NewCheckpointV1(timestamp).
				WithCommitHash("abc123").
				WithBranch("main"),
			expected: `{"version":"v1","checkpoint":{"timestamp":"2023-07-31T12:00:00Z","commitHash":"abc123","branch":"main"}}`,
		},
		{
			name: "With HTTP info",
			input: NewCheckpointV1(timestamp).
				WithHTTP("http://example.com", "GET"),
			expected: `{"version":"v1","checkpoint":{"timestamp":"2023-07-31T12:00:00Z","httpURL":"http://example.com","httpMethod":"GET"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Marshal the input to JSON
			output, err := tt.input.ToJSON()
			require.NoError(t, err)

			assert.Equal(t, string(output), tt.expected)
		})
	}
}

func TestCheckpointEnvelopeV1_UnmarshalJson(t *testing.T) {
	t.Parallel()

	// Create a fixed timestamp for consistent test results
	timestamp := time.Date(2023, 7, 31, 12, 0, 0, 0, time.UTC)

	// Test cases
	tests := []struct {
		name     string
		input    string
		expected *CheckpointEnvelopeV1
	}{
		{
			name:  "all fields set",
			input: `{"version":"v1","checkpoint":{"timestamp":"2023-07-31T12:00:00Z","commitHash":"abc123","branch":"main","version":"v1.0.0","digest":"sha256:xyz"}}`,
			expected: NewCheckpointV1(timestamp).
				WithCommitHash("abc123").
				WithBranch("main").
				WithVersion("v1.0.0").
				WithDigest("sha256:xyz"),
		},
		{
			name:     "optional fields omitted",
			input:    `{"version":"v1","checkpoint":{"timestamp":"2023-07-31T12:00:00Z"}}`,
			expected: NewCheckpointV1(timestamp),
		},
		{
			name:  "commit-related fields set",
			input: `{"version":"v1","checkpoint":{"timestamp":"2023-07-31T12:00:00Z","commitHash":"abc123","branch":"main"}}`,
			expected: NewCheckpointV1(timestamp).
				WithCommitHash("abc123").
				WithBranch("main"),
		},
		{
			name:  "With HTTP info",
			input: `{"version":"v1","checkpoint":{"timestamp":"2023-07-31T12:00:00Z","httpURL":"http://example.com","httpMethod":"GET"}}`,
			expected: NewCheckpointV1(timestamp).
				WithHTTP("http://example.com", "GET"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Unmarshal the input to a CheckpointEnvelopeV1
			var output CheckpointEnvelopeV1
			err := json.Unmarshal([]byte(tt.input), &output)
			require.NoError(t, err)

			assert.Equal(t, output, *tt.expected)
		})
	}
}
