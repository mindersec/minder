// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
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
