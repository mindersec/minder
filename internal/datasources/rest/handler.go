// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rest

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/rs/zerolog"
	"github.com/santhosh-tekuri/jsonschema/v6"
	uritemplate "github.com/std-uritemplate/std-uritemplate/go/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/engine/eval/rego"
	"github.com/mindersec/minder/internal/util/schemaupdate"
	"github.com/mindersec/minder/internal/util/schemavalidate"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/engine/v1/interfaces"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
)

const (
	// MaxBytesLimit is the maximum number of bytes to read from the response body
	// We limit to 1MB to prevent abuse
	MaxBytesLimit int64 = 1 << 20
)

var (
	metricsInit sync.Once

	dataSourceLatencyHistogram metric.Int64Histogram
)

type restHandler struct {
	rawInputSchema *structpb.Struct
	inputSchema    *jsonschema.Schema
	endpointTmpl   string
	method         string
	// used only to allow requests to localhost during tests
	testOnlyTransport http.RoundTripper
	// contains the request body or the key
	body          string
	bodyFromInput bool
	headers       map[string]string
	parse         string
	// TODO implement fallback
	// TODO implement auth
	provider interfaces.RESTProvider
}

func initMetrics() {
	metricsInit.Do(func() {
		meter := otel.Meter("minder")
		var err error
		dataSourceLatencyHistogram, err = meter.Int64Histogram(
			"datasource.rest.latency",
			metric.WithDescription("Latency of data source requests in milliseconds"),
			metric.WithUnit("ms"),
		)
		if err != nil {
			zerolog.Ctx(context.Background()).Warn().Err(err).Msg("Creating histogram for data source requests failed")
		}
	})
}

func newHandlerFromDef(def *minderv1.RestDataSource_Def, provider provinfv1.Provider) (*restHandler, error) {
	if def == nil {
		return nil, errors.New("rest data source handler definition is nil")
	}

	// schema may be nil
	schema, err := schemavalidate.CompileSchemaFromPB(def.GetInputSchema())
	if err != nil {
		return nil, err
	}

	bodyFromInput, body, err := parseRequestBodyConfig(def)
	if err != nil {
		return nil, err
	}

	initMetrics()

	// If this is not a RESTProvider, restProvider will be nil, which we already need to handle.
	restProvider, _ := interfaces.As[interfaces.RESTProvider](provider)

	return &restHandler{
		rawInputSchema: def.GetInputSchema(),
		inputSchema:    schema,
		endpointTmpl:   def.GetEndpoint(),
		method:         strings.ToUpper(cmp.Or(def.GetMethod(), http.MethodGet)),
		headers:        def.GetHeaders(),
		body:           body,
		bodyFromInput:  bodyFromInput,
		parse:          def.GetParse(),
		provider:       restProvider,
	}, nil
}

func (h *restHandler) GetArgsSchema() *structpb.Struct {
	return h.rawInputSchema
}

func (h *restHandler) ValidateArgs(args any) error {
	if h.inputSchema == nil {
		return errors.New("input schema cannot be nil")
	}

	mapobj, ok := args.(map[string]any)
	if !ok {
		return errors.New("args is not a map")
	}

	return schemavalidate.ValidateAgainstSchema(h.inputSchema, mapobj)
}

func (h *restHandler) ValidateUpdate(argsSchema *structpb.Struct) error {
	if argsSchema == nil {
		return errors.New("update schema cannot be nil")
	}

	if _, err := schemavalidate.CompileSchemaFromPB(argsSchema); err != nil {
		return fmt.Errorf("update validation failed due to invalid schema: %w", err)
	}
	return schemaupdate.ValidateSchemaUpdate(h.rawInputSchema, argsSchema)
}

func (h *restHandler) Call(ctx context.Context, _ *interfaces.Ingested, args any) (any, error) {
	argsMap, ok := args.(map[string]any)
	if !ok {
		return nil, errors.New("args is not a map")
	}

	expandedEndpoint, err := uritemplate.Expand(h.endpointTmpl, argsMap)
	if err != nil {
		return nil, err
	}

	transport := h.testOnlyTransport
	if transport == nil {
		transport = rego.LimitedDialer(nil)
	}
	// TODO: Add option to use custom client
	cli := &http.Client{
		// TODO: Make timeout configurable
		Timeout: 5 * time.Second,
		// Don't allow calling non-public addresses.
		Transport: transport,
	}

	b, bLen, err := h.getBody(argsMap)
	if err != nil {
		return nil, err
	}

	// Adapt slightly different calling patterns for Providers vs http.Client
	var req *http.Request
	var doer func(*http.Request) (*http.Response, error)
	if h.provider != nil && urlContains(h.provider.GetBaseURL(), expandedEndpoint) {
		// The RESTProvider NewRequest method inconsistently assumes either
		// parsed data (GitHub) or unparsed data (e.g. REST).  Explicitly set
		// body separately to avoid ambiguity.
		req, err = h.provider.NewRequest(h.method, expandedEndpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(b)
		req.ContentLength = int64(bLen)
		doer = func(req *http.Request) (*http.Response, error) {
			return h.provider.Do(req.Context(), req)
		}
	} else {
		req, err = http.NewRequest(h.method, expandedEndpoint, b)
		if err != nil {
			return nil, err
		}
		doer = cli.Do
	}
	req = req.WithContext(ctx)

	for k, v := range h.headers {
		req.Header.Add(k, v)
	}

	return h.doRequest(doer, req)
}

func recordMetrics(ctx context.Context, resp *http.Response, start time.Time) {
	attrs := []attribute.KeyValue{
		attribute.String("method", resp.Request.Method),
		attribute.String("endpoint", resp.Request.URL.String()),
		attribute.String("status_code", fmt.Sprintf("%d", resp.StatusCode)),
	}

	dataSourceLatencyHistogram.Record(ctx, time.Since(start).Milliseconds(), metric.WithAttributes(attrs...))
}

func (h *restHandler) doRequest(dofunc func(*http.Request) (*http.Response, error), req *http.Request) (any, error) {
	start := time.Now()
	resp, err := retriableDo(dofunc, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	recordMetrics(req.Context(), resp, start)

	bout, err := h.parseResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}

	// TODO: Handle fallback here.

	return buildRestOutput(resp.StatusCode, bout), nil
}

func (h *restHandler) getBody(args map[string]any) (io.Reader, int, error) {
	if h.bodyFromInput {
		return h.getBodyFromInput(args)
	}

	if h.body == "" {
		return nil, 0, nil
	}

	return strings.NewReader(h.body), len(h.body), nil
}

func (h *restHandler) getBodyFromInput(args map[string]any) (io.Reader, int, error) {
	if h.body == "" {
		return nil, 0, errors.New("body key is empty")
	}

	body, ok := args[h.body]
	if !ok {
		return nil, 0, fmt.Errorf("body key %q not found in args", h.body)
	}

	switch outb := body.(type) {
	case string:
		return strings.NewReader(outb), len(outb), nil
	case map[string]any:
		// stringify the object
		serialized, err := json.Marshal(outb)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot marshal body object: %w", err)
		}

		return bytes.NewBuffer(serialized), len(serialized), nil
	default:
		return nil, 0, fmt.Errorf("body key %q is not a string or object", h.body)
	}
}

func (h *restHandler) parseResponseBody(body io.Reader) (any, error) {
	var data any

	if body == nil {
		return nil, nil
	}

	lr := io.LimitReader(body, MaxBytesLimit)

	if h.parse == "json" {
		var jsonData any
		dec := json.NewDecoder(lr)
		if err := dec.Decode(&jsonData); err != nil {
			return nil, fmt.Errorf("cannot decode json: %w", err)
		}

		data = jsonData
	} else {
		bytedata, err := io.ReadAll(lr)
		if err != nil {
			return nil, fmt.Errorf("cannot read response body: %w", err)
		}

		data = string(bytedata)
	}

	return data, nil
}

func parseRequestBodyConfig(def *minderv1.RestDataSource_Def) (bool, string, error) {
	defBody := def.GetBody()
	if defBody == nil {
		return false, "", nil
	}

	switch defBody.(type) {
	case *minderv1.RestDataSource_Def_Bodyobj:
		obj, err := json.Marshal(def.GetBodyobj())
		// Since BodyObj is a proto struct, this should never error
		if err != nil {
			return false, "", err
		}

		return false, string(obj), nil
	case *minderv1.RestDataSource_Def_BodyFromField:
		if def.GetBodyFromField() == "" {
			return true, "", fmt.Errorf("body_from_field is empty")
		}
		return true, def.GetBodyFromField(), nil
	}

	return false, def.GetBodystr(), nil
}

func buildRestOutput(statusCode int, body any) any {
	return map[string]any{
		"status_code": statusCode,
		"body":        body,
	}
}

func retriableDo(dofunc func(*http.Request) (*http.Response, error), req *http.Request) (*http.Response, error) {
	var resp *http.Response
	retryCount := 0

	err := backoff.Retry(func() error {
		var err error
		resp, err = dofunc(req)
		if err != nil {
			zerolog.Ctx(req.Context()).Debug().
				Err(err).
				Int("retry", retryCount).
				Msg("HTTP request failed, retrying")
			retryCount++
			return err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			zerolog.Ctx(req.Context()).Debug().
				Int("retry", retryCount).
				Msg("rate limited, retrying")
			retryCount++
			return errors.New("rate limited")
		}

		return nil
	}, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 3))
	if err != nil {
		zerolog.Ctx(req.Context()).Warn().
			Err(err).
			Int("retries", retryCount).
			Msg("HTTP request failed after retries")
		return nil, err
	}

	return resp, nil
}

func urlContains(base, endpoint string) bool {
	baseURL, err := url.Parse(base)
	if err != nil {
		return false
	}

	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return false
	}

	// Normalize paths to have a trailing slash for prefix comparison
	basePath := strings.TrimSuffix(baseURL.Path, "/") + "/"
	endpointPath := strings.TrimSuffix(endpointURL.Path, "/") + "/"

	return baseURL.Scheme == endpointURL.Scheme &&
		baseURL.Host == endpointURL.Host &&
		strings.HasPrefix(endpointPath, basePath)
}
