// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// manifestCounter wraps an in-memory registry, counting manifest requests by
// method. When stripHeadDigest is set, HEAD responses omit Docker-Content-Digest.
type manifestCounter struct {
	next            http.Handler
	stripHeadDigest bool
	heads, gets     atomic.Int32
	// tagGets counts GETs of the "v1" tag manifest, excluding other manifest
	// GETs such as the referrers tag-schema fallback.
	tagGets atomic.Int32
}

func (m *manifestCounter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.URL.Path, "/manifests/") {
		m.next.ServeHTTP(w, r)
		return
	}

	switch r.Method {
	case http.MethodHead:
		m.heads.Add(1)
		if m.stripHeadDigest {
			rec := httptest.NewRecorder()
			m.next.ServeHTTP(rec, r)
			for k, v := range rec.Header() {
				if http.CanonicalHeaderKey(k) == "Docker-Content-Digest" {
					continue
				}
				w.Header()[k] = v
			}
			w.WriteHeader(rec.Code)
			return
		}
	case http.MethodGet:
		m.gets.Add(1)
		if strings.HasSuffix(r.URL.Path, "/manifests/v1") {
			m.tagGets.Add(1)
		}
	}
	m.next.ServeHTTP(w, r)
}

func (m *manifestCounter) reset() {
	m.heads.Store(0)
	m.gets.Store(0)
	m.tagGets.Store(0)
}

// newCountingRegistry starts a registry holding testns/myimage:v1 and returns
// the counter, an OCI client for testns, and the image's digest.
func newCountingRegistry(t *testing.T, stripHeadDigest bool) (*manifestCounter, *OCI, string) {
	t.Helper()

	counter := &manifestCounter{
		next:            registry.New(registry.Logger(log.New(io.Discard, "", 0))),
		stripHeadDigest: stripHeadDigest,
	}
	srv := httptest.NewServer(counter)
	t.Cleanup(srv.Close)
	host := strings.TrimPrefix(srv.URL, "http://")

	img, err := random.Image(256, 1)
	require.NoError(t, err)
	ref, err := name.NewTag(fmt.Sprintf("%s/testns/myimage:v1", host))
	require.NoError(t, err)
	require.NoError(t, remote.Write(ref, img))
	dig, err := img.Digest()
	require.NoError(t, err)

	counter.reset()
	return counter, New(nil, host, path.Join(host, "testns")), dig.String()
}

func TestGetDigestUsesHead(t *testing.T) {
	t.Parallel()

	t.Run("existing tag resolves with HEAD only", func(t *testing.T) {
		t.Parallel()
		counter, o, want := newCountingRegistry(t, false)

		dig, err := o.GetDigest(context.Background(), "myimage", "v1")
		require.NoError(t, err)
		assert.Equal(t, want, dig)
		assert.Equal(t, int32(1), counter.heads.Load())
		assert.Equal(t, int32(0), counter.gets.Load(), "digest lookup must not GET the manifest")
	})

	t.Run("missing tag is not found without a GET", func(t *testing.T) {
		t.Parallel()
		counter, o, _ := newCountingRegistry(t, false)

		_, err := o.GetDigest(context.Background(), "myimage", "deleted")
		require.Error(t, err)
		assert.True(t, IsNotFound(err), "expected a 404, got %v", err)
		assert.Equal(t, int32(0), counter.gets.Load())
	})

	t.Run("falls back to GET when HEAD omits the digest header", func(t *testing.T) {
		t.Parallel()
		counter, o, want := newCountingRegistry(t, true)

		dig, err := o.GetDigest(context.Background(), "myimage", "v1")
		require.NoError(t, err)
		assert.Equal(t, want, dig)
		assert.Equal(t, int32(1), counter.heads.Load())
		assert.Equal(t, int32(1), counter.gets.Load())
	})

	t.Run("GetReferrer resolves digest with HEAD", func(t *testing.T) {
		t.Parallel()
		counter, o, _ := newCountingRegistry(t, false)

		_, err := o.GetReferrer(context.Background(), "myimage", "v1", "application/vnd.dev.sigstore.bundle+json")
		require.NoError(t, err)
		assert.Equal(t, int32(1), counter.heads.Load())
		assert.Equal(t, int32(0), counter.tagGets.Load(), "digest lookup must not GET the tag manifest")
	})
}
