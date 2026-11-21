// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package artifact

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  map[string]any
		wantErr string
	}{
		{name: "nil params", params: nil},
		{name: "empty params", params: map[string]any{}},
		{name: "tags only", params: map[string]any{"tags": []any{"latest", "v1"}}},
		{name: "valid tag_regex", params: map[string]any{"tag_regex": "^v[0-9]+$"}},
		{
			name:    "tags and tag_regex",
			params:  map[string]any{"tags": []any{"latest"}, "tag_regex": "^v"},
			wantErr: "cannot specify both tags and tag_regex",
		},
		{
			name:    "empty tag",
			params:  map[string]any{"tags": []any{""}},
			wantErr: "cannot specify empty tag",
		},
		{
			name:    "invalid tag_regex",
			params:  map[string]any{"tag_regex": "("},
			wantErr: "error compiling tag regex",
		},
		{
			name:    "tag_regex too long",
			params:  map[string]any{"tag_regex": strings.Repeat("a", 513)},
			wantErr: "limited to 512 characters",
		},
		{
			name:    "undecodable params",
			params:  map[string]any{"tags": 42},
			wantErr: "error decoding ingester config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateParams(tt.params)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
