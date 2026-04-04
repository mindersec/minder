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
			name:   "empty",
			filter: "",
		},
		{
			name:        "single include",
			filter:      "foo",
			expectedInc: []string{"foo"},
		},
		{
			name:        "single exclude",
			filter:      "!foo",
			expectedExc: []string{"foo"},
		},
		{
			name:        "star include",
			filter:      "*",
			expectedInc: []string{"*"},
		},
		{
			name:        "star exclude",
			filter:      "!*",
			expectedExc: []string{"*"},
		},
		{
			name:        "multiple includes",
			filter:      "foo,bar",
			expectedInc: []string{"foo", "bar"},
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
		},
		{
			name:        "includes mixed with star",
			filter:      "*,foo",
			expectedInc: []string{"*"},
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
			inc, exc := ParseLabelFilter(tt.filter)
			require.Equal(t, tt.expectedInc, inc)
			require.Equal(t, tt.expectedExc, exc)
		})
	}
}
