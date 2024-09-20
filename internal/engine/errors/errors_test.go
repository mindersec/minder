// Copyright 2024 Stacklok, Inc.
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

package errors

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/engine/eval/templates"
)

func TestLegacyEvaluationDetailRendering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		msg     string
		args    []any
		error   string
		details string
	}{
		{
			name:    "legacy",
			msg:     "format: %s",
			args:    []any{"this is the message"},
			error:   "evaluation failure: format: this is the message",
			details: "format: this is the message",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := NewErrEvaluationFailed(
				tt.msg,
				tt.args...,
			)

			require.Equal(t, tt.error, err.Error())
			evalErr, ok := err.(*EvaluationError)
			require.True(t, ok)
			require.Equal(t, tt.details, evalErr.Msg)
		})
	}
}

func TestEvaluationDetailRendering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		msg     string
		msgArgs []any
		tmpl    string
		args    any
		error   string
		details string
	}{
		{
			name:    "legacy",
			msg:     "this is the message",
			tmpl:    "",
			args:    nil,
			error:   "evaluation failure: this is the message",
			details: "this is the message",
		},
		{
			name:    "empty template",
			msg:     "this is the message",
			tmpl:    "",
			args:    nil,
			error:   "evaluation failure: this is the message",
			details: "this is the message",
		},
		{
			name:    "simple template",
			msg:     "this is the message",
			tmpl:    "fancy template with {{ . }}",
			args:    "fancy message",
			error:   "evaluation failure: this is the message",
			details: "fancy template with fancy message",
		},
		{
			name:    "complex template",
			msg:     "this is the message",
			tmpl:    "fancy template with {{ range $idx, $val := . }}{{ if $idx }}, {{ end }}{{ . }}{{ end }}",
			args:    []any{"many", "many", "many messages"},
			error:   "evaluation failure: this is the message",
			details: "fancy template with many, many, many messages",
		},
		{
			name:    "enforced limit",
			msg:     "this is the message",
			tmpl:    "fancy template with {{ . }}",
			args:    strings.Repeat("A", 1025),
			error:   "evaluation failure: this is the message",
			details: "evaluation failure: this is the message",
		},
		// vulncheck template
		{
			name:    "vulncheck template",
			msg:     "this is the message",
			tmpl:    templates.VulncheckTemplate,
			args:    map[string]any{"packages": []string{"boto3", "urllib3", "python-oauth2"}},
			error:   "evaluation failure: this is the message",
			details: "Vulnerable packages found:\n* `boto3`\n* `urllib3`\n* `python-oauth2`\n",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := NewDetailedErrEvaluationFailed(
				tt.tmpl,
				tt.args,
				tt.msg,
				tt.msgArgs...,
			)

			require.Equal(t, tt.error, err.Error())
			evalErr, ok := err.(*EvaluationError)
			require.True(t, ok)
			require.Equal(t, tt.details, evalErr.Details())
			require.LessOrEqual(t, len(evalErr.Details()), 1024)
		})
	}
}
