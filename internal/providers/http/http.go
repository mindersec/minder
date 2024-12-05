// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package http implements an HTTP client for interacting with an HTTP API.
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers/telemetry"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// REST is the interface for interacting with an REST API.
// It implements helper functions that a provider that
// uses the `rest` trait can use.
type REST struct {
	baseURL    *url.URL
	cli        *http.Client
	credential provifv1.RestCredential
}

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
