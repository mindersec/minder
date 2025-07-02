// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package util provides helper functions for minder
package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
)

const (
	// CurlCmdMaxSize is the maximum size of the rendered curl command
	CurlCmdMaxSize = 2048
)

// GenerateCurlCommand generates a curl command from a method, apiBaseURL, endpoint, and body
// this is useful to provide a dry-run for remediations
func GenerateCurlCommand(ctx context.Context, method, apiBaseURL, endpoint, body string) (string, error) {
	if len(method) == 0 {
		return "", errors.New("method cannot be empty")
	}

	if len(apiBaseURL) == 0 {
		return "", errors.New("apiBaseURL cannot be empty")
	}

	// TODO: add support for headers
	// TODO: add toggle for token header
	tmplStr := `curl -L -X {{ .Method }} \
 -H "Accept: application/vnd.github+json" \
 -H "Authorization: Bearer $TOKEN" \
 -H "X-GitHub-Api-Version: 2022-11-28" \
 {{.URL}} \
 -d '{{.Body}}'`

	tmpl, err := NewSafeTextTemplate(&tmplStr, "curlCmd")
	if err != nil {
		return "", err
	}

	u, err := url.Parse(apiBaseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse endpoint: %w", err)
	}
	u = u.JoinPath(endpoint)

	var buf bytes.Buffer
	data := map[string]string{
		"Method": method,
		"URL":    u.String(),
		"Body":   body,
	}

	if err := tmpl.Execute(ctx, &buf, data, CurlCmdMaxSize); err != nil {
		return "", err
	}

	return buf.String(), nil
}
