// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package jsonyaml_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/util/jsonyaml"
)

func TestTranscodeYAMLToJSON(t *testing.T) {
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
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := strings.NewReader(tt.yamlCase)
			w := bytes.NewBuffer(nil)

			err := jsonyaml.TranscodeYAMLToJSON(r, w)

			if tt.wantErr {
				assert.Error(t, err, "expected error")
				return
			}

			require.NoError(t, err, "unexpected error")
			assert.Equal(t, tt.wantW, w.String(), "unexpected output")
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
