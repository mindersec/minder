// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// MockCredential implements provifv1.Credential and provifv1.OAuth2TokenCredential
type MockCredential struct {
	token string
}

type mockTokenSource struct {
	token string
}

func (m mockTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: m.token}, nil
}

func (m MockCredential) GetAsOAuth2TokenSource() oauth2.TokenSource {
	return mockTokenSource{token: m.token}
}

func TestResolveCreatedAt(t *testing.T) {
	t.Parallel()

	buildTime := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
	// poison is returned by the config getter for cases where the annotation is
	// present: if the implementation wrongly consulted the config, the assertion
	// against buildTime (or the expected error) would fail.
	poison := time.Date(1999, time.December, 31, 23, 59, 59, 0, time.UTC)
	epoch := time.Unix(0, 0).UTC()

	configReturning := func(created time.Time) func() (*v1.ConfigFile, error) {
		return func() (*v1.ConfigFile, error) {
			return &v1.ConfigFile{Created: v1.Time{Time: created}}, nil
		}
	}
	configFailing := func() (*v1.ConfigFile, error) {
		return nil, errors.New("config blob unavailable")
	}

	tests := []struct {
		name       string
		man        *v1.Manifest
		configFile func() (*v1.ConfigFile, error)
		want       time.Time
		wantErr    bool
	}{
		{
			name: "annotation present is preferred over config",
			man: &v1.Manifest{Annotations: map[string]string{
				imgspecv1.AnnotationCreated: buildTime.Format(time.RFC3339),
			}},
			configFile: configReturning(poison),
			want:       buildTime,
		},
		{
			name: "invalid annotation returns error and does not use config",
			man: &v1.Manifest{Annotations: map[string]string{
				imgspecv1.AnnotationCreated: "not-a-timestamp",
			}},
			configFile: configReturning(poison),
			wantErr:    true,
		},
		{
			name:       "missing annotation falls back to config created",
			man:        &v1.Manifest{},
			configFile: configReturning(buildTime),
			want:       buildTime,
		},
		{
			name:       "epoch config timestamp is preserved",
			man:        &v1.Manifest{Annotations: map[string]string{}},
			configFile: configReturning(epoch),
			want:       epoch,
		},
		{
			name:       "zero config timestamp is preserved",
			man:        &v1.Manifest{},
			configFile: configReturning(time.Time{}),
			want:       time.Time{},
		},
		{
			name:       "config fetch error is propagated",
			man:        &v1.Manifest{},
			configFile: configFailing,
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveCreatedAt(tc.man, tc.configFile)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Truef(t, got.Equal(tc.want), "got %s, want %s", got, tc.want)
		})
	}
}

func TestOCI_Basic(t *testing.T) {
	t.Parallel()

	o := New(nil, "invalid.registry.local", "invalid.registry.local/myrepo")
	assert.True(t, o.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_OCI))
	assert.False(t, o.CanImplement(minderv1.ProviderType_PROVIDER_TYPE_GITHUB))
	assert.Equal(t, "invalid.registry.local", o.GetRegistry())
}

func TestOCI_WithRegistry(t *testing.T) {
	t.Parallel()

	s := httptest.NewServer(registry.New())
	t.Cleanup(s.Close)
	host := strings.TrimPrefix(s.URL, "http://")

	// Push a random image to the test registry so we have something to read back.
	img, err := random.Image(1024, 1)
	require.NoError(t, err)

	ref, err := name.NewTag(host + "/myimage:v1.0")
	require.NoError(t, err)
	require.NoError(t, remote.Write(ref, img))

	o := New(nil, host, host)
	ctx := context.Background()

	tags, err := o.ListTags(ctx, "myimage")
	require.NoError(t, err)
	assert.Contains(t, tags, "v1.0")

	digest, err := o.GetDigest(ctx, "myimage", "v1.0")
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(digest, "sha256:"), "expected sha256 digest, got %s", digest)

	manifest, err := o.GetManifest(ctx, "myimage", "v1.0")
	require.NoError(t, err)
	assert.NotNil(t, manifest)
}

func TestOCI_Auth(t *testing.T) {
	t.Parallel()

	t.Run("anonymous auth", func(t *testing.T) {
		o := New(nil, "invalid.registry.local", "invalid.registry.local/myrepo")
		auth, err := o.GetAuthenticator()
		require.NoError(t, err)
		assert.NotNil(t, auth)
	})

	t.Run("valid oauth2 auth", func(t *testing.T) {
		cred := MockCredential{token: "secret-token"}
		o := New(cred, "registry.com", "registry.com/myrepo")
		auth, err := o.GetAuthenticator()
		require.NoError(t, err)
		assert.NotNil(t, auth)
	})

	t.Run("invalid credential type", func(t *testing.T) {
		o := New("not-an-oauth-cred", "registry.com", "registry.com/myrepo")
		_, err := o.GetAuthenticator()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "credential is not an OAuth2 token credential")
	})
}
