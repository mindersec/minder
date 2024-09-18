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
	"context"
	"reflect"
	"sort"
	"testing"
)

func TestFindMixedScripts(t *testing.T) {
	t.Parallel()

	// Dummy script data
	mse := &MixedScriptsProcessor{
		runeToScript: map[rune]string{
			'A': "Latin",
			'o': "Latin",
			'Б': "Cyrillic",
			' ': "Common",
			'.': "Common",
		},
	}

	tests := []struct {
		description string
		line        string
		expected    []*Violation
	}{
		{
			description: "No mixed scripts",
			line:        "Hello World.",
			expected:    []*Violation{},
		},
		{
			description: "Mixed scripts in one word",
			line:        "Hello Бorld.",
			expected: []*Violation{
				{
					mixedScript: &MixedScriptInfo{
						text:         "Бorld.",
						scriptsFound: []string{"Cyrillic", "Latin"},
					},
				},
			},
		},
		{
			description: "Multiple words with mixed scripts",
			line:        "AБ AБ.",
			expected: []*Violation{
				{
					mixedScript: &MixedScriptInfo{
						text:         "AБ",
						scriptsFound: []string{"Cyrillic", "Latin"},
					},
				},
				{
					mixedScript: &MixedScriptInfo{
						text:         "AБ.",
						scriptsFound: []string{"Cyrillic", "Latin"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			got := mse.FindMixedScripts(tt.line)
			for i := range got {
				sort.Strings(got[i].mixedScript.scriptsFound)
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("FindMixedScripts() got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoadScriptData(t *testing.T) {
	t.Parallel()

	gotMap, err := loadScriptData(context.Background(), "resources/scripts.txt")
	if err != nil {
		t.Fatalf("loadScriptData returned an error: %v", err)
	}
	if len(gotMap) == 0 {
		t.Errorf("loadScriptData returned an empty map, want non-empty")
	}

	knownEntries := map[rune]string{
		'A': "Latin",
		'Б': "Cyrillic",
		'ᠮ': "Mongolian",
		'𔐀': "Anatolian_Hieroglyphs",
		'ⴰ': "Tifinagh",
	}

	for r, script := range knownEntries {
		if gotMap[r] != script {
			t.Errorf("loadScriptData gotMap[%U] = %v, want %v", r, gotMap[r], script)
		}
	}
}

func TestStringToRune(t *testing.T) {
	t.Parallel()

	tests := []struct {
		description string
		hexStr      string
		expected    rune
		wantErr     bool
	}{
		{
			description: "Valid hex for letter A",
			hexStr:      "41",
			expected:    'A',
			wantErr:     false,
		},
		{
			description: "Valid hex for emoji",
			hexStr:      "1F600",
			expected:    '😀',
			wantErr:     false,
		},
		{
			description: "Invalid hex string",
			hexStr:      "GHIJK",
			expected:    0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.description, func(t *testing.T) {
			t.Parallel()

			result, err := stringToRune(tt.hexStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("stringToRune(%q) error = %v, wantErr %v", tt.hexStr, err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("stringToRune(%q) = %v, want %v", tt.hexStr, result, tt.expected)
			}
		})
	}
}
