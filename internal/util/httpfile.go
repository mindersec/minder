// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	maxHTTPFileSize = 10 << 20 // 10 MiB
	httpFileTimeout = 30 * time.Second
)

// OpenFileArgContext opens a file argument and returns a descriptor, closer, and error.
// If the file is "-", it will return whatever is passed in as dashOpen and a no-op closer.
// HTTP(S) requests are canceled when ctx is canceled.
func OpenFileArgContext(ctx context.Context, f string, dashOpen io.Reader) (desc io.Reader, closer func(), err error) {
	if f == "-" {
		return dashOpen, func() {}, nil
	}
	if IsHTTPURL(f) {
		reader, bodyCloser, err := fetchHTTPFile(ctx, f, newHTTPFileClient())
		if err != nil {
			return nil, nil, fmt.Errorf("error fetching URL: %w", err)
		}
		return reader, func() {
			if err := bodyCloser.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "error closing URL response body: %v\n", err)
			}
		}, nil
	}

	f = filepath.Clean(f)
	file, err := os.Open(f)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening file: %w", err)
	}

	closer = func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing file: %v\n", err)
		}
	}
	return file, closer, nil
}

func newHTTPFileClient() *http.Client {
	// The default transport honors HTTP_PROXY, HTTPS_PROXY, and NO_PROXY.
	return &http.Client{Timeout: httpFileTimeout}
}

func fetchHTTPFile(ctx context.Context, rawURL string, client *http.Client) (io.Reader, io.Closer, error) {
	if client == nil {
		return nil, nil, fmt.Errorf("HTTP client is nil")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("creating URL request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, nil, err
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_ = response.Body.Close()
		return nil, nil, fmt.Errorf("unexpected HTTP status fetching URL: %s", response.Status)
	}
	if response.ContentLength > maxHTTPFileSize {
		_ = response.Body.Close()
		return nil, nil, fmt.Errorf("remote file exceeds maximum size of %d bytes", maxHTTPFileSize)
	}

	// Read at most one byte beyond the limit so unknown-length responses can
	// still be rejected instead of silently accepting a truncated file.
	reader := &maxHTTPFileReader{reader: io.LimitReader(response.Body, maxHTTPFileSize+1)}
	return reader, response.Body, nil
}

type maxHTTPFileReader struct {
	reader   io.Reader
	read     int64
	exceeded bool
}

func (r *maxHTTPFileReader) Read(p []byte) (int, error) {
	if r.exceeded {
		return 0, fmt.Errorf("remote file exceeds maximum size of %d bytes", maxHTTPFileSize)
	}

	n, err := r.reader.Read(p)
	r.read += int64(n)
	if r.read > maxHTTPFileSize {
		r.exceeded = true
		return n, fmt.Errorf("remote file exceeds maximum size of %d bytes", maxHTTPFileSize)
	}
	return n, err
}
