// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchHTTPFile(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/policy.yaml", r.URL.Path)
		assert.Equal(t, "raw=1", r.URL.RawQuery)
		_, _ = w.Write([]byte("remote content"))
	}))
	defer server.Close()

	reader, closer, err := fetchHTTPFile(context.Background(), server.URL+"/policy.yaml?raw=1", server.Client())
	require.NoError(t, err)
	require.NotNil(t, closer)
	t.Cleanup(func() { _ = closer.Close() })
	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, "remote content", string(content))
}

func TestFetchHTTPFileCanBeCanceled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("remote content"))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := fetchHTTPFile(ctx, server.URL+"/policy.yaml", server.Client())
	require.ErrorIs(t, err, context.Canceled)
}

func TestFetchHTTPFileRejectsStatusAndOversizedBody(t *testing.T) {
	t.Parallel()

	t.Run("non-success status", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("missing"))
		}))
		defer server.Close()

		_, _, err := fetchHTTPFile(context.Background(), server.URL+"/missing.yaml", server.Client())
		require.ErrorContains(t, err, "404")
	})

	t.Run("unknown length body over limit", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			_, _ = w.Write(make([]byte, maxHTTPFileSize+1))
		}))
		defer server.Close()

		reader, closer, err := fetchHTTPFile(context.Background(), server.URL+"/large.yaml", server.Client())
		require.NoError(t, err)
		defer closer.Close()
		_, err = io.ReadAll(reader)
		require.ErrorContains(t, err, "exceeds maximum size")
	})
}
