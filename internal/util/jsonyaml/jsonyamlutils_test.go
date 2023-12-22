//
// Copyright 2023 Stacklok, Inc.
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

package jsonyaml_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stacklok/minder/internal/util/jsonyaml"
)

func TestConvertYAMLToJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yamlCase string
		wantW    string
		wantErr  bool
	}{
		{
			name:     "simple yaml",
			yamlCase: "foo: bar",
			wantW:    "{\"foo\":\"bar\"}\n",
			wantErr:  false,
		},
		{
			name: "complex yaml",
			yamlCase: `---
foo: bar
bar:
  - foo
  - bar
  - baz
`,
			wantW:   "{\"bar\":[\"foo\",\"bar\",\"baz\"],\"foo\":\"bar\"}\n",
			wantErr: false,
		},
		{
			name:     "invalid yaml",
			yamlCase: "\tThis is invalid yaml",
			wantW:    "",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := jsonyaml.ConvertYamlToJson(tt.yamlCase)

			if tt.wantErr {
				assert.Error(t, err, "expected error")
				return
			}

			assert.NoError(t, err, "unexpected error")
			assert.Equal(t, tt.wantW, string(got), "unexpected output")
		})
	}
}

func TestConvertJSONToYAML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		jsonCase string
		wantW    string
		wantErr  bool
	}{
		{
			name:     "simple yaml",
			jsonCase: "{\"foo\":\"bar\"}\n",
			wantW:    "foo: bar\n",
			wantErr:  false,
		},
		{
			name:     "complex yaml",
			jsonCase: "{\"bar\":[\"foo\",\"bar\",\"baz\"],\"foo\":\"bar\"}\n",
			wantW: `bar:
    - foo
    - bar
    - baz
foo: bar
`,
			wantErr: false,
		},
		{
			name:     "invalid yaml",
			jsonCase: "This is invalid JSON",
			wantW:    "",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := jsonyaml.ConvertJsonToYaml([]byte(tt.jsonCase))

			if tt.wantErr {
				assert.Error(t, err, "expected error")
				return
			}

			assert.NoError(t, err, "unexpected error")
			assert.Equal(t, tt.wantW, string(got), "unexpected output")
		})
	}
}
