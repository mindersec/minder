// Copyright 2023 Stacklok, Inc.
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

package domain

import (
	"reflect"
	"testing"

	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/domain/resources"
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
			//expected:    []rune{},
			expected: []*Violation{},
		},
		{
			description: "Contains Zero Width Space",
			line:        "Hello,\u200BWorld!",
			expected: []*Violation{
				{
					invisibleChar: '\u200B',
				},
			},
		},
		{
			description: "Multiple invisible characters",
			line:        "Invisible\u200BText\u200C\u200D",
			// expected:    []rune{'\u200B', '\u200C', '\u200D'},
			expected: []*Violation{
				{
					invisibleChar: '\u200B',
				},
				{
					invisibleChar: '\u200C',
				},
				{
					invisibleChar: '\u200D',
				},
			},
		},
	}

	processor := &InvisibleCharactersProcessor{
		invisibleCharacters: resources.InvisibleCharacters,
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			result := processor.FindInvisibleCharacters(tt.line)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("FindInvisibleCharacters(%q) = %v, want %v", tt.line, result, tt.expected)
			}
		})
	}
}
