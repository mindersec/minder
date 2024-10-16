// Copyright 2024 Stacklok, Inc
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
func (_ *TestKit) GetBaseURL() string {
	return ""
}

// NewRequest implements the REST interface.
func (_ *TestKit) NewRequest(method, url string, body any) (*http.Request, error) {
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

	h := func(w http.ResponseWriter, _ *http.Request) {
		for k, v := range tk.httpHeaders {
			w.Header().Set(k, v)
		}

		w.WriteHeader(tk.httpStatus)
		_, _ = w.Write(tk.httpBody)
	}

	h(tk.httpRecorder, req)

	return tk.httpRecorder.Result(), nil
}
