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

const (
	// glPerPage is the page size requested from collection endpoints.
	// 100 is the maximum GitLab allows.
	glPerPage = 100

	// glNextPageHeader is the response header GitLab uses to point at the
	// next page of an offset-paginated collection. It is empty on the
	// last page.
	glNextPageHeader = "X-Next-Page"
)

// glRESTGetPaginated retrieves every page of a GitLab collection endpoint
// and appends the items to out. GitLab caps responses at per_page items
// (20 by default), so collection endpoints have to follow the X-Next-Page
// response header to retrieve the full result set.
func glRESTGetPaginated[T any](ctx context.Context, cli genericRESTClient, path string, out *[]T) error {
	for page := "1"; page != ""; {
		items, nextPage, err := glRESTGetPage[T](ctx, cli, path, page)
		if err != nil {
			return err
		}
		*out = append(*out, items...)

		// An empty page means there is nothing left to fetch regardless
		// of what the header says, which also guards against a server
		// that keeps reporting a next page.
		if len(items) == 0 {
			break
		}
		page = nextPage
	}

	return nil
}

// glRESTGetPage retrieves a single page of a GitLab collection endpoint and
// returns the items along with the next page reported by the server.
func glRESTGetPage[T any](ctx context.Context, cli genericRESTClient, path, page string) ([]T, string, error) {
	pagedPath, err := pathWithPagination(path, page)
	if err != nil {
		return nil, "", err
	}

	req, err := cli.NewRequest(http.MethodGet, pagedPath, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := cli.Do(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get resource '%s': %w", pagedPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, "", provifv1.ErrEntityNotFound
		}
		return nil, "", fmt.Errorf("failed to get resource '%s': %s", pagedPath, resp.Status)
	}

	var items []T
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, "", fmt.Errorf("failed to decode response: %w", err)
	}

	return items, resp.Header.Get(glNextPageHeader), nil
}

// pathWithPagination sets the page and per_page query parameters on path,
// preserving any query parameters it already carries.
func pathWithPagination(path, page string) (string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("failed to parse path '%s': %w", path, err)
	}

	q := u.Query()
	q.Set("page", page)
	q.Set("per_page", strconv.Itoa(glPerPage))
	u.RawQuery = q.Encode()

	return u.String(), nil
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
