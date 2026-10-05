// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"golang.org/x/oauth2"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
	testhelper "github.com/mindersec/minder/pkg/providers/v1/testing"
)

const (
	upstreamID = "test-upstream-id"
)

func TestRegistration(t *testing.T) {
	t.Parallel()
	// We don't need a full constructor here, so we're naughty
	glc := &gitlabClient{}
	testhelper.CheckRegistrationExcept(t, glc, minderv1.Entity_ENTITY_REPOSITORIES)
}

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
			props: properties.NewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == fmt.Sprintf("/projects/%s/hooks", upstreamID) {
						// handle cleanUpStaleWebhooks
						switch r.Method {
						case http.MethodGet:
							w.Header().Set("Content-Type", "application/json")

							w.WriteHeader(http.StatusOK)
							_, err := w.Write([]byte("[]"))
							assert.NoError(t, err)
							return
						case http.MethodPost:
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
			props: properties.NewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == fmt.Sprintf("/projects/%s/hooks", upstreamID) {
						// handle cleanUpStaleWebhooks
						switch r.Method {
						case http.MethodGet:
							w.WriteHeader(http.StatusInternalServerError)
							return
						case http.MethodPost:
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
			props: properties.NewProperties(map[string]any{
				properties.PropertyUpstreamID: upstreamID,
			}),
			mockHandler: func(t *testing.T) http.HandlerFunc {
				t.Helper()
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == fmt.Sprintf("/projects/%s/hooks", upstreamID) {
						// handle cleanUpStaleWebhooks
						switch r.Method {
						case http.MethodGet:
							w.Header().Set("Content-Type", "application/json")

							w.WriteHeader(http.StatusOK)
							_, err := w.Write([]byte("[]"))
							assert.NoError(t, err)
							return
						case http.MethodPost:
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
			props:      properties.NewProperties(map[string]any{}),
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
			props: properties.NewProperties(map[string]any{
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
			props: properties.NewProperties(map[string]any{
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
			props: properties.NewProperties(map[string]any{
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
			props: properties.NewProperties(map[string]any{
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
			props: properties.NewProperties(map[string]any{
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
			props: properties.NewProperties(map[string]any{
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

type mockCredentials struct{}

// ensure that mockCredentials implements the GitLabCredential interface
var _ provifv1.GitLabCredential = (*mockCredentials)(nil)

func (*mockCredentials) SetAuthorizationHeader(_ *http.Request) {
}

func (*mockCredentials) AddToPushOptions(_ *git.PushOptions, _ string) {
}

func (*mockCredentials) AddToCloneOptions(_ *git.CloneOptions) {
}

func (*mockCredentials) GetAsOAuth2TokenSource() oauth2.TokenSource {
	return nil
}

func TestCleanUpStaleWebhooksPaginates(t *testing.T) {
	t.Parallel()

	const webhookURL = "https://minder.example.com/api/v1/webhook/gitlab"
	hooksPath := "/projects/" + upstreamID + "/hooks"

	var mu sync.Mutex
	var deleted []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == hooksPath:
			assert.Equal(t, strconv.Itoa(perPage), r.URL.Query().Get("per_page"))
			// The stale Minder hook is only on the second page. Like GitLab,
			// treat a request without a page parameter as the first page.
			switch r.URL.Query().Get("page") {
			case "", "1":
				w.Header().Set(nextPageHeader, "2")
				_, err := fmt.Fprint(w, `[{"id": 1, "url": "https://ci.example.com/hook"}]`)
				assert.NoError(t, err)
			case "2":
				_, err := fmt.Fprintf(w, `[{"id": 2, "url": %q}]`, webhookURL+"/stale")
				assert.NoError(t, err)
			default:
				t.Errorf("unexpected hooks page %q", r.URL.Query().Get("page"))
				w.WriteHeader(http.StatusBadRequest)
			}
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, hooksPath+"/"):
			mu.Lock()
			deleted = append(deleted, strings.TrimPrefix(r.URL.Path, hooksPath+"/"))
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	cli := newTestGitlabProvider(ts.URL)
	cli.webhookURL = webhookURL

	require.NoError(t, cli.cleanUpStaleWebhooks(context.Background(), upstreamID))

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"2"}, deleted, "only the Minder hook on the second page should be deleted")
}
