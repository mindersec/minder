// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/util/schemavalidate"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func Test_newHandlerFromDef(t *testing.T) {
	t.Parallel()

	type args struct {
		def *minderv1.RestDataSource_Def
	}
	tests := []struct {
		name    string
		args    args
		want    *restHandler
		wantErr bool
	}{
		{
			name: "Nil definition",
			args: args{
				def: nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Valid definition",
			args: args{
				def: &minderv1.RestDataSource_Def{
					InputSchema: &structpb.Struct{},
					Endpoint:    "http://example.com",
					Method:      "GET",
					Headers:     map[string]string{"Content-Type": "application/json"},
					Parse:       "json",
				},
			},
			want: &restHandler{
				rawnis:       &structpb.Struct{},
				nis:          &jsonschema.Schema{},
				endpointTmpl: "http://example.com",
				method:       "GET",
				headers:      map[string]string{"Content-Type": "application/json"},
				body:         "",
				parse:        "json",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := newHandlerFromDef(tt.args.def)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				assert.Equal(t, tt.want.rawnis.AsMap(), got.rawnis.AsMap())
				assert.Equal(t, tt.want.endpointTmpl, got.endpointTmpl)
				assert.Equal(t, tt.want.method, got.method)
				assert.Equal(t, tt.want.headers, got.headers)
				assert.Equal(t, tt.want.body, got.body)
				assert.Equal(t, tt.want.parse, got.parse)
			}
		})
	}
}

func Test_parseRequestBodyConfig(t *testing.T) {
	t.Parallel()

	type args struct {
		def *minderv1.RestDataSource_Def
	}
	tests := []struct {
		name string
		args args
		want string
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
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseRequestBodyConfig(tt.args.def)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_restHandler_Call(t *testing.T) {
	t.Parallel()

	type fields struct {
		endpointTmpl string
		method       string
		body         string
		headers      map[string]string
		parse        string
	}
	type args struct {
		args any
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       any
		wantErr    bool
		mockStatus int
		mockBody   string
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
			args:       args{args: map[string]any{}},
			want:       buildRestOutput(http.StatusOK, `{"key":"value"}`),
			wantErr:    false,
			mockStatus: http.StatusOK,
			mockBody:   `{"key":"value"}`,
		},
		{
			name: "Invalid method",
			fields: fields{
				endpointTmpl: "/invalid-method",
				method:       "INVALID",
				headers:      map[string]string{"Content-Type": "application/json"},
				parse:        "",
			},
			args:       args{args: map[string]any{}},
			want:       buildRestOutput(http.StatusMethodNotAllowed, `{"error":"method not allowed"}`),
			wantErr:    false,
			mockStatus: http.StatusMethodNotAllowed,
			mockBody:   `{"error":"method not allowed"}`,
		},
		{
			name: "JSON parsing",
			fields: fields{
				endpointTmpl: "/json",
				method:       "GET",
				headers:      map[string]string{"Content-Type": "application/json"},
				parse:        "json",
			},
			args:       args{args: map[string]any{}},
			want:       buildRestOutput(http.StatusOK, map[string]any{"key": "value"}),
			wantErr:    false,
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
			args:       args{args: map[string]any{}},
			want:       buildRestOutput(http.StatusOK, "plain text response"),
			wantErr:    false,
			mockStatus: http.StatusOK,
			mockBody:   "plain text response",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
				endpointTmpl: tt.fields.endpointTmpl,
				method:       tt.fields.method,
				body:         tt.fields.body,
				headers:      tt.fields.headers,
				parse:        tt.fields.parse,
			}
			got, err := h.Call(tt.args.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_restHandler_ValidateArgs(t *testing.T) {
	t.Parallel()

	type fields struct {
		rawnis       *structpb.Struct
		nis          *jsonschema.Schema
		endpointTmpl string
		method       string
		body         string
		headers      map[string]string
		parse        string
	}
	type args struct {
		args any
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Valid args",
			fields: fields{
				nis: func() *jsonschema.Schema {
					schema, err := schemavalidate.CompileSchemaFromMap(
						map[string]any{"type": "object", "properties": map[string]any{"key": map[string]any{"type": "string"}}},
					)
					require.NoError(t, err)
					return schema
				}(),
			},
			args: args{
				args: map[string]any{"key": "value"},
			},
			wantErr: false,
		},
		{
			name: "Invalid args type",
			fields: fields{
				nis: func() *jsonschema.Schema {
					schema, err := schemavalidate.CompileSchemaFromMap(
						map[string]any{"type": "object", "properties": map[string]any{"key": map[string]any{"type": "string"}}},
					)
					require.NoError(t, err)
					return schema
				}(),
			},
			args: args{
				args: "invalid_type",
			},
			wantErr: true,
		},
		{
			name: "Invalid args value",
			fields: fields{
				nis: func() *jsonschema.Schema {
					schema, err := schemavalidate.CompileSchemaFromMap(
						map[string]any{"type": "object", "properties": map[string]any{"key": map[string]any{"type": "string"}}},
					)
					require.NoError(t, err)
					return schema
				}(),
			},
			args: args{
				args: map[string]any{"key": 123},
			},
			wantErr: true,
		},
		{
			name: "No schema",
			fields: fields{
				nis: nil,
			},
			args: args{
				args: map[string]any{"key": "value"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := &restHandler{
				rawnis:       tt.fields.rawnis,
				nis:          tt.fields.nis,
				endpointTmpl: tt.fields.endpointTmpl,
				method:       tt.fields.method,
				body:         tt.fields.body,
				headers:      tt.fields.headers,
				parse:        tt.fields.parse,
			}
			err := h.ValidateArgs(tt.args.args)
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

	type args struct {
		updateSchema any
	}
	tests := []struct {
		name        string
		inputSchema map[string]any
		args        args
		wantErr     bool
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
			args: args{
				updateSchema: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"type": structpb.NewStringValue("object"),
						"properties": structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"key": structpb.NewStructValue(&structpb.Struct{
									Fields: map[string]*structpb.Value{
										"type": structpb.NewStringValue("string"),
									},
								}),
								"new_key": structpb.NewStructValue(&structpb.Struct{
									Fields: map[string]*structpb.Value{
										"type": structpb.NewStringValue("number"),
									},
								}),
							},
						}),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid map[string]any",
			inputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"key": map[string]any{"type": "string"}},
			},
			args: args{
				updateSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"key":     map[string]any{"type": "string"},
						"new_key": map[string]any{"type": "number"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid type",
			inputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"key": map[string]any{"type": "string"}},
			},
			args: args{
				updateSchema: "invalid_type",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, err := structpb.NewStruct(tt.inputSchema)
			require.NoError(t, err, "failed to create structpb.Struct")

			h := &restHandler{
				rawnis: s,
			}

			// Validate that the input schema is a valid JSON schema
			_, err = schemavalidate.CompileSchemaFromMap(tt.inputSchema)
			require.NoError(t, err, "input schema is not a valid JSON schema")

			err = h.ValidateUpdate(tt.args.updateSchema)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
