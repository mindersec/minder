// Copyright 2023 Stacklok, Inc.
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

// Package rest provides tests for the REST remediation engine
// we use the package rest directly because we need to test non-exported symbols
package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/engine/interfaces"
	"github.com/stacklok/mediator/internal/providers"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
	provifv1 "github.com/stacklok/mediator/pkg/providers/v1"
)

var (
	simpleBodyTemplate   = "{\"foo\": \"bar\"}"
	bodyTemplateWithVars = `{ "enabled": true, "allowed_actions": "{{.Profile.allowed_actions}}" }`
	invalidBodyTemplate  = "{\"foo\": {{bar}"
	validProviderBuilder = providers.NewProviderBuilder(
		&db.Provider{
			Name:    "github",
			Version: provifv1.V1,
			Implements: []db.ProviderType{
				db.ProviderTypeRest,
			},
			Definition: json.RawMessage(`{
	"rest": {
		"base_url": "https://api.github.com/"
	}
}`),
		},
		db.ProviderAccessToken{},
		"token",
	)
	invalidProviderBuilder = providers.NewProviderBuilder(
		&db.Provider{
			Name:    "github",
			Version: provifv1.V1,
			Implements: []db.ProviderType{
				db.ProviderTypeRest,
			},
			Definition: json.RawMessage(`{
	"rest": {
		"base_url": "https://api.github.com/"
}`),
		},
		db.ProviderAccessToken{},
		"token",
	)
)

func testGithubProviderBuilder(baseURL string) *providers.ProviderBuilder {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL + "/"
	}

	definitionJSON := `{
		"github": {
			"endpoint": "` + baseURL + `"
		}
	}`

	return providers.NewProviderBuilder(
		&db.Provider{
			Name:       "github",
			Version:    provifv1.V1,
			Implements: []db.ProviderType{db.ProviderTypeGithub, db.ProviderTypeRest},
			Definition: json.RawMessage(definitionJSON),
		},
		db.ProviderAccessToken{},
		"token",
	)
}

func TestNewRestRemediate(t *testing.T) {
	t.Parallel()

	type args struct {
		restCfg *pb.RestType
		pbuild  *providers.ProviderBuilder
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid rest remediatior",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "/repos/Foo/Bar",
					Body:     &simpleBodyTemplate,
				},
				pbuild: validProviderBuilder,
			},
			wantErr: false,
		},
		{
			name: "nondefault method",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "/repos/Foo/Bar",
					Body:     &simpleBodyTemplate,
					Method:   "POST",
				},
				pbuild: validProviderBuilder,
			},
			wantErr: false,
		},
		{
			name: "invalid endpoint template",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "/{{ .repos/Foo/Bar",
					Body:     &simpleBodyTemplate,
				},
				pbuild: validProviderBuilder,
			},
			wantErr: true,
		},
		{
			name: "invalid body template",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "/repos/Foo/Bar",
					Body:     &invalidBodyTemplate,
				},
				pbuild: validProviderBuilder,
			},
			wantErr: true,
		},
		{
			name: "nil body template",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "/repos/Foo/Bar",
					Body:     nil,
				},
				pbuild: validProviderBuilder,
			},
			wantErr: true,
		},
		{
			name: "invalid provider builder",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "/repos/Foo/Bar",
					Body:     &simpleBodyTemplate,
				},
				pbuild: invalidProviderBuilder,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewRestRemediate(tt.args.restCfg, tt.args.pbuild)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Nil(t, got, "expected nil")
				return
			}

			if tt.args.restCfg.Method != "" {
				require.Equal(t, tt.args.restCfg.Method, got.method, "unexpected method")
			}
			require.NoError(t, err, "unexpected error")
			require.NotNil(t, got, "expected non-nil")
		})
	}
}
func TestRestRemediate(t *testing.T) {
	t.Parallel()

	type remediateArgs struct {
		remAction interfaces.RemediateActionOpt
		ent       protoreflect.ProtoMessage
		pol       map[string]any
	}

	type newRestRemediateArgs struct {
		restCfg *pb.RestType
		pbuild  *providers.ProviderBuilder
	}

	tests := []struct {
		name        string
		newRemArgs  newRestRemediateArgs
		remArgs     remediateArgs
		testHandler http.HandlerFunc
		wantErr     bool
	}{
		{
			name: "valid remediate",
			newRemArgs: newRestRemediateArgs{
				restCfg: &pb.RestType{
					Endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}/actions/permissions",
					Body:     &bodyTemplateWithVars,
				},
			},
			remArgs: remediateArgs{
				remAction: interfaces.ActionOptOn,
				ent: &pb.Repository{
					Owner:  "OwnerVar",
					Name:   "NameVar",
					RepoId: 456,
				},
				pol: map[string]any{
					"allowed_actions": "selected",
				},
			},
			testHandler: func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, "/repos/OwnerVar/NameVar/actions/permissions", request.URL.Path, "unexpected path")
				assert.Equal(t, http.MethodPatch, request.Method, "unexpected method")

				var requestBody struct {
					Enabled        bool   `json:"enabled"`
					AllowedActions string `json:"allowed_actions"`
				}

				err := json.NewDecoder(request.Body).Decode(&requestBody)
				assert.NoError(t, err, "unexpected error decoding body")
				assert.Equal(t, true, requestBody.Enabled, "unexpected enabled")
				assert.Equal(t, "selected", requestBody.AllowedActions, "unexpected allowed actions")

				defer request.Body.Close()
				writer.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name: "valid dry run",
			newRemArgs: newRestRemediateArgs{
				restCfg: &pb.RestType{
					Endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}/actions/permissions",
					Body:     &bodyTemplateWithVars,
				},
			},
			remArgs: remediateArgs{
				remAction: interfaces.ActionOptDryRun,
				ent: &pb.Repository{
					Owner:  "OwnerVar",
					Name:   "NameVar",
					RepoId: 456,
				},
				pol: map[string]any{
					"allowed_actions": "selected",
				},
			},
			testHandler: func(writer http.ResponseWriter, request *http.Request) {
				assert.Fail(t, "unexpected request")
			},
			wantErr: false,
		},
		{
			name: "remediate http handler error",
			newRemArgs: newRestRemediateArgs{
				restCfg: &pb.RestType{
					Endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}/actions/permissions",
					Body:     &bodyTemplateWithVars,
				},
			},
			remArgs: remediateArgs{
				remAction: interfaces.ActionOptOn,
				ent: &pb.Repository{
					Owner:  "OwnerVar",
					Name:   "NameVar",
					RepoId: 456,
				},
				pol: map[string]any{
					"allowed_actions": "selected",
				},
			},
			testHandler: func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(http.StatusForbidden)
				_, err := writer.Write([]byte("forbidden"))
				assert.NoError(t, err, "unexpected error writing response")
			},
			wantErr: true,
		},
		{
			name: "invalid remediate action",
			newRemArgs: newRestRemediateArgs{
				restCfg: &pb.RestType{
					Endpoint: "/repos/Foo/Bar",
					Body:     &simpleBodyTemplate,
				},
			},
			remArgs: remediateArgs{
				remAction: interfaces.ActionOptUnknown,
				ent: &pb.Repository{
					Owner:  "Foo",
					Name:   "Bar",
					RepoId: 123,
				},
				pol: map[string]any{
					"enabled": true,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			testServer := httptest.NewServer(tt.testHandler)
			defer testServer.Close()
			tt.newRemArgs.pbuild = testGithubProviderBuilder(testServer.URL)

			engine, err := NewRestRemediate(tt.newRemArgs.restCfg, tt.newRemArgs.pbuild)
			require.NoError(t, err, "unexpected error creating remediate engine")
			require.NotNil(t, engine, "expected non-nil remediate engine")

			err = engine.Remediate(context.Background(), tt.remArgs.remAction, tt.remArgs.ent, tt.remArgs.pol)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				return
			}

			require.NoError(t, err, "unexpected error creating remediate engine")
		})
	}
}
