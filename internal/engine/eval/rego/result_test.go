// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rego

import (
	"testing"

	"github.com/stretchr/testify/require"

	evalerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval/templates"
)

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
		// rego/deny-by-default template
		{
			name:    "deny by default template",
			msg:     "this is the message",
			tmpl:    templates.RegoDenyByDefaultTemplate,
			args:    map[string]any{"message": "bar", "entityName": ""},
			error:   "evaluation failure: this is the message",
			details: "bar",
		},
		{
			name:    "optional entity name",
			msg:     "this is the message",
			tmpl:    templates.RegoDenyByDefaultTemplate,
			args:    map[string]any{"message": "bar", "entityName": "baz"},
			error:   "evaluation failure: this is the message",
			details: "bar for baz",
		},
		{
			name:    "status mandatory",
			msg:     "this is the message",
			tmpl:    templates.RegoDenyByDefaultTemplate,
			args:    map[string]any{"message": "bar"},
			error:   "evaluation failure: this is the message",
			details: "evaluation failure: this is the message",
		},
		{
			name:    "message mandatory",
			msg:     "this is the message",
			tmpl:    templates.RegoDenyByDefaultTemplate,
			args:    map[string]any{"status": "foo"},
			error:   "evaluation failure: this is the message",
			details: "evaluation failure: this is the message",
		},

		// rego/constraints template
		{
			name:    "constraints template single failures",
			msg:     "this is the message",
			tmpl:    templates.RegoConstraints,
			args:    map[string]any{"violations": []string{"sole violation"}},
			error:   "evaluation failure: this is the message",
			details: "sole violation\n",
		},
		{
			name:    "constraints template multiple failures",
			msg:     "this is the message",
			tmpl:    templates.RegoConstraints,
			args:    map[string]any{"violations": []string{"first violation", "second violation"}},
			error:   "evaluation failure: this is the message",
			details: "Multiple issues:\n* first violation\n* second violation\n",
		},
		{
			name:    "violations mandatory",
			msg:     "this is the message",
			tmpl:    templates.RegoConstraints,
			args:    map[string]any{},
			error:   "evaluation failure: this is the message",
			details: "evaluation failure: this is the message",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := evalerrors.NewDetailedErrEvaluationFailed(
				tt.tmpl,
				tt.args,
				tt.msg,
				tt.msgArgs...,
			)

			require.Equal(t, tt.error, err.Error())
			evalErr, ok := err.(*evalerrors.EvaluationError)
			require.True(t, ok)
			require.Equal(t, tt.details, evalErr.Details())
		})
	}
}
