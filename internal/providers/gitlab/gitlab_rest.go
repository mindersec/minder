//
// Copyright 2024 Stacklok, Inc.
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

package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// Do implements the REST provider interface
func (c *gitlabClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	return c.cli.Do(req)
}

// GetBaseURL implements the REST provider interface
func (c *gitlabClient) GetBaseURL() string {
	return c.glcfg.Endpoint
}

// NewRequest implements the REST provider interface
func (c *gitlabClient) NewRequest(method, requestPath string, body any) (*http.Request, error) {
	base, err := url.Parse(c.glcfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	u := base.JoinPath(requestPath)

	var buf io.ReadWriter
	if body != nil {
		buf = &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		err := enc.Encode(body)
		if err != nil {
			return nil, err
		}
	}

	// TODO: Shall we try to use the GitLab client?
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if method == http.MethodPatch || method == http.MethodPost || method == http.MethodPut {
		req.Header.Set("Content-Type", "application/json")
	}

	req.Header.Set("Accept", "application/json")
	// TODO: Get User-Agent from constants
	req.Header.Set("User-Agent", "Minder")

	c.cred.SetAuthorizationHeader(req)

	return req, nil
}

type genericRESTClient interface {
	// Do sends an HTTP request and returns an HTTP response
	Do(ctx context.Context, req *http.Request) (*http.Response, error)
	NewRequest(method, requestUrl string, body any) (*http.Request, error)
}

// NOTE: We're not using github.com/xanzy/go-gitlab to do the actual
// request here because of the way they form authentication for requests.
// It would be ideal to use it, so we should consider contributing and making
// that part more pluggable.
func glRESTGet[T any](ctx context.Context, cli genericRESTClient, path string, out T) error {
	// NewRequest already has the base URL configured, the path
	// will get appended to it.
	req, err := cli.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := cli.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get resource '%s': %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return provifv1.ErrEntityNotFound
		}
		return fmt.Errorf("failed to get resource '%s': %s", path, resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}
