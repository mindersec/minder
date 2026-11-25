// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rule_type

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const regoRuleType = `# METADATA
# title: Rego rule type
# description: A rule type written in Rego
# custom:
#   def:
#     in_entity: repository
#     ingest:
#       type: git
package minder

import rego.v1

default allow := false

allow if input.allowed
`

func TestLintRegoRuleTypes(t *testing.T) {
	t.Parallel()

	helper := "package helpers\n\nimport rego.v1\n\nx := 1\n"
	annotatedHelper := "# METADATA\n# title: Shared helpers\n# custom:\n#   owner: platform\n" + helper

	tests := []struct {
		name    string
		files   map[string]string
		target  string
		wantErr string
		wantOut string
	}{
		{
			name:   "accepts a valid rego rule type",
			files:  map[string]string{"my_rule.rego": regoRuleType},
			target: "my_rule.rego",
		},
		{
			name:    "checks the name against the file name",
			files:   map[string]string{"other.rego": "# METADATA\n# name: my_rule\n" + regoRuleType[len("# METADATA\n"):]},
			target:  "other.rego",
			wantErr: "rule type name does not match file name: my_rule != other",
		},
		{
			name:    "fails on invalid rego",
			files:   map[string]string{"my_rule.rego": regoRuleType + "\nbroken {\n"},
			target:  "my_rule.rego",
			wantErr: "failed parsing rego rule",
		},
		{
			name:    "fails on a rego file without metadata when named directly",
			files:   map[string]string{"helpers.rego": helper},
			target:  "helpers.rego",
			wantErr: "could not find metadata in Rego file",
		},
		{
			name:    "skips a rego file without metadata found in a directory",
			files:   map[string]string{"helpers.rego": helper, "my_rule.rego": regoRuleType},
			wantOut: "not a rule type",
		},
		{
			name:    "skips an annotated helper found in a directory",
			files:   map[string]string{"helpers.rego": annotatedHelper, "my_rule.rego": regoRuleType},
			wantOut: "helpers.rego: not a rule type",
		},
		{
			name:    "fails on an annotated helper when named directly",
			files:   map[string]string{"helpers.rego": annotatedHelper},
			target:  "helpers.rego",
			wantErr: `unknown field "owner"`,
		},
		{
			name: "fails on a rule type missing its definition in a directory",
			files: map[string]string{
				"my_rule.rego": "# METADATA\n# title: Unfinished\n# custom:\n#   guidance: Fix it\n" + helper,
			},
			wantErr: "invalid rule type definition",
		},
		{
			name:    "fails on malformed metadata in a directory",
			files:   map[string]string{"my_rule.rego": "# METADATA\n# title: [unclosed\n" + helper},
			wantErr: "error decoding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			for name, content := range tt.files {
				require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
			}

			var out bytes.Buffer
			cmd := CmdLint()
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs([]string{"-r", filepath.Join(dir, tt.target)})
			err := cmd.Execute()

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, out.String(), tt.wantErr)
				return
			}
			require.NoError(t, err, out.String())
			require.Contains(t, out.String(), tt.wantOut)
		})
	}
}
