// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
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
	u, err := getParsedURL(c.glcfg.Endpoint, requestPath)
	if err != nil {
		return nil, err
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
func restGet[T any](ctx context.Context, cli genericRESTClient, path string) (T, http.Header, error) {
	var out T

	// NewRequest already has the base URL configured, the path
	// will get appended to it.
	req, err := cli.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return out, nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := cli.Do(ctx, req)
	if err != nil {
		return out, nil, fmt.Errorf("failed to get resource '%s': %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return out, nil, provifv1.ErrEntityNotFound
		}
		return out, nil, fmt.Errorf("failed to get resource '%s': %s", path, resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return out, resp.Header, nil
}

const (
	// perPage is the page size requested from collection endpoints.
	// 100 is the maximum GitLab allows.
	perPage = 100

	// maxPages bounds a paginated fetch, so a very large or misbehaving
	// listing fails with an error instead of looping indefinitely.
	maxPages = 1000

	// nextPageHeader is the response header GitLab uses to point at the
	// next page of an offset-paginated collection.
	nextPageHeader = "X-Next-Page"
)

// restGetPaginated retrieves every page of a GitLab collection endpoint.
// GitLab caps responses at per_page items (20 by default), so collection
// endpoints have to follow the X-Next-Page response header to retrieve the
// full result set. The page and per_page query parameters are always set
// here, overwriting any values already in path; other query parameters are
// kept.
//
// If a page fails, the items fetched before it are returned along with the
// error, and the caller decides whether a partial result is usable.
func restGetPaginated[T any](ctx context.Context, cli genericRESTClient, path string) ([]T, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path '%s': %w", path, err)
	}
	query := u.Query()
	query.Set("per_page", strconv.Itoa(perPage))

	var all []T
	page := 1
	for range maxPages {
		query.Set("page", strconv.Itoa(page))
		u.RawQuery = query.Encode()

		items, header, err := restGet[[]T](ctx, cli, u.String())
		if err != nil {
			return all, err
		}
		all = append(all, items...)

		// A missing, malformed, or non-advancing next page ends the listing,
		// so the server cannot keep us on the same page.
		next, err := strconv.Atoi(header.Get(nextPageHeader))
		if len(items) == 0 || err != nil || next <= page {
			return all, nil
		}
		page = next
	}

	return all, fmt.Errorf("too many pages listing '%s' (limit %d)", path, maxPages)
}

func getParsedURL(endpoint, path string) (*url.URL, error) {
	base, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Explicitly parse path and query parameters. This is to ensure that
	// the path is properly escaped and that the query parameters are
	// properly encoded.
	parsedPathAndQuery, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path: %w", err)
	}

	u := base.JoinPath(parsedPathAndQuery.Path)

	// These have already been escaped by the URL parser
	u.RawQuery = parsedPathAndQuery.RawQuery
	u.Fragment = parsedPathAndQuery.Fragment

	return u, nil
}
