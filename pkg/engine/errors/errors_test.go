// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0
package errors

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
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
		{
			name:    "template parse error falls back to Error()",
			msg:     "parse error fallback",
			tmpl:    "invalid {{ .unclosed",
			args:    "test",
			error:   "evaluation failure: parse error fallback",
			details: "evaluation failure: parse error fallback",
		},
		{
			name:    "template execution error on missing key falls back to Error()",
			msg:     "missing key execution error",
			tmpl:    "hello {{ .MissingKey }}",
			args:    struct{}{},
			error:   "evaluation failure: missing key execution error",
			details: "evaluation failure: missing key execution error",
		},
	}

	for _, tt := range tests {
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

func TestEvaluationError_Unwrap(t *testing.T) {
	t.Parallel()

	err := NewErrEvaluationFailed("rule evaluation check failed")
	require.Error(t, err)

	assert.True(t, errors.Is(err, interfaces.ErrEvaluationFailed), "EvaluationError must unwrap to interfaces.ErrEvaluationFailed")
	assert.False(t, errors.Is(err, interfaces.ErrEvaluationSkipped), "EvaluationError should not match ErrEvaluationSkipped")
}

func TestNewErrEvaluationSkipped(t *testing.T) {
	t.Parallel()

	err := NewErrEvaluationSkipped("entity %s not eligible for rule", "repo-123")
	require.Error(t, err)

	assert.Contains(t, err.Error(), "evaluation skipped: entity repo-123 not eligible for rule")
	assert.True(t, errors.Is(err, interfaces.ErrEvaluationSkipped), "error must wrap interfaces.ErrEvaluationSkipped")
	assert.False(t, errors.Is(err, interfaces.ErrEvaluationFailed))
}

func TestNewErrEvaluationSkipSilently(t *testing.T) {
	t.Parallel()

	err := NewErrEvaluationSkipSilently("skipping without user notification: %s", "upstream archived")
	require.Error(t, err)

	assert.Contains(t, err.Error(), "evaluation skipped silently: skipping without user notification: upstream archived")
	assert.True(t, errors.Is(err, ErrEvaluationSkipSilently))
	assert.False(t, errors.Is(err, interfaces.ErrEvaluationSkipped))
}

func TestRateLimitError(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("upstream provider github returned HTTP 429")
	reset := time.Unix(1730000000, 0)
	err := NewRateLimitError(baseErr, 5000, 0, reset)
	require.Error(t, err)

	var rateLimitErr *RateLimitError
	require.True(t, errors.As(err, &rateLimitErr))
	assert.Equal(t, int64(5000), rateLimitErr.Limit)
	assert.Equal(t, int64(0), rateLimitErr.Remaining)
	assert.Equal(t, reset, rateLimitErr.ResetTime)
	assert.Equal(t, baseErr, rateLimitErr.Unwrap())
	assert.True(t, errors.Is(err, baseErr), "errors.Is must recognize wrapped base error")
	assert.Contains(t, err.Error(), "rate limit exceeded: upstream provider github returned HTTP 429")
	assert.Contains(t, err.Error(), "limit: 5000, remaining: 0")
}

func TestLimitedWriter_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("negative capacity returns io.ErrShortBuffer", func(t *testing.T) {
		t.Parallel()
		var buf strings.Builder
		w := LimitedWriter(&buf, -1)
		n, err := w.Write([]byte("hello"))
		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, io.ErrShortBuffer)
		assert.Empty(t, buf.String())
	})

	t.Run("chunk larger than remaining capacity returns io.ErrShortBuffer without partial write", func(t *testing.T) {
		t.Parallel()
		var buf strings.Builder
		w := LimitedWriter(&buf, 5)

		n, err := w.Write([]byte("abc"))
		require.NoError(t, err)
		assert.Equal(t, 3, n)

		// Next chunk is 3 bytes, but remaining capacity is only 2 bytes
		n2, err2 := w.Write([]byte("def"))
		assert.Equal(t, 0, n2)
		assert.ErrorIs(t, err2, io.ErrShortBuffer)
		assert.Equal(t, "abc", buf.String(), "buffer must retain previous valid writes")
	})
}

func TestActionErrorsClassification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		err             error
		wantInformative bool
		wantFatal       bool
	}{
		{
			name:            "nil error",
			err:             nil,
			wantInformative: false,
			wantFatal:       false,
		},
		{
			name:            "ErrActionSkipped",
			err:             ErrActionSkipped,
			wantInformative: true,
			wantFatal:       false,
		},
		{
			name:            "wrapped ErrActionSkipped",
			err:             fmt.Errorf("wrap: %w", ErrActionSkipped),
			wantInformative: true,
			wantFatal:       false,
		},
		{
			name:            "ErrActionNotAvailable",
			err:             ErrActionNotAvailable,
			wantInformative: true,
			wantFatal:       false,
		},
		{
			name:            "ErrActionTurnedOff",
			err:             ErrActionTurnedOff,
			wantInformative: true,
			wantFatal:       false,
		},
		{
			name:            "ErrActionPending",
			err:             ErrActionPending,
			wantInformative: true,
			wantFatal:       false,
		},
		{
			name:            "ErrActionFailed",
			err:             NewErrActionFailed("failed to create PR: %s", "network timeout"),
			wantInformative: false,
			wantFatal:       true,
		},
		{
			name:            "arbitrary fatal error",
			err:             errors.New("unexpected database error"),
			wantInformative: false,
			wantFatal:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.wantInformative, IsActionInformativeError(tt.err))
			assert.Equal(t, tt.wantFatal, IsActionFatalError(tt.err))
		})
	}
}

func TestHTTPErrorCodeToErr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code    int
		wantErr error
	}{
		// 2xx Success status codes
		{code: 200, wantErr: nil},
		{code: 201, wantErr: nil},
		{code: 204, wantErr: nil},
		{code: 299, wantErr: nil},

		// Explicit mapped HTTP client errors
		{code: 401, wantErr: ErrUnauthorized},
		{code: 403, wantErr: ErrForbidden},
		{code: 404, wantErr: ErrNotFound},
		{code: 422, wantErr: ErrValidateOrSpammed},

		// Generic 4xx client errors
		{code: 400, wantErr: ErrClientError},
		{code: 409, wantErr: ErrClientError},
		{code: 429, wantErr: ErrClientError},
		{code: 499, wantErr: ErrClientError},

		// 5xx Server errors
		{code: 500, wantErr: ErrServerError},
		{code: 502, wantErr: ErrServerError},
		{code: 503, wantErr: ErrServerError},
		{code: 599, wantErr: ErrServerError},

		// Boundary & unrecognized status codes
		{code: 100, wantErr: ErrOther},
		{code: 199, wantErr: ErrOther},
		{code: 301, wantErr: ErrOther},
		{code: 302, wantErr: ErrOther},
		{code: 0, wantErr: ErrOther},
		{code: -1, wantErr: ErrOther},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("HTTP_%d", tt.code), func(t *testing.T) {
			t.Parallel()
			err := HTTPErrorCodeToErr(tt.code)
			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.wantErr)
			}
		})
	}
}
