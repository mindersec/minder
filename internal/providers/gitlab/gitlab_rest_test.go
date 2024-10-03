//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/providers/credentials"
	"github.com/stacklok/minder/internal/util/ptr"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

type mockGitlabClient struct {
	doFunc         func(ctx context.Context, req *http.Request) (*http.Response, error)
	newRequestFunc func(method, requestUrl string, body any) (*http.Request, error)
}

func (m *mockGitlabClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return m.doFunc(ctx, req)
}

func (m *mockGitlabClient) NewRequest(method, requestUrl string, body any) (*http.Request, error) {
	return m.newRequestFunc(method, requestUrl, body)
}

func Test_gitlabClient_Do(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockServerFunc func(w http.ResponseWriter, r *http.Request)
		wantStatusCode int
		wantErr        bool
		clientTmout    *time.Duration
	}{
		{
			name: "successful request",
			mockServerFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "request with context",
			mockServerFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name: "server error scenario still returns response",
			mockServerFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantErr:        false,
		},
		{
			name: "error scenario",
			mockServerFunc: func(_ http.ResponseWriter, r *http.Request) {
				// force the client to timeout
				time.Sleep(time.Millisecond * 20)
				r.Context().Done()
			},
			wantErr:     true,
			clientTmout: ptr.Ptr(time.Millisecond * 10),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := httptest.NewServer(http.HandlerFunc(tt.mockServerFunc))
			defer ts.Close()

			htcli := ts.Client()
			if tt.clientTmout != nil {
				htcli.Timeout = *tt.clientTmout
			}

			client := &gitlabClient{
				cli: htcli,
			}

			req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
			assert.NoError(t, err)

			resp, err := client.Do(context.Background(), req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantStatusCode, resp.StatusCode)
			}
		})
	}
}

func Test_gitlabClient_GetBaseURL(t *testing.T) {
	t.Parallel()

	client := &gitlabClient{
		glcfg: &minderv1.GitLabProviderConfig{
			Endpoint: "http://example.com",
		},
	}

	assert.Equal(t, "http://example.com", client.GetBaseURL())
}

func Test_gitlabClient_NewRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		method      string
		requestPath string
		body        any
		wantErr     bool
		endpoint    string
	}{
		{
			name:        "GET request without body",
			method:      http.MethodGet,
			requestPath: "/test",
			body:        nil,
			wantErr:     false,
		},
		{
			name:        "POST request with body",
			method:      http.MethodPost,
			requestPath: "/test",
			body:        map[string]string{"key": "value"},
			wantErr:     false,
		},
		{
			name:        "invalid URL gets cleaned up",
			method:      http.MethodGet,
			requestPath: "..:/invalid-url",
			wantErr:     true,
		},
		{
			name:        "invalid URL parsing error",
			method:      http.MethodGet,
			requestPath: "/path",
			wantErr:     true,
			endpoint:    "://example.com:80:80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &gitlabClient{
				glcfg: &minderv1.GitLabProviderConfig{
					Endpoint: "http://example.com",
				},
				cred: &credentials.GitLabTokenCredential{},
			}
			if tt.endpoint != "" {
				client.glcfg.Endpoint = tt.endpoint
			}

			req, err := client.NewRequest(tt.method, tt.requestPath, tt.body)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.method, req.Method)
				url.PathEscape(tt.requestPath)
				u, err := url.JoinPath("http://example.com", tt.requestPath)
				require.NoError(t, err)
				assert.Equal(t, u, req.URL.String())
				if tt.body != nil {
					assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
				}
				assert.Equal(t, "application/json", req.Header.Get("Accept"))
				assert.Equal(t, "Minder", req.Header.Get("User-Agent"))
			}
		})
	}
}

func Test_glRESTGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mockServerFunc func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		wantResult     map[string]string
	}{
		{
			name: "successful GET request",
			mockServerFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				err := json.NewEncoder(w).Encode(map[string]string{"key": "value"})
				require.NoError(t, err)
			},
			wantErr:    false,
			wantResult: map[string]string{"key": "value"},
		},
		{
			name: "404 Not Found",
			mockServerFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr:    true,
			wantResult: nil,
		},
		{
			name: "JSON decoding error",
			mockServerFunc: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte("invalid json"))
				require.NoError(t, err)
			},
			wantErr:    true,
			wantResult: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ts := httptest.NewServer(http.HandlerFunc(tt.mockServerFunc))
			defer ts.Close()

			client := &mockGitlabClient{
				doFunc: func(_ context.Context, req *http.Request) (*http.Response, error) {
					return ts.Client().Do(req)
				},
				newRequestFunc: func(method, requestUrl string, _ any) (*http.Request, error) {
					return http.NewRequest(method, ts.URL+requestUrl, nil)
				},
			}

			var result map[string]string
			err := glRESTGet(context.Background(), client, "/test", &result)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantResult, result)
			}
		})
	}
}

func Test_getParsedURL(t *testing.T) {
	t.Parallel()

	type args struct {
		endpoint string
		// This may include query parameters
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    *url.URL
		wantErr bool
	}{
		{
			name: "valid URL",
			args: args{
				endpoint: "http://example.com/v1/api",
				path:     "/test",
			},
			want: &url.URL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "/v1/api/test",
			},
		},
		{
			name: "valid URL with query parameters",
			args: args{
				endpoint: "http://example.com/v1/api",
				path:     "/test?param1=value1&param2=value2",
			},
			want: &url.URL{
				Scheme:   "http",
				Host:     "example.com",
				Path:     "/v1/api/test",
				RawQuery: "param1=value1&param2=value2",
			},
		},
		{
			name: "valid URL with fragment identifier",
			args: args{
				endpoint: "http://example.com/v1/api",
				path:     "/test#section",
			},
			want: &url.URL{
				Scheme:   "http",
				Host:     "example.com",
				Path:     "/v1/api/test",
				Fragment: "section",
			},
		},
		{
			name: "invalid URL",
			args: args{
				endpoint: "://example.com:80:80",
				path:     "/test",
			},
			wantErr: true,
		},
		{
			name: "query parameter that needs escaping",
			args: args{
				endpoint: "http://example.com",
				path:     "/test?param1=va lue1",
			},
			want: &url.URL{
				Scheme:   "http",
				Host:     "example.com",
				Path:     "test",
				RawQuery: "param1=va%20lue1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := getParsedURL(tt.args.endpoint, tt.args.path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want.Scheme, got.Scheme, "Expected scheme to be equal")
			assert.Equal(t, tt.want.Host, got.Host, "Expected host to be equal")
			assert.Equal(t, tt.want.Path, got.Path, "Expected path to be equal")
			assert.Equal(t, tt.want.Query(), got.Query(), "Expected query to be equal")
			assert.Equal(t, tt.want.Fragment, got.Fragment, "Expected fragment to be equal")
		})
	}
}
