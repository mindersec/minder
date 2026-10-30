// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bytes"
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
		content, err := fetchHTTPFile(ctx, f, newHTTPFileClient())
		if err != nil {
			return nil, nil, fmt.Errorf("error fetching URL: %w", err)
		}
		return bytes.NewReader(content), func() {}, nil
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

func fetchHTTPFile(ctx context.Context, rawURL string, client *http.Client) ([]byte, error) {
	if client == nil {
		return nil, fmt.Errorf("HTTP client is nil")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating URL request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected HTTP status fetching URL: %s", response.Status)
	}
	if response.ContentLength > maxHTTPFileSize {
		return nil, fmt.Errorf("remote file exceeds maximum size of %d bytes", maxHTTPFileSize)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, maxHTTPFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("reading URL response: %w", err)
	}
	if int64(len(content)) > maxHTTPFileSize {
		return nil, fmt.Errorf("remote file exceeds maximum size of %d bytes", maxHTTPFileSize)
	}
	return content, nil
}
