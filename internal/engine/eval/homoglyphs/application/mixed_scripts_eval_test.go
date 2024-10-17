// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package application

import (
	"testing"

	"github.com/stretchr/testify/require"

	evalerrors "github.com/mindersec/minder/internal/engine/errors"
	"github.com/mindersec/minder/internal/engine/eval/homoglyphs/domain"
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
		// mixed scripts template
		{
			name: "mixed scripts template with one violation",
			msg:  "this is the message",
			tmpl: templates.MixedScriptsTemplate,
			args: map[string]any{
				"violations": []*domain.Violation{
					{
						MixedScript: &domain.MixedScriptInfo{
							Text:         "Бorld",
							ScriptsFound: []string{"Cyrillic", "Latin"},
						},
					},
				},
			},
			error:   "evaluation failure: this is the message",
			details: "Mixed scripts found:\n* Text: `Бorld`, Scripts: [Cyrillic, Latin]",
		},
		{
			name: "mixed scripts template with multiple violations",
			msg:  "this is the message",
			tmpl: templates.MixedScriptsTemplate,
			args: map[string]any{
				"violations": []*domain.Violation{
					{
						MixedScript: &domain.MixedScriptInfo{
							Text:         "Бorld",
							ScriptsFound: []string{"Cyrillic", "Latin"},
						},
					},
					{
						MixedScript: &domain.MixedScriptInfo{
							Text:         "Ѳ.HandleFunc(\"hi\",",
							ScriptsFound: []string{"Cyrillic", "Latin"},
						},
					},
				},
			},
			error: "evaluation failure: this is the message",
			details: "Mixed scripts found:\n* Text: `Бorld`, Scripts: [Cyrillic, Latin]\n" +
				"* Text: `Ѳ.HandleFunc(\"hi\",`, Scripts: [Cyrillic, Latin]",
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
