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

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/internal/providers/credentials"
	"github.com/mindersec/minder/internal/providers/oci"
	"github.com/mindersec/minder/internal/verifier/sigstore/container"
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

	return pushImage(t, img, host, repo, tag)
}

func pushImage(t *testing.T, img v1.Image, host, repo, tag string) string {
	t.Helper()

	ref, err := name.NewTag(fmt.Sprintf("%s/%s:%s", host, repo, tag))
	require.NoError(t, err)
	require.NoError(t, remote.Write(ref, img))

	dig, err := img.Digest()
	require.NoError(t, err)

	return dig.String()
}

func pushMultiArchIndex(t *testing.T, host, repo, tag string) string {
	t.Helper()

	var adds []mutate.IndexAddendum
	for _, arch := range []string{"amd64", "arm64"} {
		img, err := random.Image(256, 1)
		require.NoError(t, err)
		adds = append(adds, mutate.IndexAddendum{
			Add:        img,
			Descriptor: v1.Descriptor{Platform: &v1.Platform{OS: "linux", Architecture: arch}},
		})
	}
	idx := mutate.AppendManifests(empty.Index, adds...)

	ref, err := name.NewTag(fmt.Sprintf("%s/%s:%s", host, repo, tag))
	require.NoError(t, err)
	require.NoError(t, remote.WriteIndex(ref, idx))

	dig, err := idx.Digest()
	require.NoError(t, err)

	return dig.String()
}

func TestFetchAllProperties(t *testing.T) {
	t.Parallel()

	const namespace = "testns"

	host := newTestRegistry(t)
	taggedDigest := pushRandomImage(t, host, namespace+"/myimage", "v1.2")
	latestDigest := pushRandomImage(t, host, namespace+"/myimage", "latest")
	multiArchDigest := pushMultiArchIndex(t, host, namespace+"/myimage", "multi")
	// Tag resolution is only observable if the two tags point at different images.
	require.NotEqual(t, taggedDigest, latestDigest)

	q := &quayImageLister{
		OCI: oci.New(nil, host, path.Join(host, namespace)),
	}

	tests := []struct {
		name           string
		entType        minderv1.Entity
		props          *properties.Properties
		wantName       string
		wantUpstreamID string
		wantErr        string
		wantErrIs      error
	}{
		{
			name:           "explicit tag",
			entType:        minderv1.Entity_ENTITY_ARTIFACTS,
			props:          properties.NewProperties(map[string]any{properties.PropertyName: "myimage:v1.2"}),
			wantName:       "myimage:v1.2",
			wantUpstreamID: "myimage@" + taggedDigest,
		},
		{
			name:           "no tag defaults to latest",
			entType:        minderv1.Entity_ENTITY_ARTIFACTS,
			props:          properties.NewProperties(map[string]any{properties.PropertyName: "myimage"}),
			wantName:       "myimage",
			wantUpstreamID: "myimage@" + latestDigest,
		},
		{
			name:           "multi-arch tag uses index digest",
			entType:        minderv1.Entity_ENTITY_ARTIFACTS,
			props:          properties.NewProperties(map[string]any{properties.PropertyName: "myimage:multi"}),
			wantName:       "myimage:multi",
			wantUpstreamID: "myimage@" + multiArchDigest,
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
			assert.Equal(t, tt.wantUpstreamID, got.GetProperty(properties.PropertyUpstreamID).GetString())
			assert.Equal(t, string(verifyif.ArtifactTypeContainer),
				got.GetProperty(properties.ArtifactPropertyType).GetString())
		})
	}
}

func TestFetchAllPropertiesMirroredDigest(t *testing.T) {
	t.Parallel()

	const namespace = "testns"

	host := newTestRegistry(t)
	img, err := random.Image(256, 1)
	require.NoError(t, err)
	origDigest := pushImage(t, img, host, namespace+"/orig", "v1")
	mirrorDigest := pushImage(t, img, host, namespace+"/mirror", "v1")
	require.Equal(t, origDigest, mirrorDigest)

	q := &quayImageLister{
		OCI: oci.New(nil, host, path.Join(host, namespace)),
	}

	orig, err := q.FetchAllProperties(context.Background(),
		properties.NewProperties(map[string]any{properties.PropertyName: "orig:v1"}),
		minderv1.Entity_ENTITY_ARTIFACTS, nil)
	require.NoError(t, err)
	mirror, err := q.FetchAllProperties(context.Background(),
		properties.NewProperties(map[string]any{properties.PropertyName: "mirror:v1"}),
		minderv1.Entity_ENTITY_ARTIFACTS, nil)
	require.NoError(t, err)

	origID := orig.GetProperty(properties.PropertyUpstreamID).GetString()
	mirrorID := mirror.GetProperty(properties.PropertyUpstreamID).GetString()
	assert.Equal(t, "orig@"+origDigest, origID)
	assert.Equal(t, "mirror@"+mirrorDigest, mirrorID)
	assert.NotEqual(t, origID, mirrorID)
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

func TestPropertiesToProtoMessage(t *testing.T) {
	t.Parallel()

	q := &quayImageLister{namespace: "testns"}

	tests := []struct {
		name      string
		entType   minderv1.Entity
		props     *properties.Properties
		wantName  string
		wantErrIs error
	}{
		{
			name:     "tagged name",
			entType:  minderv1.Entity_ENTITY_ARTIFACTS,
			props:    oci.NewArtifactProperties("myimage:v1.2", "myimage@sha256:abc"),
			wantName: "myimage",
		},
		{
			name:     "untagged name",
			entType:  minderv1.Entity_ENTITY_ARTIFACTS,
			props:    oci.NewArtifactProperties("myimage", "myimage@sha256:abc"),
			wantName: "myimage",
		},
		{
			name:      "unsupported entity type",
			entType:   minderv1.Entity_ENTITY_REPOSITORIES,
			props:     oci.NewArtifactProperties("myimage", "myimage@sha256:abc"),
			wantErrIs: provifv1.ErrUnsupportedEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := q.PropertiesToProtoMessage(tt.entType, tt.props)
			if tt.wantErrIs != nil {
				require.ErrorIs(t, err, tt.wantErrIs)
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			art, ok := got.(*minderv1.Artifact)
			require.True(t, ok, "expected *minderv1.Artifact, got %T", got)
			assert.Equal(t, "myimage@sha256:abc", art.GetArtifactPk())
			assert.Equal(t, "testns", art.GetOwner())
			assert.Equal(t, tt.wantName, art.GetName())
			assert.Equal(t, "container", art.GetType())
		})
	}
}

// TestConvertedArtifactImageRef registers an artifact against a local registry,
// converts it the way the ingester receives it, and checks the image ref the
// sigstore verifier builds from it.
func TestConvertedArtifactImageRef(t *testing.T) {
	t.Parallel()

	const namespace = "testns"

	host := newTestRegistry(t)
	digest := pushRandomImage(t, host, namespace+"/myimage", "v1.2")

	local := &quayImageLister{
		OCI:       oci.New(nil, host, path.Join(host, namespace)),
		namespace: namespace,
	}
	props, err := local.FetchAllProperties(context.Background(),
		properties.NewProperties(map[string]any{properties.PropertyName: "myimage:v1.2"}),
		minderv1.Entity_ENTITY_ARTIFACTS, nil)
	require.NoError(t, err)

	prod, err := New(credentials.NewOAuth2TokenCredential("token"),
		&minderv1.QuayProviderConfig{Namespace: proto.String(namespace)})
	require.NoError(t, err)

	msg, err := prod.PropertiesToProtoMessage(minderv1.Entity_ENTITY_ARTIFACTS, props)
	require.NoError(t, err)
	art, ok := msg.(*minderv1.Artifact)
	require.True(t, ok, "expected *minderv1.Artifact, got %T", msg)

	assert.Equal(t, "quay.io/testns/myimage@"+digest,
		container.BuildImageRef(prod.GetRegistry(), art.GetOwner(), art.GetName(), digest))

	// Against the local registry the same ref resolves, so an unsigned image
	// is reported as unsigned rather than failing the lookup.
	res, err := container.Verify(context.Background(), nil, art.GetOwner(), art.GetName(), digest,
		container.WithRegistry(local.GetRegistry()), container.WithAuthenticator(authn.Anonymous))
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.False(t, res[0].IsSigned)
}
