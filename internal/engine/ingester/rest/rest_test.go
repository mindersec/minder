// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rest

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/internal/providers/credentials"
	"github.com/mindersec/minder/internal/providers/github/clients"
	"github.com/mindersec/minder/internal/providers/github/properties"
	"github.com/mindersec/minder/internal/providers/ratecache"
	"github.com/mindersec/minder/internal/providers/telemetry"
	"github.com/mindersec/minder/internal/providers/testproviders"
	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
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

			rest, err := testproviders.NewRESTProvider(
				&pb.RESTProviderConfig{
					BaseUrl: proto.String("https://api.github.com/"),
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

	graphQLQuery := `{"query":"query { repository(owner: \"{{.Entity.Owner}}\", name: \"{{.Entity.Name}}\") { id } }"}`
	graphQLExpected := `{"query":"query { repository(owner: \"OwnerVar\", name: \"NameVar\") { id } }"}` + "\n"
	graphQLReply := []byte(`{"data": {"repository": {"id": 456}}}`)

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
		ingResultFn func() *interfaces.Ingested
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
			ingResultFn: func() *interfaces.Ingested {
				var jReply any
				if err := json.NewDecoder(strings.NewReader(validProtectionReply)).Decode(&jReply); err != nil {
					return nil
				}

				return &interfaces.Ingested{
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
			ingResultFn: func() *interfaces.Ingested {
				var jReply any
				if err := json.NewDecoder(strings.NewReader(notFoundReply)).Decode(&jReply); err != nil {
					return nil
				}

				return &interfaces.Ingested{
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
		{
			name: "test body templates",
			newIngArgs: newRestIngestArgs{
				restCfg: &pb.RestType{
					Endpoint: `/graphql`,
					Method:   "POST",
					Parse:    "json",
					Body:     &graphQLQuery,
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
				defer request.Body.Close()
				assert.Equal(t, "/graphql", request.URL.Path, "unexpected path")
				assert.Equal(t, http.MethodPost, request.Method, "unexpected method")
				var body bytes.Buffer
				if _, err := io.Copy(&body, request.Body); err != nil {
					t.Errorf("Failed to read request body: %v", err)
				}
				assert.Equal(t, graphQLExpected, body.String())

				_, err := writer.Write(graphQLReply)
				assert.NoError(t, err, "unexpected error writing response")
				writer.WriteHeader(http.StatusOK)
			},
			ingResultFn: func() *interfaces.Ingested {
				ret := &interfaces.Ingested{}
				if err := json.Unmarshal(graphQLReply, &ret.Object); err != nil {
					return nil
				}
				return ret
			},
			wantErr: false,
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
			require.Equal(t, tt.ingResultFn().Object, result.Object, "unexpected result")
		})
	}
}

func TestIngestor_parseBodyDoesNotReadTooLargeRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *pb.RestType
		body    string
		wantErr bool
	}{
		{
			name: "raw",
			cfg:  &pb.RestType{},
			// really large body
			// casting to `int` should work.
			body: strings.Repeat("a", int(MaxBytesLimit)+1),
			// This case does not error, it simply truncates.
			wantErr: false,
		},
		{
			name: "json",
			cfg: &pb.RestType{
				Parse: "json",
			},
			// really large body
			// casting to `int` should work.
			body: "{\"a\":\"" + strings.Repeat("a", int(MaxBytesLimit)+1) + "\"}",
			// This case will error out, as truncating
			// makes the JSON invalid.
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rdi := &Ingestor{
				restCfg: tt.cfg,
			}

			got, err := rdi.parseBody(strings.NewReader(tt.body))
			if tt.wantErr {
				assert.Error(t, err, "expected error")
			} else {
				assert.NoError(t, err, "expected no error")
				assert.Equal(t, int(MaxBytesLimit), binary.Size(got), "expected body to be truncated")
			}
		})
	}
}
