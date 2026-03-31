// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasource

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"buf.build/go/protoyaml"
	"github.com/go-openapi/spec"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var stdoutMu sync.Mutex

func TestSwaggerToDataSource_Success(t *testing.T) {
	tests := []struct {
		name    string
		swagger *spec.Swagger
		assert  func(t *testing.T, ds *minderv1.DataSource)
	}{
		{
			name: "single endpoint without parameters",
			swagger: testSwagger(map[string]spec.PathItem{
				"/users": pathItem("GET", op()),
			}),
			assert: func(t *testing.T, ds *minderv1.DataSource) {
				t.Helper()
				require.NotNil(t, ds.GetRest())
				require.Len(t, ds.GetRest().GetDef(), 1)

				def := ds.GetRest().GetDef()["get_users"]
				require.NotNil(t, def)
				assert.Equal(t, "GET", def.GetMethod())
				assert.Equal(t, "/api/v1/users", def.GetEndpoint())
				assert.Equal(t, "json", def.GetParse())
				assert.Empty(t, def.GetInputSchema().AsMap())
			},
		},
		{
			name: "endpoint with path parameter",
			swagger: testSwagger(map[string]spec.PathItem{
				"/users/{id}": pathItem("GET", op(param("id", "path", true))),
			}),
			assert: func(t *testing.T, ds *minderv1.DataSource) {
				t.Helper()
				def := ds.GetRest().GetDef()["get_users_id_"]
				require.NotNil(t, def)
				assert.Equal(t, map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id": map[string]any{
							"type": "string",
						},
					},
					"required": []any{"id"},
				}, def.GetInputSchema().AsMap())
			},
		},
		{
			name: "post endpoint includes body field",
			swagger: testSwagger(map[string]spec.PathItem{
				"/users": pathItem("POST", op()),
			}),
			assert: func(t *testing.T, ds *minderv1.DataSource) {
				t.Helper()
				def := ds.GetRest().GetDef()["post_users"]
				require.NotNil(t, def)
				assert.Equal(t, "POST", def.GetMethod())
				assert.Equal(t, "body", def.GetBodyFromField())
				assert.Equal(t, map[string]any{
					"type": "object",
					"properties": map[string]any{
						"body": map[string]any{
							"type": "object",
						},
					},
				}, def.GetInputSchema().AsMap())
			},
		},
		{
			name: "multiple endpoints",
			swagger: testSwagger(map[string]spec.PathItem{
				"/users":      pathItem("GET", op()),
				"/users/{id}": pathItem("PUT", op(param("id", "path", true))),
			}),
			assert: func(t *testing.T, ds *minderv1.DataSource) {
				t.Helper()
				require.Len(t, ds.GetRest().GetDef(), 2)

				getDef := ds.GetRest().GetDef()["get_users"]
				require.NotNil(t, getDef)
				assert.Equal(t, "/api/v1/users", getDef.GetEndpoint())

				putDef := ds.GetRest().GetDef()["put_users_id_"]
				require.NotNil(t, putDef)
				assert.Equal(t, "/api/v1/users/{id}", putDef.GetEndpoint())
				assert.Equal(t, "body", putDef.GetBodyFromField())
				assert.Equal(t, map[string]any{
					"type": "object",
					"properties": map[string]any{
						"body": map[string]any{
							"type": "object",
						},
						"id": map[string]any{
							"type": "string",
						},
					},
					"required": []any{"id"},
				}, putDef.GetInputSchema().AsMap())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds, err := runSwaggerToDataSource(t, tt.swagger)
			require.NoError(t, err)
			tt.assert(t, ds)
		})
	}
}

func TestSwaggerToDataSource_ErrorCases(t *testing.T) {
	tests := []struct {
		name    string
		swagger *spec.Swagger
		wantErr string
	}{
		{
			name: "unsupported header parameter",
			swagger: testSwagger(map[string]spec.PathItem{
				"/users": pathItem("GET", op(param("X-Token", "header", true))),
			}),
			wantErr: `GET /users: unsupported parameter "X-Token" in "header"`,
		},
		{
			name: "unsupported body parameter",
			swagger: testSwagger(map[string]spec.PathItem{
				"/users": pathItem("POST", op(param("payload", "body", true))),
			}),
			wantErr: `POST /users: unsupported parameter "payload" in "body"`,
		},
		{
			name: "duplicate generated operation names",
			swagger: testSwagger(map[string]spec.PathItem{
				"/users/id/":  pathItem("GET", op()),
				"/users/{id}": pathItem("GET", op(param("id", "path", true))),
			}),
			wantErr: `duplicate generated operation name "get_users_id_"`,
		},
		{
			name:    "missing info section",
			swagger: &spec.Swagger{SwaggerProps: spec.SwaggerProps{BasePath: "/api/v1", Paths: &spec.Paths{}}},
			wantErr: "info section is required in OpenAPI spec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runSwaggerToDataSource(t, tt.swagger)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestGenerateCmdRun_InvalidSwaggerDocuments(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		wantErr string
	}{
		{
			name:    "empty document",
			content: nil,
			wantErr: "error parsing OpenAPI spec",
		},
		{
			name:    "invalid document",
			content: []byte("not: [valid"),
			wantErr: "error parsing OpenAPI spec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "swagger.yaml")
			require.NoError(t, os.WriteFile(path, tt.content, 0o600))

			err := generateCmdRun(&cobra.Command{}, []string{path})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func runSwaggerToDataSource(t *testing.T, swagger *spec.Swagger) (*minderv1.DataSource, error) {
	t.Helper()
	stdoutMu.Lock()
	defer stdoutMu.Unlock()

	stdoutFile, err := os.CreateTemp(t.TempDir(), "stdout-*.yaml")
	require.NoError(t, err)

	originalStdout := os.Stdout
	os.Stdout = stdoutFile
	defer func() {
		os.Stdout = originalStdout
	}()

	err = swaggerToDataSource(&cobra.Command{}, swagger)
	require.NoError(t, stdoutFile.Close())

	if err != nil {
		return nil, err
	}

	data, readErr := os.ReadFile(stdoutFile.Name())
	require.NoError(t, readErr)

	var ds minderv1.DataSource
	require.NoError(t, protoyaml.Unmarshal(data, &ds))

	return &ds, nil
}

func testSwagger(paths map[string]spec.PathItem) *spec.Swagger {
	return &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger:  "2.0",
			BasePath: "/api/v1",
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Title: "users-api",
				},
			},
			Paths: &spec.Paths{
				Paths: paths,
			},
		},
	}
}

func pathItem(method string, operation *spec.Operation) spec.PathItem {
	item := spec.PathItem{}
	switch method {
	case "GET":
		item.Get = operation
	case "POST":
		item.Post = operation
	case "PUT":
		item.Put = operation
	default:
		panic("unsupported test method: " + method)
	}

	return item
}

func op(params ...spec.Parameter) *spec.Operation {
	return &spec.Operation{
		OperationProps: spec.OperationProps{
			Parameters: params,
		},
	}
}

func param(name, in string, required bool) spec.Parameter {
	return spec.Parameter{
		ParamProps: spec.ParamProps{
			Name:     name,
			In:       in,
			Required: required,
		},
		SimpleSchema: spec.SimpleSchema{
			Type: "string",
		},
	}
}

func TestGenerateCmdRun_WithParsedSwaggerFixture(t *testing.T) {
	swagger := testSwagger(map[string]spec.PathItem{
		"/users": pathItem("GET", op()),
	})

	data, err := json.Marshal(swagger)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "swagger.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))

	stdoutFile, err := os.CreateTemp(t.TempDir(), "stdout-*.yaml")
	require.NoError(t, err)

	stdoutMu.Lock()
	defer stdoutMu.Unlock()

	originalStdout := os.Stdout
	os.Stdout = stdoutFile
	defer func() {
		os.Stdout = originalStdout
	}()

	err = generateCmdRun(&cobra.Command{}, []string{path})
	require.NoError(t, stdoutFile.Close())
	require.NoError(t, err)

	out, err := os.ReadFile(stdoutFile.Name())
	require.NoError(t, err)

	var ds minderv1.DataSource
	require.NoError(t, protoyaml.Unmarshal(out, &ds))
	require.Contains(t, ds.GetRest().GetDef(), "get_users")
}
