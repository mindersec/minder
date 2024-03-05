// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package http implements an HTTP client for interacting with an HTTP API.
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/telemetry"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// REST is the interface for interacting with an REST API.
type REST struct {
	baseURL *url.URL
	cli     *http.Client
	tok     string
}

// Ensure that REST implements the REST interface
var _ provifv1.REST = (*REST)(nil)

// NewREST creates a new RESTful client.
func NewREST(
	config *minderv1.RESTProviderConfig,
	metrics telemetry.HttpClientMetrics,
	tok string,
) (*REST, error) {
	var cli *http.Client
	var err error

	if tok != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: tok},
		)
		cli = oauth2.NewClient(context.Background(), ts)
	} else {
		cli = &http.Client{}
	}

	cli.Transport, err = metrics.NewDurationRoundTripper(cli.Transport, db.ProviderTraitRest)
	if err != nil {
		return nil, fmt.Errorf("error creating duration round tripper: %w", err)
	}

	var baseURL *url.URL
	baseURL, err = baseURL.Parse(config.GetBaseUrl())
	if err != nil {
		return nil, err
	}

	return &REST{
		cli:     cli,
		baseURL: baseURL,
		tok:     tok,
	}, nil
}

// GetToken returns the token for the provider
func (h *REST) GetToken() string {
	return h.tok
}

// GetBaseURL returns the base URL for the REST API.
func (h *REST) GetBaseURL() string {
	return h.baseURL.String()
}

// NewRequest creates an HTTP request.
func (h *REST) NewRequest(method, endpoint string, body any) (*http.Request, error) {
	targetURL := endpoint
	if h.baseURL != nil {
		u := h.baseURL.JoinPath(endpoint)
		targetURL = u.String()
	}

	reader, ok := body.(io.Reader)
	if !ok {
		return nil, fmt.Errorf("body is not an io.Reader")
	}
	return http.NewRequest(method, targetURL, reader)
}

// Do executes an HTTP request.
func (h *REST) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)

	return h.cli.Do(req)
}

// ParseV1Config parses the raw config into a HTTPConfig struct
func ParseV1Config(rawCfg json.RawMessage) (*minderv1.RESTProviderConfig, error) {
	type wrapper struct {
		REST *minderv1.RESTProviderConfig `json:"rest" validate:"required"`
	}

	var w wrapper
	if err := provifv1.ParseAndValidate(rawCfg, &w); err != nil {
		return nil, err
	}

	// Validate the config according to the protobuf validation rules.
	if err := w.REST.Validate(); err != nil {
		return nil, fmt.Errorf("error validating REST v1 provider config: %w", err)
	}

	return w.REST, nil
}
