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

package application

import (
	"testing"

	"github.com/stretchr/testify/require"

	evalerrors "github.com/stacklok/minder/internal/engine/errors"
	"github.com/stacklok/minder/internal/engine/eval/homoglyphs/domain"
	"github.com/stacklok/minder/internal/engine/eval/templates"
)

func TestInvisibleCharactersEvaluationDetailRendering(t *testing.T) {
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
		// invisible characters template
		{
			name: "invisible characters template with one violation",
			msg:  "this is the message",
			tmpl: templates.InvisibleCharactersTemplate,
			args: map[string]any{
				"violations": []*domain.Violation{
					{
						InvisibleChar: '\u200B',
					},
				},
			},
			error:   "evaluation failure: this is the message",
			details: "Invisible characters found:\n* U+200B",
		},
		{
			name: "invisible characters template with multiple violations",
			msg:  "this is the message",
			tmpl: templates.InvisibleCharactersTemplate,
			args: map[string]any{
				"violations": []*domain.Violation{
					{
						InvisibleChar: '\u200B',
					},
					{
						InvisibleChar: '\u200C',
					},
				},
			},
			error:   "evaluation failure: this is the message",
			details: "Invisible characters found:\n* U+200B\n* U+200C",
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
