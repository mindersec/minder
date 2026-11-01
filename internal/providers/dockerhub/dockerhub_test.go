// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package dockerhub

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

	cred := credentials.NewOAuth2TokenCredential("test-dockerhub-token")
	cfg := &minderv1.DockerHubProviderConfig{
		Namespace: proto.String("acme-org"),
	}

	lister, err := New(cred, cfg)
	require.NoError(t, err)
	require.NotNil(t, lister)
	assert.Equal(t, "acme-org", lister.namespace)
	assert.Equal(t, "https://hub.docker.com/v2/repositories/acme-org", lister.GetNamespaceURL())
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
				"dockerhub": {
					"namespace": "acme-corp"
				}
			}`,
			expectNilCfg:  false,
			expectedOwner: "acme-corp",
			expectErr:     false,
		},
		{
			name: "missing required namespace fails protobuf validation",
			rawCfg: `{
				"dockerhub": {}
			}`,
			expectNilCfg: true,
			expectErr:    true,
			errContains:  "namespace is required",
		},
		{
			name: "empty namespace string fails protobuf validation",
			rawCfg: `{
				"dockerhub": {
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
			name: "missing dockerhub block fails schema validation",
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
				"dockerhub": {
					"namespace": "acme-org"
				}
			}`,
			expectErr: false,
		},
		{
			name: "validation failure on empty namespace",
			rawCfg: `{
				"dockerhub": {
					"namespace": ""
				}
			}`,
			expectErr:   true,
			errContains: "namespace is required",
		},
		{
			name:        "invalid json syntax",
			rawCfg:      `{"dockerhub": `,
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
			var w dhConfigWrapper
			require.NoError(t, json.Unmarshal(marshaled, &w))
			require.NotNil(t, w.DockerHub)
			assert.Equal(t, "acme-org", w.DockerHub.GetNamespace())
		})
	}
}

func TestCanImplement(t *testing.T) {
	t.Parallel()

	dh := &dockerHubImageLister{}

	assert.True(t, dh.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_IMAGE_LISTER))
	assert.True(t, dh.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_OCI))
	assert.False(t, dh.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_GITHUB))
	assert.False(t, dh.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_GIT))
	assert.False(t, dh.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_REST))
}

func TestListImages(t *testing.T) {
	t.Parallel()

	t.Run("success returns container names", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/v2/repositories/acme-org", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"count": 2,
				"results": [
					{"name": "backend-service"},
					{"name": "worker-queue"}
				]
			}`))
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		targetURL, err := url.Parse(server.URL + "/v2/repositories/acme-org")
		require.NoError(t, err)

		dh := &dockerHubImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := dh.ListImages(context.Background())
		require.NoError(t, err)
		assert.Equal(t, []string{"backend-service", "worker-queue"}, images)
	})

	t.Run("empty repository list returns empty slice", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/v2/repositories/empty-org", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"count": 0, "results": []}`))
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		targetURL, err := url.Parse(server.URL + "/v2/repositories/empty-org")
		require.NoError(t, err)

		dh := &dockerHubImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := dh.ListImages(context.Background())
		require.NoError(t, err)
		assert.Empty(t, images)
	})

	t.Run("401 unauthorized returns formatted error", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/v2/repositories/private-org", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"detail": "Incorrect authentication credentials"}`))
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		targetURL, err := url.Parse(server.URL + "/v2/repositories/private-org")
		require.NoError(t, err)

		dh := &dockerHubImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := dh.ListImages(context.Background())
		require.Error(t, err)
		assert.Nil(t, images)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("404 not found returns not found error", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/v2/repositories/nonexistent-org", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"detail": "Not found"}`))
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		targetURL, err := url.Parse(server.URL + "/v2/repositories/nonexistent-org")
		require.NoError(t, err)

		dh := &dockerHubImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := dh.ListImages(context.Background())
		require.Error(t, err)
		assert.Nil(t, images)
		assert.Equal(t, "not found", err.Error())
	})

	t.Run("unexpected status code returns error", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/v2/repositories/error-org", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"detail": "internal error"}`))
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		targetURL, err := url.Parse(server.URL + "/v2/repositories/error-org")
		require.NoError(t, err)

		dh := &dockerHubImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := dh.ListImages(context.Background())
		require.Error(t, err)
		assert.Nil(t, images)
		assert.Contains(t, err.Error(), "unexpected status code: 500")
	})

	t.Run("malformed json response returns decode error", func(t *testing.T) {
		t.Parallel()

		mux := http.NewServeMux()
		mux.HandleFunc("/v2/repositories/corrupt-org", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{corrupted-json-payload`))
		})

		server := httptest.NewServer(mux)
		defer server.Close()

		targetURL, err := url.Parse(server.URL + "/v2/repositories/corrupt-org")
		require.NoError(t, err)

		dh := &dockerHubImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := dh.ListImages(context.Background())
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

		dh := &dockerHubImageLister{
			cli:    server.Client(),
			target: targetURL,
		}

		images, err := dh.ListImages(ctx)
		require.Error(t, err)
		assert.Nil(t, images)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

func TestProviderLifecycleMethods(t *testing.T) {
	t.Parallel()

	dh := &dockerHubImageLister{namespace: "test-ns"}
	ctx := context.Background()

	t.Run("CreationOptions for supported vs unsupported entity", func(t *testing.T) {
		t.Parallel()

		opts := dh.CreationOptions(minderv1.Entity_ENTITY_ARTIFACTS)
		require.NotNil(t, opts)
		assert.False(t, opts.RegisterWithProvider)
		assert.False(t, opts.PublishReconciliationEvent)

		unsupportedOpts := dh.CreationOptions(minderv1.Entity_ENTITY_REPOSITORIES)
		assert.Nil(t, unsupportedOpts)
	})

	t.Run("RegisterEntity returns props for artifacts and error for others", func(t *testing.T) {
		t.Parallel()

		props := properties.NewProperties(map[string]any{"key": "value"})

		registered, err := dh.RegisterEntity(ctx, minderv1.Entity_ENTITY_ARTIFACTS, props)
		require.NoError(t, err)
		assert.Equal(t, props, registered)

		unsupportedReg, err := dh.RegisterEntity(ctx, minderv1.Entity_ENTITY_REPOSITORIES, props)
		require.ErrorIs(t, err, provifv1.ErrUnsupportedEntity)
		assert.Nil(t, unsupportedReg)
	})

	t.Run("DeregisterEntity returns nil", func(t *testing.T) {
		t.Parallel()

		props := properties.NewProperties(map[string]any{"key": "value"})
		err := dh.DeregisterEntity(ctx, minderv1.Entity_ENTITY_ARTIFACTS, props)
		require.NoError(t, err)
	})

	t.Run("FetchProperty returns nil", func(t *testing.T) {
		t.Parallel()

		props := properties.NewProperties(map[string]any{"key": "value"})
		prop, err := dh.FetchProperty(ctx, props, minderv1.Entity_ENTITY_ARTIFACTS, "some_prop")
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

	pcm := NewDockerHubProviderClassManager(nil, nil)
	require.NotNil(t, pcm)

	t.Run("GetSupportedClasses", func(t *testing.T) {
		t.Parallel()
		classes := pcm.GetSupportedClasses()
		assert.Equal(t, []db.ProviderClass{db.ProviderClassDockerhub}, classes)
	})

	t.Run("GetProviderClassInfo", func(t *testing.T) {
		t.Parallel()

		info, err := pcm.GetProviderClassInfo(db.ProviderClassDockerhub)
		require.NoError(t, err)
		require.NotNil(t, info)
		assert.Equal(t, DockerHub, info.Class)

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

		validRaw := json.RawMessage(`{"dockerhub":{"namespace":"acme"}}`)
		res, err := pcm.MarshallConfig(context.Background(), db.ProviderClassDockerhub, validRaw)
		require.NoError(t, err)
		assert.NotEmpty(t, res)

		unsupportedRes, err := pcm.MarshallConfig(context.Background(), db.ProviderClassGithub, validRaw)
		require.Error(t, err)
		assert.Nil(t, unsupportedRes)
		assert.Contains(t, err.Error(), "provider does not implement github")
	})
}
