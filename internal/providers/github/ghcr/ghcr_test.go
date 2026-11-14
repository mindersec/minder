// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ghcr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v63/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/proto"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

type mockOAuth2Credential struct {
	token *oauth2.Token
}

func (m *mockOAuth2Credential) GetAsOAuth2TokenSource() oauth2.TokenSource {
	return oauth2.StaticTokenSource(m.token)
}

func (*mockOAuth2Credential) SetToken(_ *oauth2.Token) {}

func TestNew(t *testing.T) {
	t.Parallel()

	cred := &mockOAuth2Credential{
		token: &oauth2.Token{AccessToken: "fake-gh-token"},
	}
	cfg := &minderv1.GHCRProviderConfig{
		Namespace: proto.String("test-owner"),
	}

	lister := New(cred, cfg)
	require.NotNil(t, lister)
	assert.NotNil(t, lister.client)
	assert.Equal(t, "test-owner", lister.Namespace())
	assert.Equal(t, "ghcr.io/test-owner", lister.GetNamespaceURL())
}

func TestParseV1Config(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		rawCfg        string
		expectNilCfg  bool
		expectedOwner string
		expectErr     bool
	}{
		{
			name: "valid configuration with namespace",
			rawCfg: `{
				"ghcr": {
					"namespace": "acme-corp"
				}
			}`,
			expectNilCfg:  false,
			expectedOwner: "acme-corp",
			expectErr:     false,
		},
		{
			name: "valid configuration with empty namespace",
			rawCfg: `{
				"ghcr": {}
			}`,
			expectNilCfg:  false,
			expectedOwner: "",
			expectErr:     false,
		},
		{
			name:          "invalid json payload",
			rawCfg:        `{invalid-json: `,
			expectNilCfg:  true,
			expectedOwner: "",
			expectErr:     true,
		},
		{
			name: "missing ghcr block",
			rawCfg: `{
				"other_provider": {
					"namespace": "test"
				}
			}`,
			expectNilCfg:  true,
			expectedOwner: "",
			expectErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := ParseV1Config(json.RawMessage(tc.rawCfg))
			if tc.expectErr {
				require.Error(t, err)
				assert.Nil(t, cfg)
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

func TestCanImplement(t *testing.T) {
	t.Parallel()

	il := &ImageLister{}

	assert.True(t, il.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_IMAGE_LISTER))
	assert.False(t, il.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_GITHUB))
	assert.False(t, il.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_GIT))
	assert.False(t, il.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_REST))
}

func TestFromGitHubClient(t *testing.T) {
	t.Parallel()

	ghClient := github.NewClient(nil)
	il := FromGitHubClient(ghClient, "octocat")
	require.NotNil(t, il)
	assert.Equal(t, ghClient, il.client)
	assert.Equal(t, "octocat", il.Namespace())
	assert.Equal(t, "ghcr.io/octocat", il.GetNamespaceURL())
}

func TestListImages_Success(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/users/octocat/packages", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "container", r.URL.Query().Get("package_type"))

		packages := []*github.Package{
			{Name: proto.String("minder-server")},
			{Name: proto.String("minder-cli")},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(packages)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := github.NewClient(nil)
	serverURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client.BaseURL = serverURL

	il := FromGitHubClient(client, "octocat")
	images, err := il.ListImages(context.Background())
	require.NoError(t, err)
	require.Len(t, images, 2)
	assert.Equal(t, []string{"minder-server", "minder-cli"}, images)
}

func TestListImages_Pagination(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	var serverURLStr string

	mux.HandleFunc("/users/org-owner/packages", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")

		if page == "0" || page == "" || page == "1" {
			linkHeader := fmt.Sprintf(`<%s/users/org-owner/packages?package_type=container&page=2>; rel="next"`, serverURLStr)
			w.Header().Set("Link", linkHeader)

			packages := []*github.Package{
				{Name: proto.String("image-one")},
				{Name: proto.String("image-two")},
			}
			_ = json.NewEncoder(w).Encode(packages)
			return
		}

		if page == "2" {
			packages := []*github.Package{
				{Name: proto.String("image-three")},
			}
			_ = json.NewEncoder(w).Encode(packages)
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]*github.Package{})
	})

	server := httptest.NewServer(mux)
	defer server.Close()
	serverURLStr = server.URL

	client := github.NewClient(nil)
	parsedURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client.BaseURL = parsedURL

	il := FromGitHubClient(client, "org-owner")
	images, err := il.ListImages(context.Background())
	require.NoError(t, err)
	require.Len(t, images, 3)
	assert.Equal(t, []string{"image-one", "image-two", "image-three"}, images)
}

func TestListImages_NotFound(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/users/missing-ns/packages", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := github.NewClient(nil)
	serverURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client.BaseURL = serverURL

	il := FromGitHubClient(client, "missing-ns")
	images, err := il.ListImages(context.Background())
	require.Error(t, err)
	assert.Empty(t, images)
	assert.Contains(t, err.Error(), "packages not found for namespace missing-ns")
	assert.Contains(t, err.Error(), "not found")
}

func TestListImages_InternalServerError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/users/error-ns/packages", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"Internal Server Error"}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := github.NewClient(nil)
	serverURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client.BaseURL = serverURL

	il := FromGitHubClient(client, "error-ns")
	images, err := il.ListImages(context.Background())
	require.Error(t, err)
	assert.Empty(t, images)
	assert.Contains(t, err.Error(), "500 Internal Server Error")
}

func TestListImages_NoResponseOnError(t *testing.T) {
	t.Parallel()

	// A canceled context makes go-github return a nil response with the error.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	il := FromGitHubClient(github.NewClient(nil), "test-namespace")
	images, err := il.ListImages(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, images)
}

func TestProviderInterfaceMethods(t *testing.T) {
	t.Parallel()

	il := FromGitHubClient(github.NewClient(nil), "test-user")
	ctx := context.Background()

	t.Run("SupportsEntity", func(t *testing.T) {
		t.Parallel()
		assert.False(t, il.SupportsEntity(minderv1.Entity_ENTITY_ARTIFACTS))
		assert.False(t, il.SupportsEntity(minderv1.Entity_ENTITY_REPOSITORIES))
	})

	t.Run("ProviderClassInfo and CreationOptions", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, il.ProviderClassInfo())
		assert.Nil(t, il.CreationOptions(minderv1.Entity_ENTITY_ARTIFACTS))
	})

	t.Run("RegisterEntity returns ErrUnsupportedEntity", func(t *testing.T) {
		t.Parallel()
		props := properties.NewProperties(map[string]any{})
		res, err := il.RegisterEntity(ctx, minderv1.Entity_ENTITY_ARTIFACTS, props)
		require.ErrorIs(t, err, provifv1.ErrUnsupportedEntity)
		assert.Nil(t, res)
	})

	t.Run("DeregisterEntity", func(t *testing.T) {
		t.Parallel()
		err := il.DeregisterEntity(ctx, minderv1.Entity_ENTITY_ARTIFACTS, properties.NewProperties(map[string]any{}))
		require.NoError(t, err)
	})

	t.Run("FetchAllProperties and FetchProperty", func(t *testing.T) {
		t.Parallel()
		props := properties.NewProperties(map[string]any{})
		p, err := il.FetchAllProperties(ctx, props, minderv1.Entity_ENTITY_ARTIFACTS, props)
		require.NoError(t, err)
		assert.Nil(t, p)

		singleProp, err := il.FetchProperty(ctx, props, minderv1.Entity_ENTITY_ARTIFACTS, "name")
		require.NoError(t, err)
		assert.Nil(t, singleProp)
	})

	t.Run("GetEntityName and PropertiesToProtoMessage", func(t *testing.T) {
		t.Parallel()
		props := properties.NewProperties(map[string]any{})
		name, err := il.GetEntityName(minderv1.Entity_ENTITY_ARTIFACTS, props)
		require.NoError(t, err)
		assert.Empty(t, name)

		protoMsg, err := il.PropertiesToProtoMessage(minderv1.Entity_ENTITY_ARTIFACTS, props)
		require.NoError(t, err)
		assert.Nil(t, protoMsg)
	})
}
