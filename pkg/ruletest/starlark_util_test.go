// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestStarlarkValueToGo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    starlark.Value
		expected any
		err      bool
	}{
		{
			name:     "none",
			input:    starlark.None,
			expected: nil,
		},
		{
			name:     "bool true",
			input:    starlark.True,
			expected: true,
		},
		{
			name:     "bool false",
			input:    starlark.False,
			expected: false,
		},
		{
			name:     "int",
			input:    starlark.MakeInt(42),
			expected: int64(42),
		},
		{
			name:     "float",
			input:    starlark.Float(3.14),
			expected: float64(3.14),
		},
		{
			name:     "string",
			input:    starlark.String("hello"),
			expected: "hello",
		},
		{
			name: "list",
			input: starlark.NewList([]starlark.Value{
				starlark.String("a"),
				starlark.MakeInt(1),
			}),
			expected: []any{"a", int64(1)},
		},
		{
			name: "dict",
			input: func() starlark.Value {
				d := starlark.NewDict(2)
				_ = d.SetKey(starlark.String("key1"), starlark.String("val1"))
				_ = d.SetKey(starlark.String("key2"), starlark.MakeInt(2))
				return d
			}(),
			expected: map[string]any{
				"key1": "val1",
				"key2": int64(2),
			},
		},
		{
			name: "nested dict",
			input: func() starlark.Value {
				inner := starlark.NewDict(1)
				_ = inner.SetKey(starlark.String("innerKey"), starlark.String("innerVal"))

				outer := starlark.NewDict(1)
				_ = outer.SetKey(starlark.String("outerKey"), inner)
				return outer
			}(),
			expected: map[string]any{
				"outerKey": map[string]any{
					"innerKey": "innerVal",
				},
			},
		},
		{
			name: "dict with non-string key",
			input: func() starlark.Value {
				d := starlark.NewDict(1)
				_ = d.SetKey(starlark.MakeInt(1), starlark.String("val"))
				return d
			}(),
			expected: nil,
			err:      true,
		},
		{
			name:     "bytes unsupported",
			input:    starlark.Bytes("unsupported"),
			expected: nil,
			err:      true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			res, err := starlarkValueToGo(tc.input)
			if tc.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, res)
			}
		})
	}
}
