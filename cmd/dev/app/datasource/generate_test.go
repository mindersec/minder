// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasource

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateOpName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method   string
		path     string
		expected string
	}{
		{"GET", "/users", "get_users"},
		{"POST", "/users/{id}", "post_users_id_"},
		{"DELETE", "/users/{id}/roles", "delete_users_id_roles"},
	}

	for _, tt := range tests {
		t.Run(tt.method+"_"+tt.path, func(t *testing.T) {
			t.Parallel()
			got := generateOpName(tt.method, tt.path)
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestValidParseFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		parseFormat string
		wantErr     bool
	}{
		{"valid json", "json", false},
		{"valid empty", "", false},
		{"invalid xml", "xml", true},
		{"invalid text", "text", true},
		{"invalid random", "banana", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validParseFormat(tt.parseFormat)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
