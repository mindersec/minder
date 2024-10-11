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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
	"golang.org/x/oauth2"

	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

const (
	upstreamID = "test-upstream-id"
)

func TestRegisterEntity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		entityType  minderv1.Entity
		props       *properties.Properties
		mockHandler func(t *testing.T) http.HandlerFunc
		wantErr     bool
	}{
		{
			name:       "test register entity",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			props: MustNewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == fmt.Sprintf("/projects/%s/hooks", upstreamID) {
						// handle cleanUpStaleWebhooks
						if r.Method == http.MethodGet {
							w.Header().Set("Content-Type", "application/json")

							w.WriteHeader(http.StatusOK)
							_, err := w.Write([]byte("[]"))
							assert.NoError(t, err)
							return
						} else if r.Method == http.MethodPost {
							// handle createWebhook
							w.Header().Set("Content-Type", "application/json")

							outhook := &gitlab.Hook{
								ID: 1,
							}

							w.WriteHeader(http.StatusCreated)
							enc := json.NewEncoder(w)
							err := enc.Encode(outhook)
							assert.NoError(t, err)
							return
						}
					}
					w.WriteHeader(http.StatusNotFound)
				})
			},
		},
		{
			name:       "test register entity with error cleaning up stale webhooks still succeeds",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			props: MustNewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == fmt.Sprintf("/projects/%s/hooks", upstreamID) {
						// handle cleanUpStaleWebhooks
						if r.Method == http.MethodGet {
							w.WriteHeader(http.StatusInternalServerError)
							return
						} else if r.Method == http.MethodPost {
							// handle createWebhook
							w.Header().Set("Content-Type", "application/json")

							outhook := &gitlab.Hook{
								ID: 1,
							}

							enc := json.NewEncoder(w)
							err := enc.Encode(outhook)
							assert.NoError(t, err)

							w.WriteHeader(http.StatusCreated)
							return
						}
					}
					w.WriteHeader(http.StatusNotFound)
				})
			},
		},
		{
			name:       "test register entity with error creating webhook",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			props: MustNewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == fmt.Sprintf("/projects/%s/hooks", upstreamID) {
						// handle cleanUpStaleWebhooks
						if r.Method == http.MethodGet {
							w.Header().Set("Content-Type", "application/json")

							w.WriteHeader(http.StatusOK)
							_, err := w.Write([]byte("[]"))
							assert.NoError(t, err)
							return
						} else if r.Method == http.MethodPost {
							// handle createWebhook
							w.WriteHeader(http.StatusInternalServerError)
							return
						}
					}
					w.WriteHeader(http.StatusNotFound)
				})
			},
			wantErr: true,
		},
		{
			name:       "test register entity with missing upstream ID",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			props:      MustNewProperties(map[string]any{}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			wantErr: true,
		},
		{
			name:       "test register entity with unsupported entity type",
			entityType: minderv1.Entity_ENTITY_UNSPECIFIED,
			props: MustNewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocksrv := httptest.NewServer(tt.mockHandler(t))
			defer mocksrv.Close()

			cli := mocksrv.Client()

			glc := &gitlabClient{
				cred: &mockCredentials{},
				glcfg: &minderv1.GitLabProviderConfig{
					Endpoint: mocksrv.URL,
				},
				cli:                  cli,
				currentWebhookSecret: "test-secret",
			}

			ctx := context.Background()
			testlw := zerolog.NewTestWriter(t)

			// attach the logger to the context
			ctx = zerolog.New(testlw).With().Logger().WithContext(ctx)

			props, err := glc.RegisterEntity(ctx, tt.entityType, tt.props)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, props)
		})
	}
}

func TestDeregisterEntity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		entityType  minderv1.Entity
		props       *properties.Properties
		mockHandler func(t *testing.T) http.HandlerFunc
		wantErr     bool
	}{
		{
			name:       "test deregister entity",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			props: MustNewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
				RepoPropertyHookID:            "test-hook-id",
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == fmt.Sprintf("/projects/%s/hooks/test-hook-id", upstreamID) {
						if r.Method == http.MethodDelete {
							w.WriteHeader(http.StatusNoContent)
							return
						}
					}
					w.WriteHeader(http.StatusNotFound)
				})
			},
		},
		{
			name:       "test deregister entity with error",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			props: MustNewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
				RepoPropertyHookID:            "test-hook-id",
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			wantErr: true,
		},
		{
			name:       "test deregister entity with missing upstream ID",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			props: MustNewProperties(map[string]any{
				RepoPropertyHookID: "test-hook-id",
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			wantErr: true,
		},
		{
			name:       "test deregister entity with missing hook ID",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			props: MustNewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			wantErr: true,
		},
		{
			name:       "test deregister entity with unsupported entity type",
			entityType: minderv1.Entity_ENTITY_UNSPECIFIED,
			props: MustNewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
				RepoPropertyHookID:            "test-hook-id",
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocksrv := httptest.NewServer(tt.mockHandler(t))
			defer mocksrv.Close()

			cli := mocksrv.Client()

			glc := &gitlabClient{
				cred: &mockCredentials{},
				glcfg: &minderv1.GitLabProviderConfig{
					Endpoint: mocksrv.URL,
				},
				cli:                  cli,
				currentWebhookSecret: "test-secret",
			}

			ctx := context.Background()
			testlw := zerolog.NewTestWriter(t)

			// attach the logger to the context
			ctx = zerolog.New(testlw).With().Logger().WithContext(ctx)

			err := glc.DeregisterEntity(ctx, tt.entityType, tt.props)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestReregisterEntity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		entityType  minderv1.Entity
		props       *properties.Properties
		mockHandler func(t *testing.T) http.HandlerFunc
		wantErr     bool
	}{
		{
			name:       "test reregister entity",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			props: MustNewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
				RepoPropertyHookID:            "test-hook-id",
				RepoPropertyHookURL:           fmt.Sprintf("http://test-hook-url/%s", uuid.New().String()),
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == fmt.Sprintf("/projects/%s/hooks/test-hook-id", upstreamID) {
						if r.Method == http.MethodPut {
							w.WriteHeader(http.StatusOK)
							return
						}
					}
					w.WriteHeader(http.StatusNotFound)
				})
			},
		},
		{
			name:       "test reregister entity with error",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			props: MustNewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
				RepoPropertyHookID:            "test-hook-id",
				RepoPropertyHookURL:           "http://test-hook-url",
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			wantErr: true,
		},
		{
			name:       "test reregister entity with missing upstream ID",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			props: MustNewProperties(map[string]any{
				RepoPropertyHookID:  "test-hook-id",
				RepoPropertyHookURL: "http://test-hook-url",
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			wantErr: true,
		},
		{
			name:       "test reregister entity with missing hook ID",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			props: MustNewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
				RepoPropertyHookURL:           "http://test-hook-url",
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			wantErr: true,
		},
		{
			name:       "test reregister entity with missing hook URL",
			entityType: minderv1.Entity_ENTITY_REPOSITORIES,
			props: MustNewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
				RepoPropertyHookID:            "test-hook-id",
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			wantErr: true,
		},
		{
			name:       "test reregister entity with unsupported entity type",
			entityType: minderv1.Entity_ENTITY_UNSPECIFIED,
			props: MustNewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
				RepoPropertyHookID:            "test-hook-id",
				RepoPropertyHookURL:           "http://test-hook-url",
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocksrv := httptest.NewServer(tt.mockHandler(t))
			defer mocksrv.Close()

			cli := mocksrv.Client()

			glc := &gitlabClient{
				cred: &mockCredentials{},
				glcfg: &minderv1.GitLabProviderConfig{
					Endpoint: mocksrv.URL,
				},
				cli:                  cli,
				currentWebhookSecret: "test-secret",
			}

			ctx := context.Background()
			testlw := zerolog.NewTestWriter(t)

			// attach the logger to the context
			ctx = zerolog.New(testlw).With().Logger().WithContext(ctx)

			err := glc.ReregisterEntity(ctx, tt.entityType, tt.props)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

type mockCredentials struct{}

// ensure that mockCredentials implements the GitLabCredential interface
var _ provifv1.GitLabCredential = (*mockCredentials)(nil)

func (_ *mockCredentials) SetAuthorizationHeader(_ *http.Request) {
}

func (_ *mockCredentials) AddToPushOptions(_ *git.PushOptions, _ string) {
}

func (_ *mockCredentials) AddToCloneOptions(_ *git.CloneOptions) {
}

func (_ *mockCredentials) GetAsOAuth2TokenSource() oauth2.TokenSource {
	return nil
}
