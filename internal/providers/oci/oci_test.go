// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

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

func TestGetDigest(t *testing.T) {
	t.Parallel()

	const namespace = "testns"

	srv := httptest.NewServer(registry.New(registry.Logger(log.New(io.Discard, "", 0))))
	t.Cleanup(srv.Close)
	host := strings.TrimPrefix(srv.URL, "http://")

	img, err := random.Image(256, 1)
	require.NoError(t, err)
	imgRef, err := name.NewTag(fmt.Sprintf("%s/%s/myimage:single", host, namespace))
	require.NoError(t, err)
	require.NoError(t, remote.Write(imgRef, img))
	imgDigest, err := img.Digest()
	require.NoError(t, err)
	imgMediaType, err := img.MediaType()
	require.NoError(t, err)

	amd64, err := random.Image(256, 1)
	require.NoError(t, err)
	arm64, err := random.Image(256, 1)
	require.NoError(t, err)
	idx := mutate.AppendManifests(empty.Index,
		mutate.IndexAddendum{
			Add:        amd64,
			Descriptor: v1.Descriptor{Platform: &v1.Platform{OS: "linux", Architecture: "amd64"}},
		},
		mutate.IndexAddendum{
			Add:        arm64,
			Descriptor: v1.Descriptor{Platform: &v1.Platform{OS: "linux", Architecture: "arm64"}},
		},
	)
	idxRef, err := name.NewTag(fmt.Sprintf("%s/%s/myimage:multi", host, namespace))
	require.NoError(t, err)
	require.NoError(t, remote.WriteIndex(idxRef, idx))
	idxDigest, err := idx.Digest()
	require.NoError(t, err)
	amd64Digest, err := amd64.Digest()
	require.NoError(t, err)

	o := New(nil, host, path.Join(host, namespace))

	tests := []struct {
		name          string
		tag           string
		wantDigest    string
		wantMediaType types.MediaType
		notDigest     string
	}{
		{
			name:          "single image",
			tag:           "single",
			wantDigest:    imgDigest.String(),
			wantMediaType: imgMediaType,
		},
		{
			name:          "multi-arch index",
			tag:           "multi",
			wantDigest:    idxDigest.String(),
			wantMediaType: types.OCIImageIndex,
			notDigest:     amd64Digest.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dig, err := o.GetDigest(context.Background(), "myimage", tt.tag)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDigest, dig)
			if tt.notDigest != "" {
				assert.NotEqual(t, tt.notDigest, dig)
			}

			raw, err := o.GetRawManifest(context.Background(), "myimage", dig)
			require.NoError(t, err)
			assert.Equal(t, tt.wantMediaType, raw.MediaType)
			assert.Equal(t, dig, raw.Digest.String())
		})
	}
}

// newRegistryWithManifestOverride starts a test registry pre-loaded with v1
// and v2 tags and returns a server that intercepts GET .../manifests/v2,
// responding with the given HTTP status code instead of serving the real
// manifest. Tag listing is passed through unchanged, so v2 always appears
// in the list and getImage is actually reached for that tag.
func newRegistryWithManifestOverride(t *testing.T, namespace, overrideTag string, overrideStatus int) (host string) {
	t.Helper()
	reg := registry.New(registry.Logger(log.New(io.Discard, "", 0)))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/manifests/"+overrideTag) {
			w.WriteHeader(overrideStatus)
			return
		}
		reg.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	host = strings.TrimPrefix(srv.URL, "http://")

	for _, tag := range []string{"v1", "v2"} {
		img, err := random.Image(256, 1)
		require.NoError(t, err)
		ref, err := name.NewTag(fmt.Sprintf("%s/%s/myimage:%s", host, namespace, tag))
		require.NoError(t, err)
		require.NoError(t, remote.Write(ref, img))
	}
	return host
}

func TestGetArtifactVersions_SkipsDeletedTag(t *testing.T) {
	t.Parallel()

	const namespace = "testns"
	// 404 for v2 manifests: tag still listed but getImage returns not-found.
	host := newRegistryWithManifestOverride(t, namespace, "v2", http.StatusNotFound)

	o := New(nil, host, path.Join(host, namespace))
	artifact := &minderv1.Artifact{Name: "myimage"}

	versions, err := o.GetArtifactVersions(context.Background(), artifact, &alwaysIncludeFilter{})
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, []string{"v1"}, versions[0].Tags)
}

func TestGetArtifactVersions_ReturnsErrorOnNonNotFound(t *testing.T) {
	t.Parallel()

	const namespace = "testns"
	// 429 for v2 manifests: rate-limit must propagate as an error, not be skipped.
	host := newRegistryWithManifestOverride(t, namespace, "v2", http.StatusTooManyRequests)

	o := New(nil, host, path.Join(host, namespace))
	artifact := &minderv1.Artifact{Name: "myimage"}

	_, err := o.GetArtifactVersions(context.Background(), artifact, &alwaysIncludeFilter{})
	require.Error(t, err)
}

// alwaysIncludeFilter is a GetArtifactVersionsFilter that never skips anything.
type alwaysIncludeFilter struct{}

func (*alwaysIncludeFilter) IsSkippable(_ time.Time, _ []string) error { return nil }

func TestGetArtifactVersions_IndexWithNoAMD64Variant(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(registry.New(registry.Logger(log.New(io.Discard, "", 0))))
	t.Cleanup(srv.Close)
	host := strings.TrimPrefix(srv.URL, "http://")

	arm64img, err := random.Image(256, 1)
	require.NoError(t, err)

	idx := mutate.AppendManifests(
		mutate.Annotations(empty.Index, map[string]string{
			imgspecv1.AnnotationCreated: time.Now().UTC().Format(time.RFC3339),
		}).(v1.ImageIndex),
		mutate.IndexAddendum{
			Add: arm64img,
			Descriptor: v1.Descriptor{
				Platform: &v1.Platform{OS: "linux", Architecture: "arm64"},
			},
		},
	)

	ref, err := name.NewTag(fmt.Sprintf("%s/testns/myimage:latest", host))
	require.NoError(t, err)
	require.NoError(t, remote.WriteIndex(ref, idx))

	// Get the top-level index digest to assert against
	desc, err := remote.Get(ref)
	require.NoError(t, err)
	wantDigest := desc.Digest.String()

	o := New(nil, host, path.Join(host, "testns"))
	artifact := &minderv1.Artifact{Name: "myimage"}

	versions, err := o.GetArtifactVersions(context.Background(), artifact, &alwaysIncludeFilter{})
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, wantDigest, versions[0].Sha)
}

func TestGetArtifactVersions_IndexWithoutAnnotationUsesFirstChild(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(registry.New(registry.Logger(log.New(io.Discard, "", 0))))
	t.Cleanup(srv.Close)
	host := strings.TrimPrefix(srv.URL, "http://")

	// Build arm64-only index with NO annotation, exercises firstChildCreatedAt
	arm64img, err := random.Image(256, 1)
	require.NoError(t, err)

	// Set a known creation time in the image config
	expectedTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	arm64img, err = mutate.ConfigFile(arm64img, &v1.ConfigFile{
		Created:      v1.Time{Time: expectedTime},
		OS:           "linux",
		Architecture: "arm64",
	})
	require.NoError(t, err)

	// No annotations on the index, forces firstChildCreatedAt fallback
	idx := mutate.AppendManifests(
		empty.Index,
		mutate.IndexAddendum{
			Add: arm64img,
			Descriptor: v1.Descriptor{
				Platform: &v1.Platform{OS: "linux", Architecture: "arm64"},
			},
		},
	)

	ref, err := name.NewTag(fmt.Sprintf("%s/testns/myimage:latest", host))
	require.NoError(t, err)
	require.NoError(t, remote.WriteIndex(ref, idx))

	o := New(nil, host, path.Join(host, "testns"))
	artifact := &minderv1.Artifact{Name: "myimage"}

	versions, err := o.GetArtifactVersions(context.Background(), artifact, &alwaysIncludeFilter{})
	require.NoError(t, err)
	require.Len(t, versions, 1)
	assert.Equal(t, expectedTime, versions[0].CreatedAt.AsTime())
}

func TestGetArtifactVersions_IndexUsesFirstValidChildTimestamp(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(registry.New(registry.Logger(log.New(io.Discard, "", 0))))
	t.Cleanup(srv.Close)
	host := strings.TrimPrefix(srv.URL, "http://")

	olderTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	newerTime := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	arm64img, err := random.Image(256, 1)
	require.NoError(t, err)
	arm64img, err = mutate.ConfigFile(arm64img, &v1.ConfigFile{
		Created:      v1.Time{Time: olderTime},
		OS:           "linux",
		Architecture: "arm64",
	})
	require.NoError(t, err)

	arm32img, err := random.Image(256, 1)
	require.NoError(t, err)
	arm32img, err = mutate.ConfigFile(arm32img, &v1.ConfigFile{
		Created:      v1.Time{Time: newerTime},
		OS:           "linux",
		Architecture: "arm",
	})
	require.NoError(t, err)

	// No index-level annotation, forces firstChildCreatedAt
	idx := mutate.AppendManifests(
		empty.Index,
		mutate.IndexAddendum{
			Add: arm64img,
			Descriptor: v1.Descriptor{
				Platform: &v1.Platform{OS: "linux", Architecture: "arm64"},
			},
		},
		mutate.IndexAddendum{
			Add: arm32img,
			Descriptor: v1.Descriptor{
				Platform: &v1.Platform{OS: "linux", Architecture: "arm"},
			},
		},
	)

	ref, err := name.NewTag(fmt.Sprintf("%s/testns/myimage:latest", host))
	require.NoError(t, err)
	require.NoError(t, remote.WriteIndex(ref, idx))

	o := New(nil, host, path.Join(host, "testns"))
	artifact := &minderv1.Artifact{Name: "myimage"}

	versions, err := o.GetArtifactVersions(context.Background(), artifact, &alwaysIncludeFilter{})
	require.NoError(t, err)
	require.Len(t, versions, 1)
	// Returns the first child with a valid timestamp (arm64, olderTime)
	// All platforms in a CI run have close timestamps; first valid is enough, see issue:#6975
	assert.Equal(t, olderTime, versions[0].CreatedAt.AsTime())
}
