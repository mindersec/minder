// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package quay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers/credentials"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

func TestNew(t *testing.T) {
	t.Parallel()

	cred := credentials.NewOAuth2TokenCredential("test-quay-token")
	cfg := &minderv1.QuayProviderConfig{
		Namespace: proto.String("quay-org"),
	}

	lister, err := New(cred, cfg)
	require.NoError(t, err)
	require.NotNil(t, lister)
	assert.Equal(t, "quay-org", lister.namespace)
	assert.Equal(t, "https://quay.io/api/v1/repository?namespace=quay-org", lister.GetNamespaceURL())
}

func TestParseV1Config(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		rawCfg        string
		expectNilCfg  bool
		expectedOwner string
		expectErr     bool
		errContains   string
	}{
		{
			name: "valid configuration with namespace",
			rawCfg: `{
				"quay": {
					"namespace": "redhat-corp"
				}
			}`,
			expectNilCfg:  false,
			expectedOwner: "redhat-corp",
			expectErr:     false,
		},
		{
			name: "missing required namespace fails protobuf validation",
			rawCfg: `{
				"quay": {}
			}`,
			expectNilCfg: true,
			expectErr:    true,
			errContains:  "namespace is required",
		},
		{
			name: "empty namespace string fails protobuf validation",
			rawCfg: `{
				"quay": {
					"namespace": ""
				}
			}`,
			expectNilCfg: true,
			expectErr:    true,
			errContains:  "namespace is required",
		},
		{
			name:         "invalid json payload",
			rawCfg:       `{invalid-json: `,
			expectNilCfg: true,
			expectErr:    true,
		},
		{
			name: "missing quay block fails schema validation",
			rawCfg: `{
				"other_provider": {
					"namespace": "test"
				}
			}`,
			expectNilCfg: true,
			expectErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := ParseV1Config(json.RawMessage(tc.rawCfg))
			if tc.expectErr {
				require.Error(t, err)
				assert.Nil(t, cfg)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tc.expectNilCfg {
				assert.Nil(t, cfg)
			} else {
				require.NotNil(t, cfg)
				assert.Equal(t, tc.expectedOwner, cfg.GetNamespace())
			}
		})
	}
}

func TestMarshalV1Config(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		rawCfg      string
		expectErr   bool
		errContains string
	}{
		{
			name: "valid config marshals cleanly",
			rawCfg: `{
				"quay": {
					"namespace": "redhat-corp"
				}
			}`,
			expectErr: false,
		},
		{
			name: "validation failure on empty namespace",
			rawCfg: `{
				"quay": {
					"namespace": ""
				}
			}`,
			expectErr:   true,
			errContains: "namespace is required",
		},
		{
			name:        "invalid json syntax",
			rawCfg:      `{"quay": `,
			expectErr:   true,
			errContains: "unexpected end of JSON input",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			marshaled, err := MarshalV1Config(json.RawMessage(tc.rawCfg))
			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			var w quayConfigWrapper
			require.NoError(t, json.Unmarshal(marshaled, &w))
			require.NotNil(t, w.Quay)
			assert.Equal(t, "redhat-corp", w.Quay.GetNamespace())
		})
	}
}

func TestCanImplement(t *testing.T) {
	t.Parallel()

	q := &quayImageLister{}

	assert.True(t, q.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_IMAGE_LISTER))
	assert.True(t, q.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_OCI))
	assert.False(t, q.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_GITHUB))
	assert.False(t, q.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_GIT))
	assert.False(t, q.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_REST))
}

func TestListImages(t *testing.T) {
	t.Parallel()

	t.Run("success returns container names", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/repository", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "quay-org", r.URL.Query().Get("namespace"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"repositories": [
					{"name": "auth-service"},
					{"name": "event-collector"}
				]
			}`))
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		targetURL, err := url.Parse(server.URL + "/api/v1/repository?namespace=quay-org")
		require.NoError(t, err)

		q := &quayImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := q.ListImages(context.Background())
		require.NoError(t, err)
		assert.Equal(t, []string{"auth-service", "event-collector"}, images)
	})

	t.Run("follows next_page across two pages", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/repository", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "paged-org", r.URL.Query().Get("namespace"))
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Query().Get("next_page") == "page-2-token" {
				_, _ = w.Write([]byte(`{"repositories": [{"name": "page-two-image"}]}`))
				return
			}
			assert.Empty(t, r.URL.Query().Get("next_page"))
			_, _ = w.Write([]byte(`{"next_page": "page-2-token", "repositories": [{"name": "page-one-a"}, {"name": "page-one-b"}]}`))
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		targetURL, err := url.Parse(server.URL + "/api/v1/repository?namespace=paged-org")
		require.NoError(t, err)

		q := &quayImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := q.ListImages(context.Background())
		require.NoError(t, err)
		assert.Equal(t, []string{"page-one-a", "page-one-b", "page-two-image"}, images)
	})

	t.Run("empty repository list returns empty slice", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/repository", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"repositories": []}`))
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		targetURL, err := url.Parse(server.URL + "/api/v1/repository?namespace=empty-org")
		require.NoError(t, err)

		q := &quayImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := q.ListImages(context.Background())
		require.NoError(t, err)
		assert.Empty(t, images)
	})

	t.Run("401 unauthorized returns formatted error", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/repository", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error": "Unauthorized access"}`))
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		targetURL, err := url.Parse(server.URL + "/api/v1/repository?namespace=private-org")
		require.NoError(t, err)

		q := &quayImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := q.ListImages(context.Background())
		require.Error(t, err)
		assert.Nil(t, images)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("404 not found returns not found error", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/repository", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "Not Found"}`))
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		targetURL, err := url.Parse(server.URL + "/api/v1/repository?namespace=nonexistent-org")
		require.NoError(t, err)

		q := &quayImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := q.ListImages(context.Background())
		require.Error(t, err)
		assert.Nil(t, images)
		assert.Equal(t, "not found", err.Error())
	})

	t.Run("unexpected status code returns error", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/repository", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error": "Internal Error"}`))
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		targetURL, err := url.Parse(server.URL + "/api/v1/repository?namespace=error-org")
		require.NoError(t, err)

		q := &quayImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := q.ListImages(context.Background())
		require.Error(t, err)
		assert.Nil(t, images)
		assert.Contains(t, err.Error(), "unexpected status code: 500")
	})

	t.Run("malformed json response returns decode error", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/api/v1/repository", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{corrupt-json-stream`))
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		targetURL, err := url.Parse(server.URL + "/api/v1/repository?namespace=corrupt-org")
		require.NoError(t, err)

		q := &quayImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := q.ListImages(context.Background())
		require.Error(t, err)
		assert.Nil(t, images)
		assert.Contains(t, err.Error(), "error decoding response")
	})

	t.Run("canceled context returns error", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		targetURL, err := url.Parse(server.URL)
		require.NoError(t, err)

		q := &quayImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := q.ListImages(ctx)
		require.Error(t, err)
		assert.Nil(t, images)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

func TestProviderLifecycleMethods(t *testing.T) {
	t.Parallel()

	q := &quayImageLister{namespace: "test-ns"}
	ctx := context.Background()

	t.Run("CreationOptions for supported vs unsupported entity", func(t *testing.T) {
		t.Parallel()

		opts := q.CreationOptions(minderv1.Entity_ENTITY_ARTIFACTS)
		require.NotNil(t, opts)
		assert.False(t, opts.RegisterWithProvider)
		assert.False(t, opts.PublishReconciliationEvent)

		unsupportedOpts := q.CreationOptions(minderv1.Entity_ENTITY_REPOSITORIES)
		assert.Nil(t, unsupportedOpts)
	})

	t.Run("RegisterEntity returns props for artifacts and error for others", func(t *testing.T) {
		t.Parallel()

		props := properties.NewProperties(map[string]any{"key": "value"})

		registered, err := q.RegisterEntity(ctx, minderv1.Entity_ENTITY_ARTIFACTS, props)
		require.NoError(t, err)
		assert.Equal(t, props, registered)

		unsupportedReg, err := q.RegisterEntity(ctx, minderv1.Entity_ENTITY_REPOSITORIES, props)
		require.ErrorIs(t, err, provifv1.ErrUnsupportedEntity)
		assert.Nil(t, unsupportedReg)
	})

	t.Run("DeregisterEntity returns nil", func(t *testing.T) {
		t.Parallel()

		props := properties.NewProperties(map[string]any{"key": "value"})
		err := q.DeregisterEntity(ctx, minderv1.Entity_ENTITY_ARTIFACTS, props)
		require.NoError(t, err)
	})

	t.Run("FetchProperty returns nil", func(t *testing.T) {
		t.Parallel()

		props := properties.NewProperties(map[string]any{"key": "value"})
		prop, err := q.FetchProperty(ctx, props, minderv1.Entity_ENTITY_ARTIFACTS, "some_prop")
		require.NoError(t, err)
		assert.Nil(t, prop)
	})
}

func TestWebhookHandler(t *testing.T) {
	t.Parallel()

	pcm := &providerClassManager{}
	assert.Nil(t, pcm.GetWebhookHandler())
}

func TestProviderClassManager(t *testing.T) {
	t.Parallel()

	pcm := NewQuayProviderClassManager(nil, nil)
	require.NotNil(t, pcm)

	t.Run("GetSupportedClasses", func(t *testing.T) {
		t.Parallel()
		classes := pcm.GetSupportedClasses()
		assert.Equal(t, []db.ProviderClass{db.ProviderClassQuay}, classes)
	})

	t.Run("GetProviderClassInfo", func(t *testing.T) {
		t.Parallel()

		info, err := pcm.GetProviderClassInfo(db.ProviderClassQuay)
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.Equal(t, Quay, info.Class)

		unsupportedInfo, err := pcm.GetProviderClassInfo(db.ProviderClassGithub)
		require.Error(t, err)
		assert.Nil(t, unsupportedInfo)
		assert.Contains(t, err.Error(), "provider does not implement github")
	})

	t.Run("Delete returns nil", func(t *testing.T) {
		t.Parallel()
		err := pcm.Delete(context.Background(), &db.Provider{})
		require.NoError(t, err)
	})

	t.Run("MarshallConfig", func(t *testing.T) {
		t.Parallel()

		validRaw := json.RawMessage(`{"quay":{"namespace":"redhat"}}`)
		res, err := pcm.MarshallConfig(context.Background(), db.ProviderClassQuay, validRaw)
		require.NoError(t, err)
		assert.NotEmpty(t, res)

		unsupportedRes, err := pcm.MarshallConfig(context.Background(), db.ProviderClassGithub, validRaw)
		require.Error(t, err)
		assert.Nil(t, unsupportedRes)
		assert.Contains(t, err.Error(), "provider does not implement github")
	})
}
