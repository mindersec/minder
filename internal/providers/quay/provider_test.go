// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package quay

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/providers/oci"
	"github.com/mindersec/minder/internal/verifier/verifyif"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/entities/properties"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
	testhelper "github.com/mindersec/minder/pkg/providers/v1/testing"
)

func TestRegistration(t *testing.T) {
	t.Parallel()
	// We don't need a full constructor here, so we're naughty
	q := &quayImageLister{}
	testhelper.CheckRegistrationExcept(t, q)
}

func TestClassInfo(t *testing.T) {
	t.Parallel()

	info := ClassInfo()
	require.NotNil(t, info)

	assert.Equal(t, Quay, info.Class)
	assert.Equal(t, "Quay.io", info.DisplayName)
	assert.NotEmpty(t, info.Description)
	assert.Equal(t, providerDocsURL, info.DocumentationUrl)

	assert.ElementsMatch(t, []minderv1.AuthorizationFlow{
		minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT,
	}, info.SupportedAuthFlows)

	assert.Empty(t, info.SupportedEntities)

	assert.ElementsMatch(t, []minderv1.ProviderType{
		minderv1.ProviderType_PROVIDER_TYPE_IMAGE_LISTER,
		minderv1.ProviderType_PROVIDER_TYPE_OCI,
	}, info.SupportedProviderTypes)
}

// newTestRegistry starts an in-process OCI registry and returns its host:port.
func newTestRegistry(t *testing.T) string {
	t.Helper()

	srv := httptest.NewServer(registry.New(registry.Logger(log.New(io.Discard, "", 0))))
	t.Cleanup(srv.Close)

	return strings.TrimPrefix(srv.URL, "http://")
}

// pushRandomImage publishes a throwaway image and returns its digest.
func pushRandomImage(t *testing.T, host, repo, tag string) string {
	t.Helper()

	img, err := random.Image(256, 1)
	require.NoError(t, err)

	ref, err := name.NewTag(fmt.Sprintf("%s/%s:%s", host, repo, tag))
	require.NoError(t, err)
	require.NoError(t, remote.Write(ref, img))

	dig, err := img.Digest()
	require.NoError(t, err)

	return dig.String()
}

func TestFetchAllProperties(t *testing.T) {
	t.Parallel()

	const namespace = "testns"

	host := newTestRegistry(t)
	taggedDigest := pushRandomImage(t, host, namespace+"/myimage", "v1.2")
	latestDigest := pushRandomImage(t, host, namespace+"/myimage", "latest")
	// Tag resolution is only observable if the two tags point at different images.
	require.NotEqual(t, taggedDigest, latestDigest)

	q := &quayImageLister{
		OCI: oci.New(nil, host, path.Join(host, namespace)),
	}

	tests := []struct {
		name       string
		entType    minderv1.Entity
		props      *properties.Properties
		wantName   string
		wantDigest string
		wantErr    string
		wantErrIs  error
	}{
		{
			name:       "explicit tag",
			entType:    minderv1.Entity_ENTITY_ARTIFACTS,
			props:      properties.NewProperties(map[string]any{properties.PropertyName: "myimage:v1.2"}),
			wantName:   "myimage:v1.2",
			wantDigest: taggedDigest,
		},
		{
			name:       "no tag defaults to latest",
			entType:    minderv1.Entity_ENTITY_ARTIFACTS,
			props:      properties.NewProperties(map[string]any{properties.PropertyName: "myimage"}),
			wantName:   "myimage",
			wantDigest: latestDigest,
		},
		{
			name:    "missing name property",
			entType: minderv1.Entity_ENTITY_ARTIFACTS,
			props:   properties.NewProperties(map[string]any{}),
			wantErr: "failed to get artifact name",
		},
		{
			name:    "empty name property",
			entType: minderv1.Entity_ENTITY_ARTIFACTS,
			props:   properties.NewProperties(map[string]any{properties.PropertyName: ""}),
			wantErr: "artifact name is empty",
		},
		{
			name:    "empty tag",
			entType: minderv1.Entity_ENTITY_ARTIFACTS,
			props:   properties.NewProperties(map[string]any{properties.PropertyName: "myimage:"}),
			wantErr: "empty tag",
		},
		{
			name:    "unknown image",
			entType: minderv1.Entity_ENTITY_ARTIFACTS,
			props:   properties.NewProperties(map[string]any{properties.PropertyName: "nosuchimage:v1"}),
			wantErr: `failed to resolve digest for "nosuchimage:v1"`,
		},
		{
			name:      "unsupported entity type",
			entType:   minderv1.Entity_ENTITY_REPOSITORIES,
			props:     properties.NewProperties(map[string]any{properties.PropertyName: "myimage"}),
			wantErrIs: provifv1.ErrUnsupportedEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := q.FetchAllProperties(context.Background(), tt.props, tt.entType, nil)

			if tt.wantErrIs != nil {
				assert.ErrorIs(t, err, tt.wantErrIs)
				assert.Nil(t, got)
				return
			}
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			assert.Equal(t, tt.wantName, got.GetProperty(properties.PropertyName).GetString())
			assert.Equal(t, tt.wantDigest, got.GetProperty(properties.PropertyUpstreamID).GetString())
			assert.Equal(t, string(verifyif.ArtifactTypeContainer),
				got.GetProperty(properties.ArtifactPropertyType).GetString())
		})
	}
}

func TestGetEntityName(t *testing.T) {
	t.Parallel()

	q := &quayImageLister{}

	tests := []struct {
		name    string
		entType minderv1.Entity
		props   *properties.Properties
		want    string
		wantErr string
	}{
		{
			name:    "name is returned unchanged",
			entType: minderv1.Entity_ENTITY_ARTIFACTS,
			props:   properties.NewProperties(map[string]any{properties.PropertyName: "myimage:v1.2"}),
			want:    "myimage:v1.2",
		},
		{
			name:    "missing name property",
			entType: minderv1.Entity_ENTITY_ARTIFACTS,
			props:   properties.NewProperties(map[string]any{}),
			wantErr: "failed to get artifact name",
		},
		{
			name:    "unsupported entity type",
			entType: minderv1.Entity_ENTITY_REPOSITORIES,
			props:   properties.NewProperties(map[string]any{properties.PropertyName: "myimage"}),
			wantErr: "not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := q.GetEntityName(tt.entType, tt.props)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
