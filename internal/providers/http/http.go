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

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/telemetry"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// REST is the interface for interacting with an REST API.
type REST struct {
	baseURL    *url.URL
	cli        *http.Client
	credential provifv1.RestCredential
}

// Ensure that REST implements the REST interface
var _ provifv1.REST = (*REST)(nil)

// NewREST creates a new RESTful client.
func NewREST(
	config *minderv1.RESTProviderConfig,
	metrics telemetry.HttpClientMetrics,
	credential provifv1.RestCredential,
) (*REST, error) {
	var cli *http.Client
	var err error

	cli = &http.Client{}

	cli.Transport, err = metrics.NewDurationRoundTripper(cli.Transport, db.ProviderTypeRest)
	if err != nil {
		return nil, fmt.Errorf("error creating duration round tripper: %w", err)
	}

	var baseURL *url.URL
	baseURL, err = baseURL.Parse(config.GetBaseUrl())
	if err != nil {
		return nil, err
	}

	return &REST{
		cli:        cli,
		baseURL:    baseURL,
		credential: credential,
	}, nil
}

// CanImplement returns true/false depending on whether the Provider
// can implement the specified trait
func (_ *REST) CanImplement(trait minderv1.ProviderType) bool {
	return trait == minderv1.ProviderType_PROVIDER_TYPE_REST
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

	h.credential.SetAuthorizationHeader(req)

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
