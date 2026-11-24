// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rule_type

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	regoeval "github.com/mindersec/minder/internal/engine/eval/rego"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestValidateRegoRuleRequiresV1(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		def        string
		wantErr    string
		wantOutput bool
	}{
		{
			name: "accepts Rego V1",
			def: `package minder

import rego.v1

default allow := false

allow if {
	input.allowed
}`,
			wantOutput: true,
		},
		{
			name: "rejects Rego V0 with migration guidance",
			def: `package minder

default allow = false

allow {
	input.allowed
}`,
			wantErr: regoeval.V0MigrationMessage,
		},
		{
			name:    "rejects invalid Rego",
			def:     "package minder\n\nallow {",
			wantErr: "failed parsing rego rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			err := validateRegoRule(context.Background(), &minderv1.RuleType_Definition_Eval_Rego{
				Def: tt.def,
			}, "rule.rego", &output)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantOutput, output.Len() > 0)
		})
	}
}

func TestValidateRestRemediationBody(t *testing.T) {
	t.Parallel()

	body := func(s string) *string { return &s }

	tests := []struct {
		name    string
		rest    *minderv1.RestType
		wantErr string
	}{
		{
			name: "no rest remediation",
			rest: nil,
		},
		{
			name: "no body",
			rest: &minderv1.RestType{Endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}"},
		},
		{
			name: "static JSON object",
			rest: &minderv1.RestType{Body: body(`{"security_and_analysis": {"secret_scanning": {"status": "enabled"}}}`)},
		},
		{
			name: "templated body is only checked for template syntax",
			rest: &minderv1.RestType{Body: body(`{"allow_forking": {{ .Profile.allow_forking }}}`)},
		},
		{
			name:    "trailing comma",
			rest:    &minderv1.RestType{Body: body(`{"required_approving_review_count": 1,}`)},
			wantErr: "body is not a valid JSON object",
		},
		{
			name:    "JSON array instead of an object",
			rest:    &minderv1.RestType{Body: body(`["enabled"]`)},
			wantErr: "body is not a valid JSON object",
		},
		{
			name:    "unterminated template action",
			rest:    &minderv1.RestType{Body: body(`{"allow_forking": {{ .Profile.allow_forking }`)},
			wantErr: "body is not a valid template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateRestRemediationBody(tt.rest)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}
