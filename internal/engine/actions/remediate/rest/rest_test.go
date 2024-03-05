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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

var (
	simpleBodyTemplate   = "{\"foo\": \"bar\"}"
	bodyTemplateWithVars = `{ "enabled": true, "allowed_actions": "{{.Profile.allowed_actions}}" }`
	invalidBodyTemplate  = "{\"foo\": {{bar}"
	validProviderBuilder = providers.NewProviderBuilder(
		&db.Provider{
			Name:    "github",
			Version: provifv1.V1,
			Implements: []db.ProviderTrait{
				db.ProviderTraitRest,
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
			Implements: []db.ProviderTrait{
				db.ProviderTraitRest,
			},
			Definition: json.RawMessage(`{
	"rest": {
		"base_url": "https://api.github.com/"
}`),
		},
		db.ProviderAccessToken{},
		"token",
	)
	TestActionTypeValid interfaces.ActionType = "remediate-test"
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
			Implements: []db.ProviderTrait{db.ProviderTraitGithub, db.ProviderTraitRest},
			Definition: json.RawMessage(definitionJSON),
		},
		db.ProviderAccessToken{},
		"token",
	)
}

func TestNewRestRemediate(t *testing.T) {
	t.Parallel()

	type args struct {
		restCfg    *pb.RestType
		pbuild     *providers.ProviderBuilder
		actionType interfaces.ActionType
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid action type",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "/repos/Foo/Bar",
					Body:     &simpleBodyTemplate,
				},
				pbuild:     validProviderBuilder,
				actionType: "",
			},
			wantErr: true,
		},
		{
			name: "valid rest remediatior",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "/repos/Foo/Bar",
					Body:     &simpleBodyTemplate,
				},
				pbuild:     validProviderBuilder,
				actionType: TestActionTypeValid,
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
				pbuild:     validProviderBuilder,
				actionType: TestActionTypeValid,
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
				pbuild:     validProviderBuilder,
				actionType: TestActionTypeValid,
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
				pbuild:     validProviderBuilder,
				actionType: TestActionTypeValid,
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
				pbuild:     invalidProviderBuilder,
				actionType: TestActionTypeValid,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewRestRemediate(tt.args.actionType, tt.args.restCfg, tt.args.pbuild)
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
		remAction interfaces.ActionOpt
		ent       protoreflect.ProtoMessage
		pol       map[string]any
		params    map[string]any
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
			name: "valid remediate with PUT and no body",
			newRemArgs: newRestRemediateArgs{
				restCfg: &pb.RestType{
					Endpoint: `/repos/{{.Entity.Owner}}/{{.Entity.Name}}/branches/{{ index .Params "branch" }}/protection/required_signatures`,
					Method:   http.MethodPut,
				},
			},
			remArgs: remediateArgs{
				remAction: interfaces.ActionOptOn,
				ent: &pb.Repository{
					Owner:  "OwnerVar",
					Name:   "NameVar",
					RepoId: 456,
				},
				params: map[string]any{
					"branch": "main",
				},
			},
			testHandler: func(writer http.ResponseWriter, request *http.Request) {
				defer request.Body.Close()

				assert.Equal(t, "/repos/OwnerVar/NameVar/branches/main/protection/required_signatures", request.URL.Path, "unexpected path")
				assert.Equal(t, http.MethodPut, request.Method, "unexpected method")

				var requestBody struct{}
				err := json.NewDecoder(request.Body).Decode(&requestBody)
				assert.NoError(t, err, "unexpected error reading body")

				writer.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name: "valid remediate expanding a branch from parameters",
			newRemArgs: newRestRemediateArgs{
				restCfg: &pb.RestType{
					Endpoint: `/repos/{{.Entity.Owner}}/{{.Entity.Name}}/branches/{{ index .Params "branch" }}/protection`,
					Body:     &bodyTemplateWithVars,
					Method:   http.MethodPut,
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
				params: map[string]any{
					"branch": "main",
				},
			},
			testHandler: func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, "/repos/OwnerVar/NameVar/branches/main/protection", request.URL.Path, "unexpected path")
				assert.Equal(t, http.MethodPut, request.Method, "unexpected method")

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
			testHandler: func(_ http.ResponseWriter, _ *http.Request) {
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
			testHandler: func(writer http.ResponseWriter, _ *http.Request) {
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
			engine, err := NewRestRemediate(TestActionTypeValid, tt.newRemArgs.restCfg, tt.newRemArgs.pbuild)
			require.NoError(t, err, "unexpected error creating remediate engine")
			require.NotNil(t, engine, "expected non-nil remediate engine")

			structPol, err := structpb.NewStruct(tt.remArgs.pol)
			if err != nil {
				fmt.Printf("Error creating Struct: %v\n", err)
				return
			}
			structParams, err := structpb.NewStruct(tt.remArgs.params)
			if err != nil {
				fmt.Printf("Error creating Struct: %v\n", err)
				return
			}
			evalParams := &interfaces.EvalStatusParams{
				Rule: &pb.Profile_Rule{
					Def:    structPol,
					Params: structParams,
				},
			}

			retMeta, err := engine.Do(context.Background(), interfaces.ActionCmdOn, tt.remArgs.remAction, tt.remArgs.ent,
				evalParams, nil)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Nil(t, retMeta, "expected nil metadata")
				return
			}

			require.NoError(t, err, "unexpected error creating remediate engine")
		})
	}
}
