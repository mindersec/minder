// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
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
