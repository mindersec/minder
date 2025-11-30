// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package vulncheck

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
		// vulncheck template
		{
			name:    "vulncheck template",
			msg:     "this is the message",
			tmpl:    templates.VulncheckTemplate,
			args:    map[string]any{"packages": []string{"boto3", "urllib3", "python-oauth2"}},
			error:   "evaluation failure: this is the message",
			details: "Vulnerable packages found:\n* boto3\n* urllib3\n* python-oauth2\n",
		},
	}

	for _, tt := range tests {

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
