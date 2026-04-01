// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package labels

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLabelFilter(t *testing.T) {
	tests := []struct {
		name          string
		filter        string
		expectedInc   []string
		expectedExc   []string
		expectedError error
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
			name:          "invalid star exclude",
			filter:        "!*",
			expectedError: ErrInvalidLabel,
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
			name:          "star mixed with includes",
			filter:        "foo,*",
			expectedError: ErrInvalidLabel,
		},
		{
			name:          "includes mixed with star",
			filter:        "*,foo",
			expectedError: ErrInvalidLabel,
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
		t.Run(tt.name, func(t *testing.T) {
			inc, exc, err := ParseLabelFilter(tt.filter)
			if tt.expectedError != nil {
				require.Error(t, err)
				require.True(t, errors.Is(err, tt.expectedError))
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedInc, inc)
				require.Equal(t, tt.expectedExc, exc)
			}
		})
	}
}
