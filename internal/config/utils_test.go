//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestGetKeysWithNullValueFromYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		yamlInput string
		want      []string
	}{
		{
			name: "Test with null values",
			yamlInput: `
key1: null
key2:
  subkey1: null
  subkey2: value
key3: [null, value]
`,
			want: []string{
				".key1",
				".key2.subkey1",
				".key3[0]",
			},
		},
		{
			name: "Test without null values",
			yamlInput: `
key1: value1
key2:
  subkey1: subvalue1
  subkey2: subvalue2
key3: [value1, value2]
`,
			want: []string{},
		},
		{
			name: "Test with highly nested null values",
			yamlInput: `
key1: value1
key2:
  subkey1: null
  subkey2: value2
  subkey3:
    subsubkey1: null
    subsubkey2: [value1, null, value2]
    subsubkey3:
      subsubsubkey1: [value1, value2, null]
      subsubsubkey2: null
key3: [value1, null, value2]
key4:
  subkey1: [value1, value2, null]
  subkey2:
    subsubkey1: null
    subsubkey2: [null, value1, value2]
    subsubkey3:
      subsubsubkey1: [value1, null, value2]
      subsubsubkey2: null
`,
			want: []string{
				".key2.subkey1",
				".key2.subkey3.subsubkey1",
				".key2.subkey3.subsubkey2[1]",
				".key2.subkey3.subsubkey3.subsubsubkey1[2]",
				".key2.subkey3.subsubkey3.subsubsubkey2",
				".key3[1]",
				".key4.subkey1[2]",
				".key4.subkey2.subsubkey1",
				".key4.subkey2.subsubkey2[0]",
				".key4.subkey2.subsubkey3.subsubsubkey1[1]",
				".key4.subkey2.subsubkey3.subsubsubkey2",
			},
		},
		{
			name: "Test with null values with integer, boolean, and null keys",
			yamlInput: `
key1: value1
true:
  null: null
  1: value2
  2:
    false: null
    3: [value1, null, value2]
    4:
      true: [value1, value2, null]
      5: null
key3: [value1, null, value2]
6:
  true: [value1, value2, null]
  7:
    false: null
    8: [null, value1, value2]
    9:
      true: [value1, null, value2]
      10: null
`,
			want: []string{
				".true.null",
				".true.2.false",
				".true.2.3[1]",
				".true.2.4.true[2]",
				".true.2.4.5",
				".key3[1]",
				".6.true[2]",
				".6.7.false",
				".6.7.8[0]",
				".6.7.9.true[1]",
				".6.7.9.10",
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var data interface{}
			err := yaml.Unmarshal([]byte(test.yamlInput), &data)
			if err != nil {
				t.Fatalf("Error parsing YAML: %v", err)
			}

			got := GetKeysWithNullValueFromYAML(data, "")
			assert.ElementsMatchf(t, got, test.want, "GetKeysWithNullValueFromYAML() = %v, want %v", got, test.want)
		})
	}
}

func TestGetRelevantCfgPath(t *testing.T) {
	t.Parallel()

	type args struct {
		paths []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test with empty paths",
			args: args{
				paths: []string{},
			},
			want: "",
		},
		{
			name: "Test with one empty path",
			args: args{
				paths: []string{""},
			},
			want: "",
		},
		{
			name: "Test with one non-empty path",
			args: args{
				paths: []string{"config.yaml"},
			},
			want: "config.yaml",
		},
		{
			name: "Test with multiple paths",
			args: args{
				paths: []string{"", "config.yaml", "config.yml", "config.json"},
			},
			want: "config.yaml",
		},
		{
			name: "Test with multiple paths with empty path in the middle",
			args: args{
				paths: []string{"config.yaml", "", "config.yml", "config.json"},
			},
			want: "config.yaml",
		},
		{
			name: "Test with multiple paths with empty path at the end",
			args: args{
				paths: []string{"config.yaml", "config.yml", "config.json", ""},
			},
			want: "config.yaml",
		},
		{
			name: "Test with multiple paths with empty path at the beginning",
			args: args{
				paths: []string{"", "config.yaml", "config.yml", "config.json"},
			},
			want: "config.yaml",
		},
		{
			name: "Test with multiple paths with all empty paths",
			args: args{
				paths: []string{"", "", "", ""},
			},
			want: "",
		},
		{
			name: "Test with multiple paths with all non-empty paths",
			args: args{
				paths: []string{"config.yaml", "config.yml", "config.json"},
			},
			want: "config.yaml",
		},
		{
			name: "Test with one non-empty path and all empty paths",
			args: args{
				paths: []string{"", "", "", "config.yaml"},
			},
			want: "config.yaml",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			baseDir := t.TempDir()
			createdpaths := []string{}
			for _, path := range tt.args.paths {
				if path != "" {
					f, err := os.Create(filepath.Clean(filepath.Join(baseDir, path)))
					require.NoError(t, err)
					createdpaths = append(createdpaths, f.Name())
				}
			}

			got := GetRelevantCfgPath(createdpaths)
			assert.Regexp(t, regexp.MustCompile("^.*"+tt.want+"$"), got)
		})
	}
}
