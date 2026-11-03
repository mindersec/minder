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
	"sync/atomic"
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
	"google.golang.org/protobuf/types/known/timestamppb"

	artif "github.com/mindersec/minder/internal/providers/artifact"
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

func TestGetArtifactVersions(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(registry.New(registry.Logger(log.New(io.Discard, "", 0))))
	t.Cleanup(srv.Close)
	host := strings.TrimPrefix(srv.URL, "http://")
	created := time.Now().UTC().Truncate(time.Second)
	imageAt := func(at time.Time) v1.Image {
		t.Helper()
		img, err := mutate.ConfigFile(empty.Image, &v1.ConfigFile{
			Created: v1.Time{Time: at}, OS: "linux", Architecture: "amd64",
		})
		require.NoError(t, err)
		return img
	}
	img := imageAt(created)
	other := imageAt(created.Add(-time.Hour))
	old := imageAt(created.AddDate(-1, 0, 0))
	idx := mutate.AppendManifests(empty.Index, mutate.IndexAddendum{
		Add:        img,
		Descriptor: v1.Descriptor{Platform: &v1.Platform{OS: "linux", Architecture: "amd64"}},
	})

	for _, entry := range []struct {
		img  v1.Image
		tags []string
	}{
		{img, []string{"latest", "stable", "sha256-test.sig"}},
		{other, []string{"beta"}},
		{old, []string{"old", "old-stable"}},
	} {
		for _, tag := range entry.tags {
			ref, err := name.NewTag(host + "/myimage:" + tag)
			require.NoError(t, err)
			require.NoError(t, remote.Write(ref, entry.img))
		}
	}
	for _, tag := range []string{"multi-latest", "multi-stable"} {
		ref, err := name.NewTag(host + "/myimage:" + tag)
		require.NoError(t, err)
		require.NoError(t, remote.WriteIndex(ref, idx))
	}
	imgDigest, err := img.Digest()
	require.NoError(t, err)
	otherDigest, err := other.Digest()
	require.NoError(t, err)
	idxDigest, err := idx.Digest()
	require.NoError(t, err)
	require.NotEqual(t, imgDigest, idxDigest)

	imageVersion := &minderv1.ArtifactVersion{
		Sha: imgDigest.String(), Tags: []string{"latest", "stable"}, CreatedAt: timestamppb.New(created),
	}
	otherVersion := &minderv1.ArtifactVersion{
		Sha: otherDigest.String(), Tags: []string{"beta"}, CreatedAt: timestamppb.New(created.Add(-time.Hour)),
	}
	indexVersion := &minderv1.ArtifactVersion{
		Sha: idxDigest.String(), Tags: []string{"multi-latest", "multi-stable"}, CreatedAt: timestamppb.New(created),
	}
	o := New(nil, host, host)

	for _, tt := range []struct {
		name  string
		tags  []string
		regex string
		want  []*minderv1.ArtifactVersion
	}{
		{name: "all tags on one digest", tags: []string{"latest", "stable"}, want: []*minderv1.ArtifactVersion{imageVersion}},
		{name: "tags on different digests do not match", tags: []string{"latest", "beta"}},
		{name: "single tag returns all version tags", tags: []string{"stable"}, want: []*minderv1.ArtifactVersion{imageVersion}},
		{name: "regex matches any version tag", regex: "^stable$", want: []*minderv1.ArtifactVersion{imageVersion}},
		{name: "no filter returns unique versions in tag order", want: []*minderv1.ArtifactVersion{otherVersion, imageVersion, indexVersion}},
		{name: "old images remain excluded", tags: []string{"old", "old-stable"}},
		{name: "digest tags remain excluded", tags: []string{"sha256-test.sig"}},
		{name: "index retains top-level digest", tags: []string{"multi-latest", "multi-stable"}, want: []*minderv1.ArtifactVersion{indexVersion}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			filter, err := artif.BuildFilter(tt.tags, tt.regex)
			require.NoError(t, err)
			got, err := o.GetArtifactVersions(context.Background(), &minderv1.Artifact{Name: "myimage"}, filter)
			require.NoError(t, err)
			require.Len(t, got, len(tt.want))
			for i, want := range tt.want {
				assert.Equal(t, want, got[i])
			}
		})
	}
}

func TestGetArtifactVersions_GroupEdges(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name      string
		alias     string
		failOnGet int32
		status    int
		wantTags  []string
		wantErr   bool
	}{
		{name: "signature alias excludes its whole group", alias: "v2.sig"},
		{name: "deleted alias preserves remaining group", alias: "v2", failOnGet: 1,
			status: http.StatusNotFound, wantTags: []string{"v1", "v3"}},
		{name: "digest not found propagates", alias: "v2", failOnGet: 2,
			status: http.StatusNotFound, wantErr: true},
		{name: "digest forbidden propagates", alias: "v2", failOnGet: 2,
			status: http.StatusForbidden, wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := registry.New(registry.Logger(log.New(io.Discard, "", 0)))
			var gets atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/manifests/"+tt.alias) {
					// The first GET loads the image; the second resolves its top-level digest.
					if n := gets.Add(1); tt.failOnGet > 0 && n >= tt.failOnGet {
						w.WriteHeader(tt.status)
						return
					}
				}
				reg.ServeHTTP(w, r)
			}))
			t.Cleanup(srv.Close)
			host := strings.TrimPrefix(srv.URL, "http://")
			img, err := mutate.ConfigFile(empty.Image, &v1.ConfigFile{
				Created: v1.Time{Time: time.Now().UTC()}, OS: "linux", Architecture: "amd64",
			})
			require.NoError(t, err)
			for _, tag := range []string{"v1", tt.alias, "v3"} {
				ref, err := name.NewTag(host + "/myimage:" + tag)
				require.NoError(t, err)
				require.NoError(t, remote.Write(ref, img))
			}
			filter, err := artif.BuildFilter(nil, "")
			require.NoError(t, err)
			got, err := New(nil, host, host).GetArtifactVersions(
				context.Background(), &minderv1.Artifact{Name: "myimage"}, filter)
			if tt.wantErr {
				require.ErrorContains(t, err, "unable to get digest for tag "+tt.alias)
				assert.Nil(t, got, "do not return a partial group after a digest failure")
			} else {
				require.NoError(t, err)
				if tt.wantTags == nil {
					assert.Empty(t, got)
				} else {
					require.Len(t, got, 1)
					assert.Equal(t, tt.wantTags, got[0].Tags)
					digest, err := img.Digest()
					require.NoError(t, err)
					assert.Equal(t, digest.String(), got[0].Sha)
				}
			}
			assert.GreaterOrEqual(t, gets.Load(), tt.failOnGet)
		})
	}
}
