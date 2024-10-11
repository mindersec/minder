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

package rego

import (
	"testing"

	"github.com/stretchr/testify/require"

	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/eval/templates"
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
