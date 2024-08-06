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
func (c *gitlabClient) NewRequest(method, requestUrl string, body any) (*http.Request, error) {
	u, err := url.JoinPath(c.glcfg.Endpoint, requestUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to join URL: %w", err)
	}

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
	req, err := http.NewRequest(method, u, buf)
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
