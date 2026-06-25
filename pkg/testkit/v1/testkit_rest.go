// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/rs/zerolog"
)

// GetBaseURL implements the REST interface.
func (*TestKit) GetBaseURL() string {
	return ""
}

// NewRequest implements the REST interface.
func (*TestKit) NewRequest(method, url string, body any) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body.([]byte))
	}
	return httptest.NewRequest(method, url, r), nil
}

// Do executes an HTTP request.
func (tk *TestKit) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	zerolog.Ctx(ctx).Debug().
		Str("component", "testkit").
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Msg("HTTP request")

	recorder := httptest.NewRecorder()
	if tk.httpHandler != nil {
		tk.httpHandler.ServeHTTP(recorder, req)
	}
	resp := recorder.Result()
	resp.Request = req
	return resp, nil
}
