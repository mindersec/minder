// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func Test_restHandler_isExpectedStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		expectedStatus []int32
		statusCode     int
		want           bool
	}{
		{
			name:           "unset defaults to 200 only",
			expectedStatus: nil,
			statusCode:     http.StatusOK,
			want:           true,
		},
		{
			name:           "unset rejects non-200",
			expectedStatus: nil,
			statusCode:     http.StatusNotFound,
			want:           false,
		},
		{
			name:           "explicit list matches",
			expectedStatus: []int32{http.StatusOK, http.StatusAccepted},
			statusCode:     http.StatusAccepted,
			want:           true,
		},
		{
			name:           "explicit list rejects unlisted code",
			expectedStatus: []int32{http.StatusOK, http.StatusAccepted},
			statusCode:     http.StatusNotFound,
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := &restHandler{expectedStatus: tt.expectedStatus}
			assert.Equal(t, tt.want, h.isExpectedStatus(tt.statusCode))
		})
	}
}

func Test_restHandler_matchFallback(t *testing.T) {
	t.Parallel()

	fallbacks := []*minderv1.RestDataSource_Def_Fallback{
		{HttpStatus: http.StatusNotFound, Body: `{"note":"not found fallback"}`},
		{HttpStatus: http.StatusServiceUnavailable, Body: `{"note":"unavailable fallback"}`},
	}

	tests := []struct {
		name       string
		fallback   []*minderv1.RestDataSource_Def_Fallback
		statusCode int
		wantOK     bool
		wantBody   string
	}{
		{
			name:       "no fallback configured",
			fallback:   nil,
			statusCode: http.StatusNotFound,
			wantOK:     false,
		},
		{
			name:       "matching fallback",
			fallback:   fallbacks,
			statusCode: http.StatusNotFound,
			wantOK:     true,
			wantBody:   `{"note":"not found fallback"}`,
		},
		{
			name:       "no match for this status",
			fallback:   fallbacks,
			statusCode: http.StatusInternalServerError,
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := &restHandler{fallback: tt.fallback}
			fb, ok := h.matchFallback(tt.statusCode)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				require.NotNil(t, fb)
				assert.Equal(t, tt.wantBody, fb.GetBody())
			}
		})
	}
}

// Test_restHandler_HTTPCall_Fallback exercises the fallback behavior through
// the full Call path, following the same server/table pattern as
// Test_restHandler_HTTPCall.
func Test_restHandler_HTTPCall_Fallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockStatus     int
		mockBody       string
		expectedStatus []int32
		fallback       []*minderv1.RestDataSource_Def_Fallback
		want           any
	}{
		{
			name:       "expected status ignores fallback",
			mockStatus: http.StatusOK,
			mockBody:   `{"key":"value"}`,
			fallback: []*minderv1.RestDataSource_Def_Fallback{
				{HttpStatus: http.StatusNotFound, Body: `{"note":"fallback"}`},
			},
			want: buildRestOutput(http.StatusOK, `{"key":"value"}`),
		},
		{
			name:       "unexpected status with matching fallback returns fallback body",
			mockStatus: http.StatusNotFound,
			mockBody:   `{"error":"real upstream body, should not surface"}`,
			fallback: []*minderv1.RestDataSource_Def_Fallback{
				{HttpStatus: http.StatusNotFound, Body: `{"note":"fallback"}`},
			},
			want: buildRestOutput(http.StatusNotFound, `{"note":"fallback"}`),
		},
		{
			name:       "unexpected status without matching fallback keeps existing behavior",
			mockStatus: http.StatusInternalServerError,
			mockBody:   `{"error":"boom"}`,
			fallback: []*minderv1.RestDataSource_Def_Fallback{
				{HttpStatus: http.StatusNotFound, Body: `{"note":"fallback"}`},
			},
			want: buildRestOutput(http.StatusInternalServerError, `{"error":"boom"}`),
		},
		{
			name:           "explicit expected_status without fallback passes status through",
			mockStatus:     http.StatusAccepted,
			mockBody:       `{"queued":true}`,
			expectedStatus: []int32{http.StatusOK, http.StatusAccepted},
			want:           buildRestOutput(http.StatusAccepted, `{"queued":true}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.mockStatus)
				_, err := w.Write([]byte(tt.mockBody))
				require.NoError(t, err)
			}))
			defer server.Close()

			h := &restHandler{
				endpointTmpl:      server.URL,
				method:            http.MethodGet,
				testOnlyTransport: http.DefaultTransport,
				expectedStatus:    tt.expectedStatus,
				fallback:          tt.fallback,
			}
			initMetrics()

			got, err := h.Call(context.Background(), nil, map[string]any{})
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
