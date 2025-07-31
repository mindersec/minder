// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package rego provides the rego rule evaluator
package rego_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/engine/eval/rego"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
)

func TestLimitedDialer(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": 1}`))
	}))
	t.Cleanup(ts.Close)

	ruleDef := `
		package minder
		import rego.v1

		default allow := false
		resp := http.send({"url": "%s", "method": "GET", "raise_error": false})
		allow if {
		  not resp.error
		}
		message := resp.error.message
		`

	tests := []struct {
		name    string
		url     string
		wantErr string
	}{{
		name:    "test blocked fetch by name",
		url:     ts.URL,
		wantErr: "remote address is not public",
	}, {
		name: "google.com not blocked",
		url:  "http://www.google.com",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			eval, err := rego.NewRegoEvaluator(
				&minderv1.RuleType_Definition_Eval_Rego{
					Type: rego.DenyByDefaultEvaluationType.String(),
					Def:  fmt.Sprintf(ruleDef, tt.url),
				},
			)
			require.NoError(t, err, "could not create evaluator")

			emptyPol := map[string]any{}

			res, err := eval.Eval(context.Background(), emptyPol, nil, &interfaces.Ingested{})

			if tt.wantErr == "" {
				require.NoError(t, err, "expected no error")
				require.NotNil(t, res, "expected a result")
				return
			}

			require.Equal(t, &interfaces.EvaluationResult{
				Output: `Get "` + tt.url + `": remote address is not public`,
			}, res)
			require.ErrorIs(t, err, interfaces.ErrEvaluationFailed)
			detailErr := err.(interfaces.EvalError)
			require.Contains(t, detailErr.Details(), tt.wantErr)
		})
	}
}
