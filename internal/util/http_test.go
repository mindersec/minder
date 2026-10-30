// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestNewHTTPFileClientUsesEnvironmentProxy(t *testing.T) {
	t.Parallel()

	client := newHTTPFileClient()
	require.Nil(t, client.Transport)
	assert.Equal(t, 30*time.Second, client.Timeout)
	transport, ok := http.DefaultTransport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.Proxy)
	assert.Equal(t,
		reflect.ValueOf(http.ProxyFromEnvironment).Pointer(),
		reflect.ValueOf(transport.Proxy).Pointer(),
	)
}

func TestFetchHTTPFileWithClient(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "/policy.yaml", req.URL.Path)
		assert.Equal(t, "raw=1", req.URL.RawQuery)
		return testHTTPResponse(req, http.StatusOK, "remote content", nil), nil
	})}
	content, err := fetchHTTPFile(context.Background(), "https://example.com/policy.yaml?raw=1", client)
	require.NoError(t, err)
	assert.Equal(t, "remote content", string(content))
}

func TestFetchHTTPFileCanBeCanceled(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		close(started)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result := make(chan error, 1)
	go func() {
		_, err := fetchHTTPFile(ctx, "https://example.com/policy.yaml", client)
		result <- err
	}()
	<-started
	cancel()
	require.ErrorIs(t, <-result, context.Canceled)
}

func TestFetchHTTPFileRejectsStatusAndOversizedBody(t *testing.T) {
	t.Parallel()

	t.Run("non-success status", func(t *testing.T) {
		t.Parallel()
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return testHTTPResponse(req, http.StatusNotFound, "missing", nil), nil
		})}
		_, err := fetchHTTPFile(context.Background(), "https://example.com/missing.yaml", client)
		require.ErrorContains(t, err, "404")
	})

	t.Run("unknown length body over limit", func(t *testing.T) {
		t.Parallel()
		body := bytes.NewReader(make([]byte, maxHTTPFileSize+1))
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			response := testHTTPResponse(req, http.StatusOK, "", nil)
			response.Body = io.NopCloser(body)
			response.ContentLength = -1
			return response, nil
		})}
		_, err := fetchHTTPFile(context.Background(), "https://example.com/large.yaml", client)
		require.ErrorContains(t, err, "exceeds maximum size")
	})
}

func testHTTPResponse(req *http.Request, status int, body string, headers map[string]string) *http.Response {
	header := make(http.Header)
	for key, value := range headers {
		header.Set(key, value)
	}
	return &http.Response{
		StatusCode:    status,
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:        header,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Request:       req,
	}
}
