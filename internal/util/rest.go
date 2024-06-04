//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package util provides helper functions for minder
package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	// CurlCmdMaxSize is the maximum size of the rendered curl command
	CurlCmdMaxSize = 2048
)

// HttpMethodFromString returns the HTTP method from a string based on upprecase inMeth, defaulting to dfl
func HttpMethodFromString(inMeth, dfl string) string {
	method := strings.ToUpper(inMeth)
	if len(method) == 0 {
		method = dfl
	}

	return method
}

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
