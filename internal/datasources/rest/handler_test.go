// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rest

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/util/schemavalidate"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	mock_v1 "github.com/mindersec/minder/pkg/providers/v1/mock"
)

func Test_parseRequestBodyConfig(t *testing.T) {
	t.Parallel()

	type args struct {
		def *minderv1.RestDataSource_Def
	}
	tests := []struct {
		name          string
		args          args
		want          string
		wantFromInput bool
		errMsg        string
	}{
		{
			name: "Nil body",
			args: args{
				def: &minderv1.RestDataSource_Def{},
			},
			want: "",
		},
		{
			name: "Body as string",
			args: args{
				def: &minderv1.RestDataSource_Def{
					Body: &minderv1.RestDataSource_Def_Bodystr{Bodystr: "test body"},
				},
			},
			want: "test body",
		},
		{
			name: "Body as object",
			args: args{
				def: &minderv1.RestDataSource_Def{
					Body: &minderv1.RestDataSource_Def_Bodyobj{
						Bodyobj: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"key": structpb.NewStringValue("value"),
							},
						},
					},
				},
			},
			want: `{"key":"value"}`,
		},
		{
			name: "Body from input",
			args: args{
				def: &minderv1.RestDataSource_Def{
					Body: &minderv1.RestDataSource_Def_BodyFromField{
						BodyFromField: "key",
					},
				},
			},
			want:          "key",
			wantFromInput: true,
		},
		{
			name: "Error in body from input",
			args: args{
				def: &minderv1.RestDataSource_Def{
					Body: &minderv1.RestDataSource_Def_BodyFromField{},
				},
			},
			errMsg: "body_from_field is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotFromInput, gotStr, err := parseRequestBodyConfig(tt.args.def)
			if tt.errMsg != "" {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, gotStr)
			assert.Equal(t, tt.wantFromInput, gotFromInput)
		})
	}
}

func Test_restHandler_HTTPCall(t *testing.T) {
	t.Parallel()

	type fields struct {
		endpointTmpl string
		method       string
		body         string
		headers      map[string]string
		parse        string
	}
	tests := []struct {
		name        string
		fields      fields
		args        any
		addProvider bool
		want        any
		errMsg      string
		mockStatus  int
		mockBody    string
	}{
		{
			name: "Example test case",
			fields: fields{
				endpointTmpl: "/example",
				method:       "GET",
				headers:      map[string]string{"Content-Type": "application/json"},
				// No parsing
				parse: "",
			},
			args:        map[string]any{},
			addProvider: true,
			want:        buildRestOutput(http.StatusOK, `{"key":"value"}`),
			mockStatus:  http.StatusOK,
			mockBody:    `{"key":"value"}`,
		},
		{
			name: "Invalid method",
			fields: fields{
				endpointTmpl: "/invalid-method",
				method:       "INVALID",
				headers:      map[string]string{"Content-Type": "application/json"},
				parse:        "",
			},
			args:       map[string]any{},
			want:       buildRestOutput(http.StatusMethodNotAllowed, `{"error":"method not allowed"}`),
			mockStatus: http.StatusMethodNotAllowed,
			mockBody:   `{"error":"method not allowed"}`,
		},
		{
			name: "Invalid args",
			fields: fields{
				endpointTmpl: "/invalid-args",
				method:       "GET",
			},
			args:   "wrong",
			errMsg: "args is not a map",
		},
		{
			name: "Invalid URL",
			fields: fields{
				endpointTmpl: "/missingBracket}",
				method:       "GET",
			},
			args:   map[string]any{},
			errMsg: "failed to expand token, invalid at col: 37",
		},
		{
			name: "JSON parsing",
			fields: fields{
				endpointTmpl: "/json",
				method:       "GET",
				headers:      map[string]string{"Content-Type": "application/json"},
				parse:        "json",
			},
			args:       map[string]any{},
			want:       buildRestOutput(http.StatusOK, map[string]any{"key": "value"}),
			mockStatus: http.StatusOK,
			mockBody:   `{"key":"value"}`,
		},
		{
			name: "Non-JSON response",
			fields: fields{
				endpointTmpl: "/non-json",
				method:       "GET",
				headers:      map[string]string{"Content-Type": "application/json"},
				parse:        "",
			},
			args:       map[string]any{},
			want:       buildRestOutput(http.StatusOK, "plain text response"),
			mockStatus: http.StatusOK,
			mockBody:   "plain text response",
		},
		{
			name: "Backoff requested",
			fields: fields{
				endpointTmpl: "/backoff",
			},
			args:       map[string]any{},
			mockStatus: http.StatusTooManyRequests,
			errMsg:     "rate limited",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			var provider interfaces.RESTProvider
			if tt.addProvider {
				mock := mock_v1.NewMockREST(ctrl)
				mock.EXPECT().GetBaseURL().Return("http://provider")
				provider = mock
			}

			// Create a new httptest.Server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.mockStatus)
				_, err := w.Write([]byte(tt.mockBody))
				require.NoError(t, err)
			}))
			defer server.Close()

			// Update endpointTmpl to use the test server URL
			tt.fields.endpointTmpl = server.URL + tt.fields.endpointTmpl

			h := &restHandler{
				endpointTmpl:      tt.fields.endpointTmpl,
				method:            tt.fields.method,
				body:              tt.fields.body,
				headers:           tt.fields.headers,
				parse:             tt.fields.parse,
				testOnlyTransport: http.DefaultTransport,
				provider:          provider,
			}
			initMetrics()

			got, err := h.Call(context.Background(), nil, tt.args)
			if tt.errMsg != "" {
				assert.ErrorContains(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_restHandler_ProviderCall(t *testing.T) {
	t.Parallel()

	type fields struct {
		endpointTmpl  string
		method        string
		body          string
		bodyFromInput bool
		headers       map[string]string
		parse         string
	}
	tests := []struct {
		name        string
		fields      fields
		args        any
		prepareMock func(provider *mock_v1.MockREST)
		want        any
		errMsg      string
	}{
		{
			name: "Example test case",
			fields: fields{
				endpointTmpl: "/example",
				method:       "GET",
				headers:      map[string]string{"Content-Type": "application/json"},
			},
			args: map[string]any{},
			prepareMock: func(provider *mock_v1.MockREST) {
				provider.EXPECT().NewRequest("GET", "http://provider/example", nil).Return(
					http.NewRequest("GET", "http://provider/example", nil))
				provider.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(`{"key":"value"}`)),
							Request:    req,
						}, nil
					})
			},
			want: buildRestOutput(http.StatusOK, `{"key":"value"}`),
		},
		{
			name: "Invalid method",
			fields: fields{
				endpointTmpl: "/invalid-method",
				method:       "INVALID",
				headers:      map[string]string{"Content-Type": "application/json"},
			},
			prepareMock: func(provider *mock_v1.MockREST) {
				provider.EXPECT().NewRequest("INVALID", "http://provider/invalid-method", nil).Return(
					http.NewRequest("INVALID", "http://provider/invalid-method", nil))
				provider.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusMethodNotAllowed,
							Body:       io.NopCloser(strings.NewReader(`{"error":"method not allowed"}`)),
							Request:    req,
						}, nil
					})
			},
			args: map[string]any{},
			want: buildRestOutput(http.StatusMethodNotAllowed, `{"error":"method not allowed"}`),
		},
		{
			name: "Failed provider Do",
			fields: fields{
				endpointTmpl: "/failure",
			},
			prepareMock: func(provider *mock_v1.MockREST) {
				provider.EXPECT().NewRequest("", "http://provider/failure", nil).Return(
					http.NewRequest("GET", "http://provider/failure", nil))
				provider.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, _ *http.Request) (*http.Response, error) {
						return nil, errors.New("failed to do request")
					}).Times(4)
			},
			args:   map[string]any{},
			errMsg: "failed to do request",
		},
		{
			name: "Static body",
			fields: fields{
				endpointTmpl: "/static-body",
				method:       "POST",
				body:         `{"static":"body"}`,
			},
			prepareMock: func(provider *mock_v1.MockREST) {
				provider.EXPECT().NewRequest("POST", "http://provider/static-body", nil).Return(
					http.NewRequest("POST", "http://provider/static-body", nil))
				provider.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, req *http.Request) (*http.Response, error) {
						b, err := io.ReadAll(req.Body)
						if err != nil {
							t.Logf("Failed to read body: %v", err)
							return nil, err
						}
						if string(b) != `{"static":"body"}` {
							t.Logf("Got %q", string(b))
							return nil, fmt.Errorf("request body %q does not match expected", string(b))
						}
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(`{"result":"success"}`)),
							Request:    req,
						}, nil
					})
			},
			args: map[string]any{},
			want: buildRestOutput(http.StatusOK, `{"result":"success"}`),
		},
		{
			name: "Body from input object",
			fields: fields{
				endpointTmpl:  "/static-body",
				method:        "POST",
				bodyFromInput: true,
				body:          "request",
			},
			prepareMock: func(provider *mock_v1.MockREST) {
				provider.EXPECT().NewRequest("POST", "http://provider/static-body", nil).Return(
					http.NewRequest("POST", "http://provider/static-body", nil))
				provider.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, req *http.Request) (*http.Response, error) {
						b, err := io.ReadAll(req.Body)
						if err != nil {
							t.Logf("Failed to read body: %v", err)
							return nil, err
						}
						if string(b) != `{"count":2,"dynamic":"content"}` {
							t.Logf("Got %q", string(b))
							return nil, fmt.Errorf("request body %q does not match expected", string(b))
						}
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(`{"result":"success"}`)),
							Request:    req,
						}, nil
					})
			},
			args: map[string]any{
				"request": map[string]any{
					"dynamic": "content",
					"count":   2,
				},
			},
			want: buildRestOutput(http.StatusOK, `{"result":"success"}`),
		},
		{
			name: "Body from input string",
			fields: fields{
				endpointTmpl:  "/static-body",
				method:        "POST",
				bodyFromInput: true,
				body:          "request",
			},
			prepareMock: func(provider *mock_v1.MockREST) {
				provider.EXPECT().NewRequest("POST", "http://provider/static-body", nil).Return(
					http.NewRequest("POST", "http://provider/static-body", nil))
				provider.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, req *http.Request) (*http.Response, error) {
						b, err := io.ReadAll(req.Body)
						if err != nil {
							t.Logf("Failed to read body: %v", err)
							return nil, err
						}
						if string(b) != `a string` {
							t.Logf("Got %q", string(b))
							return nil, fmt.Errorf("request body %q does not match expected", string(b))
						}
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(`{"result":"success"}`)),
							Request:    req,
						}, nil
					})
			},
			args: map[string]any{
				"request": "a string",
			},
			want: buildRestOutput(http.StatusOK, `{"result":"success"}`),
		},
		{
			name: "Body error",
			fields: fields{
				endpointTmpl:  "/static-body",
				bodyFromInput: true,
				body:          "missing",
			},
			args:   map[string]any{},
			errMsg: `body key "missing" not found in args`,
		},
		{
			name: "Body encode error",
			fields: fields{
				endpointTmpl:  "/static-body",
				method:        "POST",
				bodyFromInput: true,
				body:          "request",
			},
			args: map[string]any{
				"request": 4,
			},
			errMsg: `body key "request" is not a string or object`,
		},
		{
			name: "NewRequest error",
			fields: fields{
				endpointTmpl: "/norequest",
			},
			prepareMock: func(provider *mock_v1.MockREST) {
				provider.EXPECT().NewRequest("", "http://provider/norequest", nil).Return(nil, errors.New("fail"))
			},
			args:   map[string]any{},
			errMsg: "fail",
		},
		{
			name: "Invalid body content",
			fields: fields{
				endpointTmpl: "/invalid-body",
				parse:        "json",
			},
			prepareMock: func(provider *mock_v1.MockREST) {
				provider.EXPECT().NewRequest("", "http://provider/invalid-body", nil).Return(
					http.NewRequest("GET", "http://provider/invalid-body", nil))
				provider.EXPECT().Do(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader(`{invalid json`)),
							Request:    req,
						}, nil
					})
			},
			args:   map[string]any{},
			errMsg: "cannot decode json: invalid character 'i' looking for beginning of object key string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			provider := mock_v1.NewMockREST(ctrl)

			tt.fields.endpointTmpl = "http://provider" + tt.fields.endpointTmpl

			if tt.prepareMock != nil {
				provider.EXPECT().GetBaseURL().Return("http://provider")
				tt.prepareMock(provider)
			}

			h := &restHandler{
				endpointTmpl:  tt.fields.endpointTmpl,
				method:        tt.fields.method,
				body:          tt.fields.body,
				bodyFromInput: tt.fields.bodyFromInput,
				headers:       tt.fields.headers,
				parse:         tt.fields.parse,
				provider:      provider,
			}
			initMetrics()

			got, err := h.Call(context.Background(), nil, tt.args)
			if tt.errMsg != "" {
				assert.ErrorContains(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_restHandler_ValidateArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		hasSchema bool
		args      any
		wantErr   bool
	}{
		{
			name:      "Valid args",
			hasSchema: true,
			args:      map[string]any{"key": "value"},
			wantErr:   false,
		},
		{
			name:      "Invalid args type",
			hasSchema: true,
			args:      "invalid_type",
			wantErr:   true,
		},
		{
			name:      "Invalid args value",
			hasSchema: true,
			args:      map[string]any{"key": 123},
			wantErr:   true,
		},
		{
			name:      "No schema",
			hasSchema: false,
			args:      map[string]any{"key": "value"},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var schema *jsonschema.Schema
			if tt.hasSchema {
				var err error
				schema, err = schemavalidate.CompileSchemaFromMap(
					map[string]any{"type": "object", "properties": map[string]any{"key": map[string]any{"type": "string"}}},
				)
				require.NoError(t, err)
			}

			h := &restHandler{
				inputSchema: schema,
			}
			err := h.ValidateArgs(tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_restHandler_ValidateUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		inputSchema  map[string]any
		updateSchema *structpb.Struct
		wantErr      bool
	}{
		{
			name: "Valid structpb.Struct",
			inputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type": "string",
					},
				},
			},
			updateSchema: func() *structpb.Struct {
				res, err := structpb.NewStruct(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"key": map[string]any{
							"type": "string",
						},
						"new_key": map[string]any{
							"type": "number",
						},
					},
				})
				require.NoError(t, err)
				return res
			}(),
			wantErr: false,
		},
		{
			name: "nil update schema",
			inputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"key": map[string]any{"type": "string"}},
			},
			updateSchema: nil,
			wantErr:      true,
		},
		{
			name: "Invalid update schema",
			inputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"key": map[string]any{"type": "string"}},
			},
			updateSchema: func() *structpb.Struct {
				res, err := structpb.NewStruct(map[string]any{
					"type": 5,
				})
				require.NoError(t, err)
				return res
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, err := structpb.NewStruct(tt.inputSchema)
			require.NoError(t, err, "failed to create structpb.Struct")

			h := &restHandler{
				rawInputSchema: s,
			}

			// Validate that the input schema is a valid JSON schema
			_, err = schemavalidate.CompileSchemaFromMap(tt.inputSchema)
			require.NoError(t, err, "input schema is not a valid JSON schema")

			err = h.ValidateUpdate(tt.updateSchema)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
