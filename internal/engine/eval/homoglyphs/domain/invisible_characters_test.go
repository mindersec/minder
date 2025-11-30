// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"reflect"
	"testing"

	"github.com/mindersec/minder/internal/engine/eval/homoglyphs/domain/resources"
)

func TestFindInvisibleCharacters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description string
		line        string
		expected    []*Violation
	}{
		{
			description: "No invisible characters",
			line:        "Hello, World!",
			expected:    []*Violation{},
		},
		{
			description: "Contains Zero Width Space",
			line:        "Hello,\u200BWorld!",
			expected: []*Violation{
				{
					InvisibleChar: '\u200B',
				},
			},
		},
		{
			description: "Multiple invisible characters",
			line:        "Invisible\u200BText\u200C\u200D",
			expected: []*Violation{
				{
					InvisibleChar: '\u200B',
				},
				{
					InvisibleChar: '\u200C',
				},
				{
					InvisibleChar: '\u200D',
				},
			},
		},
	}

	processor := &InvisibleCharactersProcessor{
		invisibleCharacters: resources.InvisibleCharacters,
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			result := processor.FindInvisibleCharacters(tt.line)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("FindInvisibleCharacters(%q) = %v, want %v", tt.line, result, tt.expected)
			}
		})
	}
}
