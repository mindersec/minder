// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package rest provides tests for the REST remediation engine
// we use the package rest directly because we need to test non-exported symbols
package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/engine/interfaces"
	"github.com/mindersec/minder/internal/providers/credentials"
	"github.com/mindersec/minder/internal/providers/github/clients"
	"github.com/mindersec/minder/internal/providers/github/properties"
	"github.com/mindersec/minder/internal/providers/ratecache"
	"github.com/mindersec/minder/internal/providers/telemetry"
	"github.com/mindersec/minder/internal/providers/testproviders"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	engif "github.com/mindersec/minder/pkg/engine/v1/interfaces"
	"github.com/mindersec/minder/pkg/profiles/models"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

var (
	simpleBodyTemplate                         = "{\"foo\": \"bar\"}"
	bodyTemplateWithVars                       = `{ "enabled": true, "allowed_actions": "{{.Profile.allowed_actions}}" }`
	invalidBodyTemplate                        = "{\"foo\": {{bar}"
	TestActionTypeValid  interfaces.ActionType = "remediate-test"
)

func testGithubProvider(baseURL string) (provifv1.REST, error) {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL + "/"
	}

	return clients.NewRestClient(
		&pb.GitHubProviderConfig{
			Endpoint: &baseURL,
		},
		nil,
		nil,
		&ratecache.NoopRestClientCache{},
		credentials.NewGitHubTokenCredential("token"),
		clients.NewGitHubClientFactory(telemetry.NewNoopMetrics()),
		properties.NewPropertyFetcherFactory(),
		"",
	)
}

func TestNewRestRemediate(t *testing.T) {
	t.Parallel()

	type args struct {
		restCfg    *pb.RestType
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
				actionType: TestActionTypeValid,
			},
			wantErr: true,
		},
		{
			name: "invalid method template",
			args: args{
				restCfg: &pb.RestType{
					// No {{end}}
					Method:   "{{if .EvalResultOutput.doPatch}}patch{{else}}put",
					Endpoint: "/graphql",
					Body:     &simpleBodyTemplate,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			restProvider, err := testproviders.NewRESTProvider(
				&pb.RESTProviderConfig{
					BaseUrl: proto.String("https://api.github.com/"),
				},
				telemetry.NewNoopMetrics(),
				credentials.NewGitHubTokenCredential("token"),
			)
			require.NoError(t, err)

			got, err := NewRestRemediate(
				tt.args.actionType, tt.args.restCfg, restProvider, models.ActionOptOn)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Nil(t, got, "expected nil")
				return
			}

			require.NoError(t, err, "unexpected error")
			require.NotNil(t, got, "expected non-nil")
			methodBytes := new(bytes.Buffer)
			require.NoError(t, got.method.Execute(context.Background(), methodBytes, nil, 10))
			// We don't actually do any template evaluation here, just check that pass-through worked
			if tt.args.restCfg.Method != "" {
				require.Equal(t, tt.args.restCfg.Method, methodBytes.String())
			}
		})
	}
}

func TestRestRemediate(t *testing.T) {
	t.Parallel()

	type remediateArgs struct {
		remAction models.ActionOpt
		ent       protoreflect.ProtoMessage
		pol       map[string]any
		params    map[string]any
		output    any
	}

	type newRestRemediateArgs struct {
		restCfg *pb.RestType
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
				remAction: models.ActionOptOn,
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
				remAction: models.ActionOptOn,
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
				remAction: models.ActionOptOn,
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
			name: "PUT or POST on Eval Output",
			newRemArgs: newRestRemediateArgs{
				restCfg: &pb.RestType{
					Method:   `{{if .EvalResultOutput.Id}}PUT{{else}}POST{{end}}`,
					Endpoint: `/repos/{{.Entity.Owner}}/{{.Entity.Name}}/rulesets{{if .EvalResultOutput.Id}}/{{ .EvalResultOutput.Id }}{{end}}`,
					Body:     &bodyTemplateWithVars,
				},
			},
			remArgs: remediateArgs{
				remAction: models.ActionOptOn,
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
				output: map[string]any{"Id": "1234"},
			},
			testHandler: func(writer http.ResponseWriter, request *http.Request) {
				assert.Equal(t, "/repos/OwnerVar/NameVar/rulesets/1234", request.URL.Path, "unexpected path")
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
			wantErr: false},
		{
			name: "valid dry run",
			newRemArgs: newRestRemediateArgs{
				restCfg: &pb.RestType{
					Endpoint: "/repos/{{.Entity.Owner}}/{{.Entity.Name}}/actions/permissions",
					Body:     &bodyTemplateWithVars,
				},
			},
			remArgs: remediateArgs{
				remAction: models.ActionOptDryRun,
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
				remAction: models.ActionOptOn,
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
				remAction: models.ActionOptUnknown,
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
			provider, err := testGithubProvider(testServer.URL)
			require.NoError(t, err)
			engine, err := NewRestRemediate(
				TestActionTypeValid, tt.newRemArgs.restCfg, provider, tt.remArgs.remAction)
			require.NoError(t, err, "unexpected error creating remediate engine")
			require.NotNil(t, engine, "expected non-nil remediate engine")

			evalParams := &interfaces.EvalStatusParams{
				Rule: &models.RuleInstance{
					Def:    tt.remArgs.pol,
					Params: tt.remArgs.params,
				},
			}
			if tt.remArgs.output != nil {
				res := engif.EvaluationResult{
					Output: tt.remArgs.output,
				}
				evalParams.SetEvalResult(&res)
			}

			retMeta, err := engine.Do(
				context.Background(), interfaces.ActionCmdOn, tt.remArgs.ent,
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
