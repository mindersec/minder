// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package labels

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLabelFilter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		filter      string
		expectedInc []string
		expectedExc []string
	}{
		{
			name:        "empty",
			filter:      "",
			expectedInc: nil,
			expectedExc: nil,
		},
		{
			name:        "single include",
			filter:      "foo",
			expectedInc: []string{"foo"},
			expectedExc: nil,
		},
		{
			name:        "single exclude",
			filter:      "!foo",
			expectedInc: nil,
			expectedExc: []string{"foo"},
		},
		{
			name:        "star include",
			filter:      "*",
			expectedInc: []string{"*"},
			expectedExc: nil,
		},
		{
			name:        "star exclude",
			filter:      "!*",
			expectedInc: nil,
			expectedExc: []string{"*"},
		},
		{
			name:        "multiple includes",
			filter:      "foo,bar",
			expectedInc: []string{"foo", "bar"},
			expectedExc: nil,
		},
		{
			name:        "includes and excludes",
			filter:      "foo,!bar,baz,!qux",
			expectedInc: []string{"foo", "baz"},
			expectedExc: []string{"bar", "qux"},
		},
		{
			name:        "star mixed with includes",
			filter:      "foo,*",
			expectedInc: []string{"*"},
			expectedExc: nil,
		},
		{
			name:        "includes mixed with star",
			filter:      "*,foo",
			expectedInc: []string{"*"},
			expectedExc: nil,
		},
		{
			name:        "star and excludes",
			filter:      "*,!foo",
			expectedInc: []string{"*"},
			expectedExc: []string{"foo"},
		},
		{
			name:        "whitespace handling",
			filter:      " foo , !bar ",
			expectedInc: []string{"foo"},
			expectedExc: []string{"bar"},
		},
		{
			name:        "trailing commas",
			filter:      "foo,,!bar,",
			expectedInc: []string{"foo"},
			expectedExc: []string{"bar"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			inc, exc, err := ParseLabelFilter(tt.filter)
			require.NoError(t, err)
			require.Equal(t, tt.expectedInc, inc)
			require.Equal(t, tt.expectedExc, exc)
		})
	}
}
