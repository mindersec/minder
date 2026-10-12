// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestConfigAsKeyValues(t *testing.T) {
	t.Parallel()

	cfgStruct, err := structpb.NewStruct(map[string]any{
		"endpoint": "https://api.github.com",
	})
	require.NoError(t, err)

	nestedCfgStruct, err := structpb.NewStruct(map[string]any{
		"auth": map[string]any{
			"token": "secret",
		},
	})
	require.NoError(t, err)

	tests := []struct {
		name     string
		provider *minderv1.Provider
		want     string
	}{
		{
			name:     "nil provider",
			provider: nil,
			want:     "",
		},
		{
			name:     "provider with nil config",
			provider: &minderv1.Provider{},
			want:     "",
		},
		{
			name: "provider with simple config",
			provider: &minderv1.Provider{
				Config: cfgStruct,
			},
			want: "endpoint=https://api.github.com",
		},
		{
			name: "provider with nested config",
			provider: &minderv1.Provider{
				Config: nestedCfgStruct,
			},
			want: "auth.token=secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := configAsKeyValues(tt.provider)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMapToKvPairs(t *testing.T) {
	t.Parallel()

	t.Run("nesting limit aborts", func(t *testing.T) {
		t.Parallel()
		var res []string
		m := map[string]any{"key": "value"}
		mapToKvPairs(m, "parent", &res, 11)
		assert.Empty(t, res)
	})

	t.Run("nested maps", func(t *testing.T) {
		t.Parallel()
		var res []string
		m := map[string]any{
			"top": map[string]any{
				"sub": "val",
			},
		}
		mapToKvPairs(m, "", &res, 0)
		assert.Equal(t, []string{"top.sub=val"}, res)
	})

	t.Run("multiple values", func(t *testing.T) {
		t.Parallel()
		var res []string
		m := map[string]any{
			"k1": "v1",
			"k2": 123,
			"k3": true,
		}
		mapToKvPairs(m, "base", &res, 0)
		assert.ElementsMatch(t, []string{
			"base.k1=v1",
			"base.k2=123",
			"base.k3=true",
		}, res)
	})
}
