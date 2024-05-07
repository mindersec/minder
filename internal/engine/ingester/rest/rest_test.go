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
// Package rule provides the CLI subcommand for managing rules

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

	engif "github.com/stacklok/minder/internal/engine/interfaces"
	"github.com/stacklok/minder/internal/providers/credentials"
	"github.com/stacklok/minder/internal/providers/github/clients"
	httpclient "github.com/stacklok/minder/internal/providers/http"
	"github.com/stacklok/minder/internal/providers/ratecache"
	"github.com/stacklok/minder/internal/providers/telemetry"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

func TestNewRestRuleDataIngest(t *testing.T) {
	t.Parallel()

	type args struct {
		restCfg *pb.RestType
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid rest",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "/repos/Foo/Bar",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid template",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "{{",
				},
			},
			wantErr: true,
		},
		{
			name: "empty endpoint",
			args: args{
				restCfg: &pb.RestType{
					Endpoint: "",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rest, err := httpclient.NewREST(
				&pb.RESTProviderConfig{
					BaseUrl: "https://api.github.com/",
				},
				telemetry.NewNoopMetrics(),
				credentials.NewGitHubTokenCredential("token"),
			)
			require.NoError(t, err)

			got, err := NewRestRuleDataIngest(tt.args.restCfg, rest)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				require.Nil(t, got, "expected nil")
				return
			}

			require.NoError(t, err, "unexpected error")
			require.NotNil(t, got, "expected non-nil")
		})
	}
}

func testGithubProviderBuilder(baseURL string) (provifv1.REST, error) {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL + "/"
	}

	return clients.NewRestClient(
		&pb.GitHubProviderConfig{
			Endpoint: baseURL,
		},
		&ratecache.NoopRestClientCache{},
		credentials.NewGitHubTokenCredential("token"),
		clients.NewGitHubClientFactory(telemetry.NewNoopMetrics()),
		"",
	)
}

const (
	validProtectionReply = `
{
  "url": "https://api.github.com/repos/jakubtestorg/testrepo/branches/main/protection",
  "required_pull_request_reviews": {
    "url": "https://api.github.com/repos/jakubtestorg/testrepo/branches/main/protection/required_pull_request_reviews",
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": true,
    "require_last_push_approval": false,
    "required_approving_review_count": 2
  },
  "required_signatures": {
    "url": "https://api.github.com/repos/jakubtestorg/testrepo/branches/main/protection/required_signatures",
    "enabled": false
  },
  "enforce_admins": {
    "url": "https://api.github.com/repos/jakubtestorg/testrepo/branches/main/protection/enforce_admins",
    "enabled": false
  },
  "required_linear_history": {
    "enabled": false
  },
  "allow_force_pushes": {
    "enabled": false
  },
  "allow_deletions": {
    "enabled": false
  },
  "block_creations": {
    "enabled": false
  },
  "required_conversation_resolution": {
    "enabled": false
  },
  "lock_branch": {
    "enabled": false
  },
  "allow_fork_syncing": {
    "enabled": false
  }
}
`
	notFoundReply = `{"message": "Not Found"}`
)

func TestRestIngest(t *testing.T) {
	t.Parallel()

	type ingestArgs struct {
		ent    protoreflect.ProtoMessage
		params map[string]any
	}

	type newRestIngestArgs struct {
		restCfg *pb.RestType
	}

	tests := []struct {
		name        string
		newIngArgs  newRestIngestArgs
		ingArgs     ingestArgs
		testHandler http.HandlerFunc
		ingResultFn func() *engif.Result
		wantErr     bool
	}{
		{
			name: "valid ingest",
			newIngArgs: newRestIngestArgs{
				restCfg: &pb.RestType{
					Endpoint: `/repos/{{.Entity.Owner}}/{{.Entity.Name}}/branches/{{ index .Params "branch" }}/protection`,
					Parse:    "json",
				},
			},
			ingArgs: ingestArgs{
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
				assert.Equal(t, "/repos/OwnerVar/NameVar/branches/main/protection", request.URL.Path, "unexpected path")
				assert.Equal(t, http.MethodGet, request.Method, "unexpected method")

				_, err := writer.Write([]byte(validProtectionReply))
				assert.NoError(t, err, "unexpected error writing response")
				writer.WriteHeader(http.StatusOK)
			},
			ingResultFn: func() *engif.Result {
				var jReply any
				if err := json.NewDecoder(strings.NewReader(validProtectionReply)).Decode(&jReply); err != nil {
					return nil
				}

				return &engif.Result{
					Object: jReply,
				}
			},
			wantErr: false,
		},
		{
			name: "test fallback",
			newIngArgs: newRestIngestArgs{
				restCfg: &pb.RestType{
					Endpoint: `/repos/{{.Entity.Owner}}/{{.Entity.Name}}/branches/{{ index .Params "branch" }}/protection`,
					Parse:    "json",
					Fallback: []*pb.RestType_Fallback{
						{
							HttpCode: http.StatusNotFound,
							Body:     notFoundReply,
						},
					},
				},
			},
			ingArgs: ingestArgs{
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
				assert.Equal(t, "/repos/OwnerVar/NameVar/branches/main/protection", request.URL.Path, "unexpected path")
				assert.Equal(t, http.MethodGet, request.Method, "unexpected method")

				writer.WriteHeader(http.StatusNotFound)
			},
			ingResultFn: func() *engif.Result {
				var jReply any
				if err := json.NewDecoder(strings.NewReader(notFoundReply)).Decode(&jReply); err != nil {
					return nil
				}

				return &engif.Result{
					Object: jReply,
				}
			},
			wantErr: false,
		},
		{
			name: "test http error",
			newIngArgs: newRestIngestArgs{
				restCfg: &pb.RestType{
					Endpoint: `/repos/{{.Entity.Owner}}/{{.Entity.Name}}/branches/{{ index .Params "branch" }}/protection`,
					Parse:    "json",
				},
			},
			ingArgs: ingestArgs{
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
				assert.Equal(t, "/repos/OwnerVar/NameVar/branches/main/protection", request.URL.Path, "unexpected path")
				assert.Equal(t, http.MethodGet, request.Method, "unexpected method")

				writer.WriteHeader(http.StatusForbidden)
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

			rest, err := testGithubProviderBuilder(testServer.URL)
			require.NoError(t, err)
			engine, err := NewRestRuleDataIngest(tt.newIngArgs.restCfg, rest)
			require.NoError(t, err, "unexpected error creating ingestion engine")
			require.NotNil(t, engine, "expected non-nil ingestion engine")

			result, err := engine.Ingest(context.Background(), tt.ingArgs.ent, tt.ingArgs.params)
			if tt.wantErr {
				require.Error(t, err, "expected error")
				return
			}

			require.NoError(t, err, "unexpected error creating remediate engine")
			require.Equal(t, tt.ingResultFn(), result, "unexpected result")
		})
	}
}
